package rpcd

import (
	"context"

	"github.com/Cloud-Foundations/Dominator/lib/errors"
	lib_grpc "github.com/Cloud-Foundations/Dominator/lib/grpc"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/fleetmanager"
	pb "github.com/Cloud-Foundations/Dominator/proto/fleetmanager/grpc"
)

// SRPC handler
func (t *srpcType) ListHypervisorLocations(conn *srpc.Conn,
	request proto.ListHypervisorLocationsRequest,
	reply *proto.ListHypervisorLocationsResponse) error {
	locations, err := t.hypervisorsManager.ListLocations(request.TopLocation)
	*reply = proto.ListHypervisorLocationsResponse{
		Locations: locations,
		Error:     errors.ErrorToString(err),
	}
	return nil
}

// gRPC handler
func (s *grpcServer) ListHypervisorLocations(ctx context.Context,
	req *pb.ListHypervisorLocationsRequest) (*pb.ListHypervisorLocationsResponse, error) {

	locations, err := s.hypervisorsManager.ListLocations(req.TopLocation)
	if err != nil {
		return nil, lib_grpc.ErrorToStatus(err)
	}

	return &pb.ListHypervisorLocationsResponse{
		Locations: locations,
	}, nil
}
