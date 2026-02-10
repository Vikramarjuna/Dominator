package rpcd

import (
	"io"

	"google.golang.org/grpc"

	"github.com/Cloud-Foundations/Dominator/fleetmanager/hypervisors"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/lib/srpc/serverutil"
	pb "github.com/Cloud-Foundations/Dominator/proto/fleetmanager/grpc"
)

type srpcType struct {
	hypervisorsManager *hypervisors.Manager
	logger             log.DebugLogger
	*serverutil.PerUserMethodLimiter
}

// grpcServer implements the FleetManager gRPC service.
type grpcServer struct {
	pb.UnimplementedFleetManagerServer
	hypervisorsManager *hypervisors.Manager
	logger             log.DebugLogger
}

type htmlWriter srpcType

func (hw *htmlWriter) WriteHtml(writer io.Writer) {
	hw.writeHtml(writer)
}

// Setup registers the FleetManager SRPC service.
func Setup(hypervisorsManager *hypervisors.Manager, logger log.DebugLogger) (
	*htmlWriter, error) {
	srpcObj := &srpcType{
		hypervisorsManager: hypervisorsManager,
		logger:             logger,
		PerUserMethodLimiter: serverutil.NewPerUserMethodLimiter(
			map[string]uint{
				"GetMachineInfo": 1,
				"GetUpdates":     1,
			}),
	}
	srpc.RegisterNameWithOptions("FleetManager", srpcObj,
		srpc.ReceiverOptions{
			PublicMethods: []string{
				"ChangeMachineTags",
				"GetHypervisorForVM",
				"GetHypervisorsInLocation",
				"GetIpInfo",
				"GetMachineInfo",
				"GetUpdates",
				"ListHypervisorLocations",
				"ListHypervisorsInLocation",
				"ListVMsInLocation",
				"PowerOnMachine",
			}})
	return (*htmlWriter)(srpcObj), nil
}

// SetupGRPC registers the FleetManager gRPC service with the given gRPC server.
func SetupGRPC(server *grpc.Server, manager *hypervisors.Manager,
	logger log.DebugLogger) {

	pb.RegisterFleetManagerServer(server, &grpcServer{
		hypervisorsManager: manager,
		logger:             logger,
	})
}
