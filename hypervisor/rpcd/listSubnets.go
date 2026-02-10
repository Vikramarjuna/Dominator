package rpcd

import (
	"context"

	lib_grpc "github.com/Cloud-Foundations/Dominator/lib/grpc"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
	pb "github.com/Cloud-Foundations/Dominator/proto/hypervisor/grpc"
)

// SRPC handler
func (t *srpcType) ListSubnets(conn *srpc.Conn,
	request hypervisor.ListSubnetsRequest,
	reply *hypervisor.ListSubnetsResponse) error {
	*reply = hypervisor.ListSubnetsResponse{
		Subnets: t.manager.ListSubnets(request.Sort)}
	return nil
}

// gRPC handler
func (s *grpcServer) ListSubnets(ctx context.Context,
	req *pb.ListSubnetsRequest) (*pb.ListSubnetsResponse, error) {
	subnets := s.manager.ListSubnets(req.Sort)

	// Convert subnets to proto
	pbSubnets := make([]*pb.Subnet, len(subnets))
	for i, subnet := range subnets {
		pbSubnets[i] = subnetToProto(&subnet)
	}

	return &pb.ListSubnetsResponse{
		Subnets: pbSubnets,
	}, nil
}

// subnetToProto converts hypervisor.Subnet to proto Subnet
func subnetToProto(s *hypervisor.Subnet) *pb.Subnet {
	if s == nil {
		return nil
	}

	pbSubnet := &pb.Subnet{
		Id:              s.Id,
		IpGateway:       []byte(s.IpGateway),
		IpMask:          []byte(s.IpMask),
		DomainName:      s.DomainName,
		DisableMetadata: s.DisableMetadata,
		Manage:          s.Manage,
		VlanId:          uint32(s.VlanId),
		AllowedGroups:   s.AllowedGroups,
		AllowedUsers:    s.AllowedUsers,
		FirstDynamicIp:  []byte(s.FirstDynamicIP),
		LastDynamicIp:   []byte(s.LastDynamicIP),
		Tags:            lib_grpc.TagsToProto(s.Tags),
	}

	// Convert DomainNameServers
	for _, dns := range s.DomainNameServers {
		pbSubnet.DomainNameServers = append(pbSubnet.DomainNameServers, []byte(dns))
	}

	return pbSubnet
}
