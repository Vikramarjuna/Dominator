package rpcd

import (
	"context"

	"github.com/Cloud-Foundations/Dominator/fleetmanager/hypervisors"
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	lib_grpc "github.com/Cloud-Foundations/Dominator/lib/grpc"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	fm_proto "github.com/Cloud-Foundations/Dominator/proto/fleetmanager"
	pb "github.com/Cloud-Foundations/Dominator/proto/fleetmanager/grpc"
	hyper_proto "github.com/Cloud-Foundations/Dominator/proto/hypervisor"
	hypervisor_grpc "github.com/Cloud-Foundations/Dominator/proto/hypervisor/grpc"
)

// SRPC handler
func (t *srpcType) GetMachineInfo(conn *srpc.Conn,
	request fm_proto.GetMachineInfoRequest,
	reply *fm_proto.GetMachineInfoResponse) error {
	if response, err := getMachineInfo(t.hypervisorsManager, request); err != nil {
		*reply = fm_proto.GetMachineInfoResponse{
			Error: errors.ErrorToString(err)}
	} else {
		*reply = response
	}
	return nil
}

// gRPC handler
func (s *grpcServer) GetMachineInfo(ctx context.Context,
	req *pb.GetMachineInfoRequest) (*pb.GetMachineInfoResponse, error) {
	internalReq := fm_proto.GetMachineInfoRequest{
		Hostname:               req.Hostname,
		IgnoreMissingLocalTags: req.IgnoreMissingLocalTags,
	}

	internalResp, err := getMachineInfo(s.hypervisorsManager, internalReq)
	if err != nil {
		return nil, lib_grpc.ErrorToStatus(err)
	}

	return getMachineInfoResponseToProto(&internalResp), nil
}

// getMachineInfo contains the shared business logic for getting machine information.
func getMachineInfo(manager *hypervisors.Manager, request fm_proto.GetMachineInfoRequest) (
	fm_proto.GetMachineInfoResponse, error) {
	topology, err := manager.GetTopology()
	if err != nil {
		return fm_proto.GetMachineInfoResponse{}, err
	}
	location, err := topology.GetLocationOfMachine(request.Hostname)
	if err != nil {
		return fm_proto.GetMachineInfoResponse{}, err
	}
	machine, err := manager.GetMachineInfo(request)
	if err != nil {
		return fm_proto.GetMachineInfoResponse{}, err
	}
	tSubnets, err := topology.GetSubnetsForMachine(request.Hostname)
	if err != nil {
		return fm_proto.GetMachineInfoResponse{}, err
	}
	subnets := make([]*hyper_proto.Subnet, 0, len(tSubnets))
	for _, tSubnet := range tSubnets {
		subnets = append(subnets, &tSubnet.Subnet)
	}
	return fm_proto.GetMachineInfoResponse{
		Location: location,
		Machine:  machine,
		Subnets:  subnets,
	}, nil
}

func getMachineInfoResponseToProto(t *fm_proto.GetMachineInfoResponse) *pb.GetMachineInfoResponse {
	if t == nil {
		return nil
	}
	resp := &pb.GetMachineInfoResponse{
		Location: t.Location,
		Machine:  machineToProto(&t.Machine),
	}
	// Convert Subnet slice
	resp.Subnets = make([]*hypervisor_grpc.Subnet, 0, len(t.Subnets))
	for _, subnet := range t.Subnets {
		resp.Subnets = append(resp.Subnets, subnetToProto(subnet))
	}
	return resp
}

func subnetToProto(t *hyper_proto.Subnet) *hypervisor_grpc.Subnet {
	if t == nil {
		return nil
	}
	return &hypervisor_grpc.Subnet{
		Id:         t.Id,
		IpGateway:  ipToBytes(t.IpGateway),
		IpMask:     ipToBytes(t.IpMask),
		DomainName: t.DomainName,
		VlanId:     uint32(t.VlanId),
	}
}
