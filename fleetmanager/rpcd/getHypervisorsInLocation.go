package rpcd

import (
	"context"

	"github.com/Cloud-Foundations/Dominator/lib/errors"
	lib_grpc "github.com/Cloud-Foundations/Dominator/lib/grpc"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	fm_proto "github.com/Cloud-Foundations/Dominator/proto/fleetmanager"
	proto "github.com/Cloud-Foundations/Dominator/proto/fleetmanager"
	pb "github.com/Cloud-Foundations/Dominator/proto/fleetmanager/grpc"
	hypervisor_grpc "github.com/Cloud-Foundations/Dominator/proto/hypervisor/grpc"
)

// SRPC handler
func (t *srpcType) GetHypervisorsInLocation(conn *srpc.Conn,
	request proto.GetHypervisorsInLocationRequest,
	reply *proto.GetHypervisorsInLocationResponse) error {
	response, err := t.hypervisorsManager.GetHypervisorsInLocation(request)
	if err == nil {
		*reply = response
	} else {
		*reply = proto.GetHypervisorsInLocationResponse{
			Error: errors.ErrorToString(err),
		}
	}
	return nil
}

// gRPC handler
func (s *grpcServer) GetHypervisorsInLocation(ctx context.Context,
	req *pb.GetHypervisorsInLocationRequest) (*pb.GetHypervisorsInLocationResponse, error) {

	// Convert gRPC request to internal request (inline - simple conversion)
	internalReq := fm_proto.GetHypervisorsInLocationRequest{
		HypervisorTagsToMatch: lib_grpc.MatchTagsFromProto(req.HypervisorTagsToMatch),
		IncludeUnhealthy:      req.IncludeUnhealthy,
		IncludeVMs:            req.IncludeVms,
		Location:              req.Location,
		SubnetId:              req.SubnetId,
	}

	// Call hypervisorsManager
	internalResp, err := s.hypervisorsManager.GetHypervisorsInLocation(internalReq)
	if err != nil {
		return nil, lib_grpc.ErrorToStatus(err)
	}

	// Convert internal response to gRPC response
	return getHypervisorsInLocationResponseToProto(&internalResp), nil
}

// Converters for GetHypervisorsInLocation (used only in this file)

func getHypervisorsInLocationResponseToProto(t *fm_proto.GetHypervisorsInLocationResponse) *pb.GetHypervisorsInLocationResponse {
	if t == nil {
		return nil
	}
	resp := &pb.GetHypervisorsInLocationResponse{
		Hypervisors: make([]*pb.Hypervisor, len(t.Hypervisors)),
	}
	for i, h := range t.Hypervisors {
		resp.Hypervisors[i] = hypervisorToProto(&h)
	}
	return resp
}

func hypervisorToProto(t *fm_proto.Hypervisor) *pb.Hypervisor {
	if t == nil {
		return nil
	}
	pbHypervisor := &pb.Hypervisor{
		AllocatedMilliCpus:   t.AllocatedMilliCPUs,
		AllocatedMemory:      t.AllocatedMemory,
		AllocatedVolumeBytes: t.AllocatedVolumeBytes,
		Machine:              machineToProto(&t.Machine),
	}
	if t.VMs != nil {
		pbHypervisor.Vms = make([]*hypervisor_grpc.VmInfo, len(t.VMs))
		for i, vm := range t.VMs {
			pbHypervisor.Vms[i] = vmInfoToProto(&vm)
		}
	}
	return pbHypervisor
}
