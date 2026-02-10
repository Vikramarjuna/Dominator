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
func (t *srpcType) ChangeVmOwnerUsers(conn *srpc.Conn,
	request hypervisor.ChangeVmOwnerUsersRequest,
	reply *hypervisor.ChangeVmOwnerUsersResponse) error {
	response := hypervisor.ChangeVmOwnerUsersResponse{
		errors.ErrorToString(
			t.manager.ChangeVmOwnerUsers(request.IpAddress,
				conn.GetAuthInformation(),
				request.OwnerUsers))}
	*reply = response
	return nil
}

// gRPC handler
func (s *grpcServer) ChangeVmOwnerUsers(ctx context.Context,
	req *pb.ChangeVmOwnerUsersRequest) (*pb.ChangeVmOwnerUsersResponse, error) {
	ipAddr := net.IP(req.IpAddress)

	conn := lib_grpc.GetConn(ctx)
	err := s.manager.ChangeVmOwnerUsers(ipAddr, conn.GetAuthInformation(),
		req.OwnerUsers)
	if err != nil {
		return nil, lib_grpc.ErrorToStatus(err)
	}
	return &pb.ChangeVmOwnerUsersResponse{}, nil
}
