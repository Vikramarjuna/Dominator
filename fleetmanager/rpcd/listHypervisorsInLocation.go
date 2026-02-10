package rpcd

import (
	"context"

	"github.com/Cloud-Foundations/Dominator/lib/errors"
	lib_grpc "github.com/Cloud-Foundations/Dominator/lib/grpc"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	common_grpc "github.com/Cloud-Foundations/Dominator/proto/common/grpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/fleetmanager"
	pb "github.com/Cloud-Foundations/Dominator/proto/fleetmanager/grpc"
)

// SRPC handler
func (t *srpcType) ListHypervisorsInLocation(conn *srpc.Conn,
	request proto.ListHypervisorsInLocationRequest,
	reply *proto.ListHypervisorsInLocationResponse) error {
	response, err := t.hypervisorsManager.ListHypervisorsInLocation(request)
	if err == nil {
		*reply = response
	} else {
		*reply = proto.ListHypervisorsInLocationResponse{
			Error: errors.ErrorToString(err),
		}
	}
	return nil
}

// gRPC handler
func (s *grpcServer) ListHypervisorsInLocation(ctx context.Context,
	req *pb.ListHypervisorsInLocationRequest) (*pb.ListHypervisorsInLocationResponse, error) {

	// Convert to internal request type
	internalReq := proto.ListHypervisorsInLocationRequest{
		HypervisorTagsToMatch: lib_grpc.MatchTagsFromProto(req.HypervisorTagsToMatch),
		IncludeUnhealthy:      req.IncludeUnhealthy,
		Location:              req.Location,
		SubnetId:              req.SubnetId,
		TagsToInclude:         req.TagsToInclude,
	}

	response, err := s.hypervisorsManager.ListHypervisorsInLocation(internalReq)
	if err != nil {
		return nil, lib_grpc.ErrorToStatus(err)
	}

	// Convert tags for each hypervisor
	var tagsForHypervisors []*common_grpc.Tags
	if response.TagsForHypervisors != nil {
		tagsForHypervisors = make([]*common_grpc.Tags, len(response.TagsForHypervisors))
		for i, t := range response.TagsForHypervisors {
			tagsForHypervisors[i] = lib_grpc.TagsToProto(t)
		}
	}

	return &pb.ListHypervisorsInLocationResponse{
		HypervisorAddresses: response.HypervisorAddresses,
		TagsForHypervisors:  tagsForHypervisors,
	}, nil
}
