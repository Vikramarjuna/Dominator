package rpcd

import (
	"io"
	"net"
	"sync"

	"github.com/Cloud-Foundations/Dominator/hypervisor/manager"
	lib_grpc "github.com/Cloud-Foundations/Dominator/lib/grpc"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/hypervisor"
	pb "github.com/Cloud-Foundations/Dominator/proto/hypervisor/grpc"
	"google.golang.org/grpc"
)

// methodOptions defines authorization levels for Hypervisor methods.
// Used by both SRPC and gRPC registration.
var methodOptions = srpc.ReceiverOptions{
	PublicMethods: []string{
		"AcknowledgeVm",
		"AddVmVolumes",
		"BecomePrimaryVmOwner",
		"ChangeVmConsoleType",
		"ChangeVmCpuPriority",
		"ChangeVmDestroyProtection",
		"ChangeVmHostname",
		"ChangeVmMachineType",
		"ChangeVmNumNetworkQueues",
		"ChangeVmOwnerGroups",
		"ChangeVmOwnerUsers",
		"ChangeVmSize",
		"ChangeVmSubnet",
		"ChangeVmTags",
		"ChangeVmVolumeInterfaces",
		"ChangeVmVolumeSize",
		"ChangeVmVolumeStorageIndex",
		"CommitImportedVm",
		"ConnectToVmConsole",
		"ConnectToVmSerialPort",
		"CopyVm",
		"CreateVm",
		"DebugVmImage",
		"DeleteVmVolume",
		"DestroyVm",
		"DiscardVmAccessToken",
		"DiscardVmOldImage",
		"DiscardVmOldUserData",
		"DiscardVmSnapshot",
		"ExportLocalVm",
		"GetCapacity",
		"GetIdentityProvider",
		"GetPublicKey",
		"GetRootCookiePath",
		"GetUpdates",
		"GetVmAccessToken",
		"GetVmCreateRequest",
		"GetVmInfo",
		"GetVmInfos",
		"GetVmLastPatchLog",
		"GetVmUserData",
		"GetVmVolume",
		"GetVmVolumeStorageConfiguration",
		"ImportLocalVm",
		"ListSubnets",
		"ListVMs",
		"ListVolumeDirectories",
		"MigrateVm",
		"PatchVmImage",
		"ProbeVmPort",
		"RebootVm",
		"ReplaceVmCredentials",
		"ReplaceVmIdentity",
		"ReplaceVmImage",
		"ReplaceVmUserData",
		"RestoreVmFromSnapshot",
		"RestoreVmImage",
		"RestoreVmUserData",
		"ReorderVmVolumes",
		"ScanVmRoot",
		"SnapshotVm",
		"StartVm",
		"StopVm",
		"TraceVmMetadata",
	},
}

type DhcpServer interface {
	AddLease(address proto.Address, hostname string) error
	AddNetbootLease(address proto.Address, hostname string,
		subnet *proto.Subnet) error
	ClosePacketWatchChannel(channel <-chan proto.WatchDhcpResponse)
	MakeAcknowledgmentChannel(ipAddr net.IP) <-chan struct{}
	MakePacketWatchChannel() <-chan proto.WatchDhcpResponse
	RemoveLease(ipAddr net.IP)
}

type ipv4Address [4]byte

type srpcType struct {
	dhcpServer           DhcpServer
	logger               log.DebugLogger
	manager              *manager.Manager
	tftpbootServer       TftpbootServer
	mutex                sync.Mutex             // Protect everything below.
	externalLeases       map[ipv4Address]string // Value: MAC address.
	manageExternalLeases bool
}

type TftpbootServer interface {
	RegisterFiles(ipAddr net.IP, files map[string][]byte)
	UnregisterFiles(ipAddr net.IP)
}

type htmlWriter srpcType

func (hw *htmlWriter) WriteHtml(writer io.Writer) {
	hw.writeHtml(writer)
}

// grpcServer implements the gRPC HypervisorServer interface
type grpcServer struct {
	dhcpServer     DhcpServer
	logger         log.DebugLogger
	manager        *manager.Manager
	tftpbootServer TftpbootServer
}

func Setup(manager *manager.Manager, dhcpServer DhcpServer,
	tftpbootServer TftpbootServer, logger log.DebugLogger) (*htmlWriter, error) {
	srpcObj := &srpcType{
		dhcpServer:     dhcpServer,
		logger:         logger,
		manager:        manager,
		tftpbootServer: tftpbootServer,
		externalLeases: make(map[ipv4Address]string),
	}
	srpc.SetDefaultGrantMethod(
		func(_ string, authInfo *srpc.AuthInformation) bool {
			return manager.CheckOwnership(authInfo)
		})
	srpc.RegisterNameWithOptions("Hypervisor", srpcObj, methodOptions)
	return (*htmlWriter)(srpcObj), nil
}

// grpcToSrpcMethodMapping maps gRPC method names to SRPC equivalents.
var grpcToSrpcMethodMapping = map[string]string{
	"CreateVmStreaming": "CreateVm",
	"GetVm":             "GetVmInfo",
	"ListVms":           "ListVMs",
}

// gRPC-only methods (no SRPC equivalent).
var (
	grpcOnlyPublicMethods          = []string{"CreateVm"} // Async/unary
	grpcOnlyUnauthenticatedMethods []string
)

func SetupGRPC(server *grpc.Server, manager *manager.Manager,
	dhcpServer DhcpServer, tftpbootServer TftpbootServer,
	logger log.DebugLogger) pb.HypervisorServer {
	lib_grpc.RegisterServiceOptions("hypervisor.Hypervisor", lib_grpc.ServiceOptions{
		PublicMethods:                  methodOptions.PublicMethods,
		UnauthenticatedMethods:         methodOptions.UnauthenticatedMethods,
		GrpcToSrpcMethods:              grpcToSrpcMethodMapping,
		GrpcOnlyPublicMethods:          grpcOnlyPublicMethods,
		GrpcOnlyUnauthenticatedMethods: grpcOnlyUnauthenticatedMethods,
	})
	srv := &grpcServer{
		dhcpServer:     dhcpServer,
		logger:         logger,
		manager:        manager,
		tftpbootServer: tftpbootServer,
	}
	pb.RegisterHypervisorServer(server, srv)
	return srv
}
