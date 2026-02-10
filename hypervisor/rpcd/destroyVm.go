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

// SRPC handler
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

// gRPC sync handler - blocks until VM is fully destroyed
func (s *grpcServer) DestroyVm(ctx context.Context,
	req *pb.DestroyVmRequest) (*pb.DestroyVmResponse, error) {
	ipAddr := net.IP(req.IpAddress)

	conn := lib_grpc.GetConn(ctx)
	authInfo := conn.GetAuthInformation()
	s.logger.Debugf(1, "DestroyVm(%s) starting, IP=%s\n",
		authInfo.Username, ipAddr)
	err := s.manager.DestroyVm(ipAddr, authInfo, req.AccessToken)
	if err == nil {
		s.logger.Debugf(1, "DestroyVm(%s) finished, IP=%s\n",
			authInfo.Username, ipAddr)
	} else {
		s.logger.Debugf(1, "DestroyVm(%s) failed, IP=%s, error: %s\n",
			authInfo.Username, ipAddr, err)
		return nil, lib_grpc.ErrorToStatus(err)
	}
	return &pb.DestroyVmResponse{}, nil
}

// gRPC async handler - returns immediately (AWS EC2 pattern)
func (s *grpcServer) DestroyVmAsync(ctx context.Context,
	req *pb.DestroyVmAsyncRequest) (*pb.DestroyVmAsyncResponse, error) {
	ipAddr := net.IP(req.IpAddress)

	conn := lib_grpc.GetConn(ctx)
	authInfo := conn.GetAuthInformation()
	s.logger.Debugf(1, "DestroyVmAsync(%s) starting, IP=%s\n",
		authInfo.Username, ipAddr)
	vmInfo, err := s.manager.DestroyVmAsync(ipAddr, authInfo, req.AccessToken)
	if err != nil {
		s.logger.Debugf(1, "DestroyVmAsync(%s) failed, IP=%s, error: %s\n",
			authInfo.Username, ipAddr, err)
		return nil, lib_grpc.ErrorToStatus(err)
	}

	s.logger.Debugf(1, "DestroyVmAsync(%s) initiated, IP=%s, state=%s\n",
		authInfo.Username, ipAddr, vmInfo.State)

	pbVmInfo := vmInfoToProto(vmInfo)

	return &pb.DestroyVmAsyncResponse{
		VmInfo: pbVmInfo,
	}, nil
}
