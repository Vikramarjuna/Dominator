package rpcd

import (
	"context"

	"github.com/Cloud-Foundations/Dominator/lib/errors"
	lib_grpc "github.com/Cloud-Foundations/Dominator/lib/grpc"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
	pb "github.com/Cloud-Foundations/Dominator/proto/hypervisor/grpc"
)

// SRPC handler
func (t *srpcType) GetVmInfos(conn *srpc.Conn,
	request hypervisor.GetVmInfosRequest,
	reply *hypervisor.GetVmInfosResponse) error {
	vmInfos, err := t.manager.GetVmInfos(request)
	*reply = hypervisor.GetVmInfosResponse{
		Error:   errors.ErrorToString(err),
		VmInfos: vmInfos,
	}
	return nil
}

// gRPC handler
func (s *grpcServer) GetVmInfos(ctx context.Context,
	req *pb.GetVmInfosRequest) (*pb.GetVmInfosResponse, error) {
	hypervisorReq := hypervisor.GetVmInfosRequest{
		IgnoreStateMask: req.IgnoreStateMask,
		OwnerGroups:     req.OwnerGroups,
		OwnerUsers:      req.OwnerUsers,
	}

	// Convert MatchTags
	if req.VmTagsToMatch != nil {
		hypervisorReq.VmTagsToMatch = lib_grpc.MatchTagsFromProto(req.VmTagsToMatch)
	}

	vmInfos, err := s.manager.GetVmInfos(hypervisorReq)
	if err != nil {
		return nil, lib_grpc.ErrorToStatus(err)
	}

	// Convert VmInfos to proto
	pbVmInfos := make([]*pb.VmInfo, len(vmInfos))
	for i, vmInfo := range vmInfos {
		pbVmInfos[i] = vmInfoToProto(&vmInfo)
	}

	return &pb.GetVmInfosResponse{
		VmInfos: pbVmInfos,
	}, nil
}
