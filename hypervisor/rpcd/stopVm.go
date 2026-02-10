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
func (t *srpcType) StopVm(conn *srpc.Conn,
	request hypervisor.StopVmRequest, reply *hypervisor.StopVmResponse) error {
	response := hypervisor.StopVmResponse{
		errors.ErrorToString(t.manager.StopVm(request.IpAddress,
			conn.GetAuthInformation(), request.AccessToken))}
	*reply = response
	return nil
}

// gRPC handler
func (s *grpcServer) StopVm(ctx context.Context,
	req *pb.StopVmRequest) (*pb.StopVmResponse, error) {
	ipAddr := net.IP(req.IpAddress)

	conn := lib_grpc.GetConn(ctx)
	err := s.manager.StopVm(ipAddr, conn.GetAuthInformation(), req.AccessToken)
	if err != nil {
		return nil, lib_grpc.ErrorToStatus(err)
	}
	return &pb.StopVmResponse{}, nil
}
