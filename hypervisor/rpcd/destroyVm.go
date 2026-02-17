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

func (t *srpcType) DestroyVm(conn *srpc.Conn,
	request hypervisor.DestroyVmRequest,
	reply *hypervisor.DestroyVmResponse) error {
	authInfo := conn.GetAuthInformation()
	t.logger.Debugf(1, "DestroyVm(%s) starting, IP=%s\n",
		authInfo.Username, request.IpAddress)
	err := t.manager.DestroyVm(request.IpAddress, authInfo, request.AccessToken)
	if err == nil {
		t.logger.Debugf(1, "DestroyVm(%s) finished, IP=%s\n",
			authInfo.Username, request.IpAddress)
	} else {
		t.logger.Debugf(1, "DestroyVm(%s) failed, IP=%s, error: %s\n",
			authInfo.Username, request.IpAddress, err)
	}
	response := hypervisor.DestroyVmResponse{errors.ErrorToString(err)}
	*reply = response
	return nil
}

// DestroyVm gRPC handler
func (s *grpcServer) DestroyVm(ctx context.Context, req *pb.DestroyVmRequest) (*pb.DestroyVmResponse, error) {
	conn := lib_grpc.ConnFromContext(ctx)
	if conn == nil {
		return nil, errors.NewUnauthenticatedError("no connection in context")
	}
	authInfo := conn.GetAuthInformation()
	if authInfo == nil {
		return nil, errors.NewUnauthenticatedError("no auth information")
	}

	ipAddr := net.ParseIP(req.IpAddress)
	s.logger.Debugf(1, "DestroyVm(%s) starting, IP=%s\n", authInfo.Username, req.IpAddress)

	err := s.manager.DestroyVm(ipAddr, authInfo, nil)
	if err == nil {
		s.logger.Debugf(1, "DestroyVm(%s) finished, IP=%s\n", authInfo.Username, req.IpAddress)
	} else {
		s.logger.Debugf(1, "DestroyVm(%s) failed, IP=%s, error: %s\n", authInfo.Username, req.IpAddress, err)
		return nil, lib_grpc.ErrorToStatus(err)
	}

	return &pb.DestroyVmResponse{}, nil
}
