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

func (t *srpcType) GetVmInfo(conn *srpc.Conn,
	request hypervisor.GetVmInfoRequest,
	reply *hypervisor.GetVmInfoResponse) error {
	info, err := t.manager.GetVmInfo(request.IpAddress)
	response := hypervisor.GetVmInfoResponse{
		VmInfo: info,
		Error:  errors.ErrorToString(err),
	}
	*reply = response
	return nil
}

// GetVm gRPC handler
func (s *grpcServer) GetVm(ctx context.Context, req *pb.GetVmRequest) (*pb.Vm, error) {
	username := "unknown"
	if conn := lib_grpc.ConnFromContext(ctx); conn != nil {
		if authInfo := conn.GetAuthInformation(); authInfo != nil {
			username = authInfo.Username
		}
	}

	ipAddr := net.ParseIP(req.IpAddress)
	s.logger.Debugf(1, "GetVm(%s) called for IP=%s\n", username, req.IpAddress)

	vmInfo, err := s.manager.GetVmInfo(ipAddr)
	if err != nil {
		s.logger.Debugf(1, "GetVm(%s) failed: %v\n", username, err)
		return nil, lib_grpc.ErrorToStatus(err)
	}

	// Default to FULL view for GetVm
	view := req.View
	if view == pb.VmView_VM_VIEW_UNSPECIFIED {
		view = pb.VmView_VM_VIEW_FULL
	}

	return vmInfoToProto(&vmInfo, view), nil
}
