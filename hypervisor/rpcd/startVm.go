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

func (t *srpcType) StartVm(conn *srpc.Conn,
	request hypervisor.StartVmRequest,
	reply *hypervisor.StartVmResponse) error {
	dhcpTimedOut, err := t.manager.StartVm(request.IpAddress,
		conn.GetAuthInformation(), request.AccessToken, request.DhcpTimeout)
	response := hypervisor.StartVmResponse{dhcpTimedOut,
		errors.ErrorToString(err)}
	*reply = response
	return nil
}

// StartVm gRPC handler
func (s *grpcServer) StartVm(ctx context.Context, req *pb.StartVmRequest) (*pb.Vm, error) {
	conn := lib_grpc.ConnFromContext(ctx)
	if conn == nil {
		return nil, errors.NewUnauthenticatedError("no connection in context")
	}
	authInfo := conn.GetAuthInformation()
	if authInfo == nil {
		return nil, errors.NewUnauthenticatedError("no auth information")
	}

	ipAddr := net.ParseIP(req.IpAddress)
	s.logger.Debugf(1, "StartVm(%s) called for IP=%s\n", authInfo.Username, req.IpAddress)

	_, err := s.manager.StartVm(ipAddr, authInfo, nil, 0)
	if err != nil {
		s.logger.Debugf(1, "StartVm(%s) failed: %v\n", authInfo.Username, err)
		return nil, lib_grpc.ErrorToStatus(err)
	}

	// Get updated VM info to return
	vmInfo, err := s.manager.GetVmInfo(ipAddr)
	if err != nil {
		// VM was started but we couldn't get updated info - return minimal response
		return &pb.Vm{IpAddress: req.IpAddress}, nil
	}

	return vmInfoToProto(&vmInfo, pb.VmView_VM_VIEW_FULL), nil
}
