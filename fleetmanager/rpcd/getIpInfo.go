package rpcd

import (
	"context"
	"net"

	"github.com/Cloud-Foundations/Dominator/lib/errors"
	lib_grpc "github.com/Cloud-Foundations/Dominator/lib/grpc"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/fleetmanager"
	pb "github.com/Cloud-Foundations/Dominator/proto/fleetmanager/grpc"
)

// SRPC handler
func (t *srpcType) GetIpInfo(conn *srpc.Conn,
	request proto.GetIpInfoRequest,
	reply *proto.GetIpInfoResponse) error {
	ipAddr := request.IpAddress
	if response, err := t.hypervisorsManager.GetIpInfo(ipAddr); err != nil {
		*reply = proto.GetIpInfoResponse{
			Error: errors.ErrorToString(err),
		}
	} else {
		*reply = response
	}
	return nil
}

// gRPC handler
func (s *grpcServer) GetIpInfo(ctx context.Context,
	req *pb.GetIpInfoRequest) (*pb.GetIpInfoResponse, error) {

	// Convert gRPC request to internal request (inline - simple conversion)
	ipAddr := net.IP(req.IpAddress)

	// Call hypervisorsManager
	internalResp, err := s.hypervisorsManager.GetIpInfo(ipAddr)
	if err != nil {
		return nil, lib_grpc.ErrorToStatus(err)
	}

	// Convert internal response to gRPC response
	resp := &pb.GetIpInfoResponse{
		HypervisorAddress: internalResp.HypervisorAddress,
	}
	if internalResp.VM != nil {
		resp.Vm = vmInfoToProto(internalResp.VM)
	}
	return resp, nil
}
