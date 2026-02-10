package rpcd

import (
	"context"
	"fmt"

	"github.com/Cloud-Foundations/Dominator/lib/constants"
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	lib_grpc "github.com/Cloud-Foundations/Dominator/lib/grpc"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/fleetmanager"
	pb "github.com/Cloud-Foundations/Dominator/proto/fleetmanager/grpc"
)

// SRPC handler
func (t *srpcType) GetHypervisorForVM(conn *srpc.Conn,
	request proto.GetHypervisorForVMRequest,
	reply *proto.GetHypervisorForVMResponse) error {
	hypervisor, err := t.hypervisorsManager.GetHypervisorForVm(
		request.IpAddress)
	response := proto.GetHypervisorForVMResponse{
		Error: errors.ErrorToString(err),
	}
	if err == nil {
		response.HypervisorAddress = fmt.Sprintf("%s:%d",
			hypervisor, constants.HypervisorPortNumber)
	}
	*reply = response
	return nil
}

// gRPC handler
func (s *grpcServer) GetHypervisorForVM(ctx context.Context,
	req *pb.GetHypervisorForVMRequest) (*pb.GetHypervisorForVMResponse, error) {

	hypervisor, err := s.hypervisorsManager.GetHypervisorForVm(lib_grpc.IpFromBytes(req.IpAddress))
	if err != nil {
		return nil, lib_grpc.ErrorToStatus(err)
	}

	return &pb.GetHypervisorForVMResponse{
		HypervisorAddress: fmt.Sprintf("%s:%d",
			hypervisor, constants.HypervisorPortNumber),
	}, nil
}
