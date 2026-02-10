package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	"github.com/Cloud-Foundations/Dominator/fleetmanager/httpd"
	"github.com/Cloud-Foundations/Dominator/fleetmanager/hypervisors"
	"github.com/Cloud-Foundations/Dominator/fleetmanager/hypervisors/fsstorer"
	"github.com/Cloud-Foundations/Dominator/fleetmanager/rpcd"
	"github.com/Cloud-Foundations/Dominator/fleetmanager/topology"
	"github.com/Cloud-Foundations/Dominator/lib/constants"
	"github.com/Cloud-Foundations/Dominator/lib/flags/loadflags"
	"github.com/Cloud-Foundations/Dominator/lib/fsutil"
	libgrpc "github.com/Cloud-Foundations/Dominator/lib/grpc"
	"github.com/Cloud-Foundations/Dominator/lib/json"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/log/serverlogger"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/lib/srpc/proxy"
	"github.com/Cloud-Foundations/Dominator/lib/srpc/setupserver"
	pb "github.com/Cloud-Foundations/Dominator/proto/fleetmanager/grpc"
	"github.com/Cloud-Foundations/tricorder/go/tricorder"
)

var (
	checkTopology = flag.Bool("checkTopology", false,
		"If true, perform a one-time check, write to stdout and exit")
	grpcPortNum = flag.Uint("grpcPortNum", constants.FleetManagerPortNumber-100,
		"Port number to listen on for gRPC (0 = disabled)")
	restPortNum = flag.Uint("restPortNum", 0,
		"Port number to listen on for REST API via grpc-gateway (0 = disabled)")
	ipmiPasswordFile = flag.String("ipmiPasswordFile", "",
		"Name of password file used to authenticate for IPMI requests")
	ipmiUsername = flag.String("ipmiUsername", "",
		"Name of user to authenticate as when making IPMI requests")
	topologyCheckInterval = flag.Duration("topologyCheckInterval",
		time.Minute, "Configuration check interval")
	portNum = flag.Uint("portNum", constants.FleetManagerPortNumber,
		"Port number to allocate and listen on for HTTP/RPC")
	stateDir = flag.String("stateDir", "/var/lib/fleet-manager",
		"Name of state directory")
	topologyDir = flag.String("topologyDir", "",
		"Name of local topology directory or directory in Git repository")
	topologyRepository = flag.String("topologyRepository", "",
		"URL of Git repository containing repository")
	variablesDir = flag.String("variablesDir", "",
		"Name of local variables directory or directory in Git repository")
)

func doCheck(logger log.DebugLogger) {
	topo, err := topology.LoadWithParams(topology.Params{
		Logger:       logger,
		TopologyDir:  *topologyDir,
		VariablesDir: *variablesDir,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := json.WriteWithIndent(os.Stdout, "    ", topo); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

// internalGrpcPort is used for the REST gateway to connect to gRPC without TLS
var internalGrpcServer *grpc.Server

func startGrpcServer(manager *hypervisors.Manager, port uint, restPort uint,
	logger log.DebugLogger) error {
	// Get TLS config from SRPC - reuse the same certificates
	tlsConfig := srpc.GetServerTlsConfig()
	if tlsConfig == nil {
		return fmt.Errorf("no TLS config available for gRPC server")
	}

	// Create gRPC server with TLS and auth interceptors
	creds := credentials.NewTLS(tlsConfig)
	server := grpc.NewServer(
		grpc.Creds(creds),
		grpc.UnaryInterceptor(libgrpc.UnaryAuthInterceptor),
		grpc.StreamInterceptor(libgrpc.StreamAuthInterceptor),
	)

	// Register FleetManager service
	rpcd.SetupGRPC(server, manager, logger)

	// Register reflection service for grpcurl and other tools
	reflection.Register(server)

	// Start listening
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", port, err)
	}

	logger.Printf("Starting gRPC server on port %d\n", port)

	// Serve in goroutine so we don't block
	go func() {
		if err := server.Serve(lis); err != nil {
			logger.Fatalf("gRPC server failed: %s\n", err)
		}
	}()

	// If REST gateway is enabled, also start an internal gRPC server without TLS
	// This allows the gateway to connect without needing client certificates
	if restPort > 0 {
		internalGrpcServer = grpc.NewServer()
		rpcd.SetupGRPC(internalGrpcServer, manager, logger)

		// Listen on localhost only (not exposed externally)
		internalPort := port + 1000 // Use port+1000 for internal
		internalLis, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", internalPort))
		if err != nil {
			return fmt.Errorf("failed to listen on internal port %d: %w", internalPort, err)
		}

		logger.Printf("Starting internal gRPC server on 127.0.0.1:%d for REST gateway\n", internalPort)

		go func() {
			if err := internalGrpcServer.Serve(internalLis); err != nil {
				logger.Fatalf("Internal gRPC server failed: %s\n", err)
			}
		}()
	}

	return nil
}

func startRestGateway(grpcPort, restPort uint, logger log.DebugLogger) error {
	ctx := context.Background()

	// Connect to the internal gRPC server (localhost only, no TLS)
	internalPort := grpcPort + 1000
	grpcServerEndpoint := fmt.Sprintf("127.0.0.1:%d", internalPort)

	// Create gateway mux
	mux := runtime.NewServeMux()

	// Register the FleetManager handler
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if err := pb.RegisterFleetManagerHandlerFromEndpoint(ctx, mux, grpcServerEndpoint, opts); err != nil {
		return fmt.Errorf("failed to register gateway handler: %w", err)
	}

	// Get TLS config from SRPC - reuse the same certificates and CA
	// This enables mTLS: server presents cert, client must present valid cert
	tlsConfig := srpc.GetServerTlsConfig()
	if tlsConfig == nil {
		return fmt.Errorf("no TLS config available for REST gateway")
	}

	// Clone the config to avoid modifying the shared one
	restTlsConfig := tlsConfig.Clone()
	// Require client certificates (mTLS)
	restTlsConfig.ClientAuth = tls.RequireAndVerifyClientCert

	// Create HTTPS server with mTLS
	server := &http.Server{
		Addr:      fmt.Sprintf(":%d", restPort),
		Handler:   mux,
		TLSConfig: restTlsConfig,
	}

	logger.Printf("Starting REST gateway on port %d (HTTPS with mTLS)\n", restPort)

	go func() {
		// TLSConfig is already set, so ListenAndServeTLS uses it
		// Empty cert/key paths because they're in TLSConfig
		if err := server.ListenAndServeTLS("", ""); err != nil {
			logger.Fatalf("REST gateway failed: %s\n", err)
		}
	}()

	return nil
}

func main() {
	if err := loadflags.LoadForDaemon("fleet-manager"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	flag.Parse()
	tricorder.RegisterFlags()
	logger := serverlogger.New("")
	srpc.SetDefaultLogger(logger)
	if *checkTopology {
		doCheck(logger)
	}
	params := setupserver.Params{Logger: logger}
	if err := setupserver.SetupTlsWithParams(params); err != nil {
		logger.Fatalln(err)
	}
	if err := proxy.New(logger); err != nil {
		logger.Fatalln(err)
	}
	if err := os.MkdirAll(*stateDir, fsutil.DirPerms); err != nil {
		logger.Fatalf("Cannot create state directory: %s\n", err)
	}
	topologyChannel, err := topology.WatchWithParams(topology.WatchParams{
		Params: topology.Params{
			Logger:       logger,
			TopologyDir:  *topologyDir,
			VariablesDir: *variablesDir,
		},
		CheckInterval:      *topologyCheckInterval,
		LocalRepositoryDir: filepath.Join(*stateDir, "topology"),
		TopologyRepository: *topologyRepository,
	},
	)
	if err != nil {
		logger.Fatalf("Cannot watch for topology: %s\n", err)
	}
	storer, err := fsstorer.New(filepath.Join(*stateDir, "hypervisor-db"),
		logger)
	if err != nil {
		logger.Fatalf("Cannot create DB: %s\n", err)
	}
	hyperManager, err := hypervisors.New(hypervisors.StartOptions{
		IpmiPasswordFile: *ipmiPasswordFile,
		IpmiUsername:     *ipmiUsername,
		Logger:           logger,
		Storer:           storer,
	})
	if err != nil {
		logger.Fatalf("Cannot create hypervisors manager: %s\n", err)
	}
	rpcHtmlWriter, err := rpcd.Setup(hyperManager, logger)
	if err != nil {
		logger.Fatalf("Cannot start rpcd: %s\n", err)
	}
	webServer, err := httpd.StartServer(*portNum, logger)
	if err != nil {
		logger.Fatalf("Unable to create http server: %s\n", err)
	}
	webServer.AddHtmlWriter(hyperManager)
	webServer.AddHtmlWriter(rpcHtmlWriter)
	webServer.AddHtmlWriter(logger)
	// Start gRPC server if enabled
	if *grpcPortNum > 0 {
		if err := startGrpcServer(hyperManager, *grpcPortNum, *restPortNum, logger); err != nil {
			logger.Fatalf("Cannot start gRPC server: %s\n", err)
		}
		// Start REST gateway if enabled (requires gRPC to be enabled)
		if *restPortNum > 0 {
			if err := startRestGateway(*grpcPortNum, *restPortNum, logger); err != nil {
				logger.Fatalf("Cannot start REST gateway: %s\n", err)
			}
		}
	} else if *restPortNum > 0 {
		logger.Fatalln("REST gateway requires gRPC to be enabled (-grpcPortNum)")
	}
	for topology := range topologyChannel {
		logger.Println("Received new topology")
		webServer.UpdateTopology(topology)
		hyperManager.UpdateTopology(topology)
	}
}
