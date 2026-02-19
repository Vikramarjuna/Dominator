package server

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"golang.org/x/net/http2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"

	lib_grpc "github.com/Cloud-Foundations/Dominator/lib/grpc"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

type Config struct {
	Port        uint
	Logger      log.DebugLogger
	HttpHandler http.Handler // Required. Serves SRPC and HTML.
	GrpcHandler func(grpcServer *grpc.Server, gatewayMux *runtime.ServeMux) (http.Handler, error)
}

// Start starts a server. Routes by content-type and path. Blocks.
// Uses protocol detection: TLS connections get TLS handling, plain HTTP passes
// through for SRPC's own TLS upgrade mechanism.
func Start(config Config) error {
	if config.HttpHandler == nil {
		return fmt.Errorf("HttpHandler is required")
	}
	tlsConfig := srpc.GetServerTlsConfig()
	if tlsConfig == nil {
		return fmt.Errorf("TLS config not available")
	}
	if config.GrpcHandler != nil {
		tlsConfig.ClientAuth = tls.VerifyClientCertIfGiven
		tlsConfig.NextProtos = append(tlsConfig.NextProtos, "h2", "http/1.1")
	}
	tcpListener, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Port))
	if err != nil {
		return fmt.Errorf("cannot create listener: %w", err)
	}

	handler, err := buildHandler(config)
	if err != nil {
		return err
	}

	server := &http.Server{Handler: handler}
	if err := http2.ConfigureServer(server, &http2.Server{}); err != nil {
		return fmt.Errorf("failed to configure HTTP/2: %w", err)
	}

	if config.Logger != nil {
		if config.GrpcHandler != nil {
			config.Logger.Printf("Started server on port %d (gRPC+REST+HTTP)\n", config.Port)
		} else {
			config.Logger.Printf("Started HTTP server on port %d\n", config.Port)
		}
	}

	// Use protocol-detecting listener: TLS clients get TLS, plain HTTP
	// passes through for SRPC's HTTP CONNECT + TLS upgrade mechanism.
	protoListener := &protocolDetectingListener{
		Listener:  tcpListener,
		tlsConfig: tlsConfig,
	}
	return server.Serve(protoListener)
}

func buildHandler(config Config) (http.Handler, error) {
	if config.GrpcHandler == nil {
		return config.HttpHandler, nil
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(lib_grpc.UnaryAuthInterceptor),
		grpc.StreamInterceptor(lib_grpc.StreamAuthInterceptor),
	)
	reflection.Register(grpcServer)

	gatewayMux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				EmitUnpopulated: false,
			},
			UnmarshalOptions: protojson.UnmarshalOptions{
				DiscardUnknown: true,
			},
		}),
	)
	restHandler, err := config.GrpcHandler(grpcServer, gatewayMux)
	if err != nil {
		return nil, fmt.Errorf("failed to register services: %w", err)
	}
	if restHandler == nil {
		restHandler = gatewayMux
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc"):
			grpcServer.ServeHTTP(w, r)
		case isRestApiPath(r.URL.Path):
			restHandler.ServeHTTP(w, r)
		case isSrpcPath(r.URL.Path):
			// SRPC handlers are registered on http.DefaultServeMux
			http.DefaultServeMux.ServeHTTP(w, r)
		default:
			config.HttpHandler.ServeHTTP(w, r)
		}
	}), nil
}

func isRestApiPath(path string) bool {
	return strings.HasPrefix(path, "/v1/")
}

func isSrpcPath(path string) bool {
	return strings.HasPrefix(path, "/_goSRPC_/") ||
		strings.HasPrefix(path, "/_go_TLS_SRPC_/") ||
		strings.HasPrefix(path, "/_SRPC_/")
}

// protocolDetectingListener wraps a net.Listener and detects TLS vs plain HTTP.
type protocolDetectingListener struct {
	net.Listener
	tlsConfig *tls.Config
}

func (l *protocolDetectingListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	// Extract TCP connection if available.
	var tcpConn *net.TCPConn
	if tc, ok := conn.(*net.TCPConn); ok {
		tcpConn = tc
	}
	// Wrap connection with buffered reader for peeking.
	bufferedConn := &bufferedConn{
		Conn:    conn,
		reader:  bufio.NewReader(conn),
		tcpConn: tcpConn,
	}
	// Peek at first byte to detect TLS.
	firstByte, err := bufferedConn.reader.Peek(1)
	if err != nil {
		conn.Close()
		return nil, err
	}
	// TLS ClientHello starts with 0x16 (handshake record type).
	if firstByte[0] == 0x16 {
		return tls.Server(bufferedConn, l.tlsConfig), nil
	}
	// Plain HTTP - return as-is for SRPC's HTTP CONNECT + TLS upgrade.
	return bufferedConn, nil
}

// bufferedConn wraps a net.Conn with a buffered reader for peeking.
// Implements TCP-specific methods if the underlying connection is TCP.
type bufferedConn struct {
	net.Conn
	reader  *bufio.Reader
	tcpConn *net.TCPConn // nil if not TCP
}

func (c *bufferedConn) Read(b []byte) (int, error) {
	return c.reader.Read(b)
}

func (c *bufferedConn) SetKeepAlive(keepalive bool) error {
	if c.tcpConn != nil {
		return c.tcpConn.SetKeepAlive(keepalive)
	}
	return nil
}

func (c *bufferedConn) SetKeepAlivePeriod(d time.Duration) error {
	if c.tcpConn != nil {
		return c.tcpConn.SetKeepAlivePeriod(d)
	}
	return nil
}
