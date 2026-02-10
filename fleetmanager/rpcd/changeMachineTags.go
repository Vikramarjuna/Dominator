package rpcd

import (
	"context"

	"github.com/Cloud-Foundations/Dominator/lib/errors"
	lib_grpc "github.com/Cloud-Foundations/Dominator/lib/grpc"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/fleetmanager"
	pb "github.com/Cloud-Foundations/Dominator/proto/fleetmanager/grpc"
)

// SRPC handler
func (t *srpcType) ChangeMachineTags(conn *srpc.Conn,
	request fleetmanager.ChangeMachineTagsRequest,
	reply *fleetmanager.ChangeMachineTagsResponse) error {
	*reply = fleetmanager.ChangeMachineTagsResponse{
		errors.ErrorToString(t.hypervisorsManager.ChangeMachineTags(
			request.Hostname, conn.GetAuthInformation(), request.Tags))}
	return nil
}

// gRPC handler
func (s *grpcServer) ChangeMachineTags(ctx context.Context,
	req *pb.ChangeMachineTagsRequest) (*pb.ChangeMachineTagsResponse, error) {

	conn := lib_grpc.GetConn(ctx)

	err := s.hypervisorsManager.ChangeMachineTags(
		req.Hostname,
		conn.GetAuthInformation(),
		lib_grpc.TagsFromProto(req.Tags))

	if err != nil {
		return nil, lib_grpc.ErrorToStatus(err)
	}

	return &pb.ChangeMachineTagsResponse{}, nil
}
