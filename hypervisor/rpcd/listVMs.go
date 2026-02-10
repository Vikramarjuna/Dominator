package rpcd

import (
	"context"
	"net"

	lib_grpc "github.com/Cloud-Foundations/Dominator/lib/grpc"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
	pb "github.com/Cloud-Foundations/Dominator/proto/hypervisor/grpc"
)

// SRPC handler
func (t *srpcType) ListVMs(conn *srpc.Conn,
	request hypervisor.ListVMsRequest,
	reply *hypervisor.ListVMsResponse) error {
	ipAddressStrings := t.manager.ListVMs(request)
	ipAddresses := make([]net.IP, 0, len(ipAddressStrings))
	for _, ipAddressString := range ipAddressStrings {
		ipAddress := net.ParseIP(ipAddressString)
		if shrunkIP := ipAddress.To4(); shrunkIP != nil {
			ipAddress = shrunkIP
		}
		ipAddresses = append(ipAddresses, ipAddress)
	}
	*reply = hypervisor.ListVMsResponse{IpAddresses: ipAddresses}
	return nil
}

// gRPC handler
func (s *grpcServer) ListVMs(ctx context.Context,
	req *pb.ListVMsRequest) (*pb.ListVMsResponse, error) {
	// Convert proto request to hypervisor request
	hypervisorReq := hypervisor.ListVMsRequest{
		IgnoreStateMask: req.IgnoreStateMask,
		OwnerGroups:     req.OwnerGroups,
		OwnerUsers:      req.OwnerUsers,
		Sort:            req.Sort,
	}

	// Convert MatchTags
	if req.VmTagsToMatch != nil {
		hypervisorReq.VmTagsToMatch = lib_grpc.MatchTagsFromProto(req.VmTagsToMatch)
	}

	ipAddressStrings := s.manager.ListVMs(hypervisorReq)
	ipAddresses := make([][]byte, 0, len(ipAddressStrings))
	for _, ipAddressString := range ipAddressStrings {
		ipAddress := net.ParseIP(ipAddressString)
		if shrunkIP := ipAddress.To4(); shrunkIP != nil {
			ipAddress = shrunkIP
		}
		ipAddresses = append(ipAddresses, []byte(ipAddress))
	}

	return &pb.ListVMsResponse{
		IpAddresses: ipAddresses,
	}, nil
}
