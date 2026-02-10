package rpcd

import (
	"context"
	"net"

	"github.com/Cloud-Foundations/Dominator/lib/errors"
	lib_grpc "github.com/Cloud-Foundations/Dominator/lib/grpc"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	fm_proto "github.com/Cloud-Foundations/Dominator/proto/fleetmanager"
	pb "github.com/Cloud-Foundations/Dominator/proto/fleetmanager/grpc"
)

// SRPC handler
func (t *srpcType) MoveIpAddresses(conn *srpc.Conn,
	request fm_proto.MoveIpAddressesRequest,
	reply *fm_proto.MoveIpAddressesResponse) error {
	err := t.hypervisorsManager.MoveIpAddresses(request.HypervisorHostname,
		request.IpAddresses)
	if err != nil {
		*reply = fm_proto.MoveIpAddressesResponse{
			Error: errors.ErrorToString(err)}
	}
	return nil
}

// gRPC handler
func (s *grpcServer) MoveIpAddresses(ctx context.Context,
	req *pb.MoveIpAddressesRequest) (*pb.MoveIpAddressesResponse, error) {

	// Convert gRPC request to internal format (inline - simple conversion)
	ipAddresses := make([]net.IP, len(req.IpAddresses))
	for i, ipBytes := range req.IpAddresses {
		ipAddresses[i] = net.IP(ipBytes)
	}

	// Call hypervisorsManager
	err := s.hypervisorsManager.MoveIpAddresses(req.HypervisorHostname, ipAddresses)
	if err != nil {
		return nil, lib_grpc.ErrorToStatus(err)
	}

	return &pb.MoveIpAddressesResponse{}, nil
}
