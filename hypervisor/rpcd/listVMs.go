package rpcd

import (
	"context"
	"net"

	"github.com/Cloud-Foundations/Dominator/lib/errors"
	lib_grpc "github.com/Cloud-Foundations/Dominator/lib/grpc"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
	pb "github.com/Cloud-Foundations/Dominator/proto/hypervisor/grpc"
)

func (t *srpcType) ListVMs(conn *srpc.Conn,
	request hypervisor.ListVMsRequest,
	reply *hypervisor.ListVMsResponse) error {
	ipAddressStrings := t.manager.ListVMs(request)
	ipAddresses := make([]net.IP, 0, len(ipAddressStrings))
	for _, ipAddressString := range ipAddressStrings {
		ipAddress := net.ParseIP(ipAddressString)
		if shrunkIP := ipAddress.To4(); shrunkIP != nil {
			ipAddress = shrunkIP
		}
		ipAddresses = append(ipAddresses, ipAddress)
	}
	*reply = hypervisor.ListVMsResponse{IpAddresses: ipAddresses}
	return nil
}

// ListVms gRPC handler
func (s *grpcServer) ListVms(ctx context.Context, req *pb.ListVmsRequest) (*pb.ListVmsResponse, error) {
	username := "unknown"
	if conn := lib_grpc.ConnFromContext(ctx); conn != nil {
		if authInfo := conn.GetAuthInformation(); authInfo != nil {
			username = authInfo.Username
		}
	}

	s.logger.Debugf(1, "ListVms(%s) called\n", username)

	// Check if FULL view is requested - return UNIMPLEMENTED for now
	if req.View == pb.VmView_VM_VIEW_FULL {
		return nil, errors.NewUnimplementedError("VM_VIEW_FULL for ListVms - use GetVm for full details")
	}

	// Call manager's ListVMs with empty request (list all)
	ipStrings := s.manager.ListVMs(hypervisor.ListVMsRequest{})

	// Convert IP strings to Vm objects with BASIC view
	vms := make([]*pb.Vm, 0, len(ipStrings))
	for _, ipStr := range ipStrings {
		vms = append(vms, &pb.Vm{
			IpAddress: ipStr,
			// Note: For BASIC view, we only have IP address from ListVMs
			// hostname and state would require GetVmInfo calls (N+1 problem)
		})
	}

	return &pb.ListVmsResponse{
		Vms: vms,
	}, nil
}
