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
func (t *srpcType) ChangeVmTags(conn *srpc.Conn,
	request hypervisor.ChangeVmTagsRequest,
	reply *hypervisor.ChangeVmTagsResponse) error {
	response := hypervisor.ChangeVmTagsResponse{
		errors.ErrorToString(
			t.manager.ChangeVmTags(request.IpAddress, conn.GetAuthInformation(),
				request.Tags))}
	*reply = response
	return nil
}

// gRPC handler
func (s *grpcServer) ChangeVmTags(ctx context.Context,
	req *pb.ChangeVmTagsRequest) (*pb.ChangeVmTagsResponse, error) {
	ipAddr := net.IP(req.IpAddress)
	tags := lib_grpc.TagsFromProto(req.Tags)

	conn := lib_grpc.GetConn(ctx)
	err := s.manager.ChangeVmTags(ipAddr, conn.GetAuthInformation(), tags)
	if err != nil {
		return nil, lib_grpc.ErrorToStatus(err)
	}
	return &pb.ChangeVmTagsResponse{}, nil
}
