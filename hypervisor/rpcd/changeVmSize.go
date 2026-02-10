package rpcd

import (
	"context"
	"net"

	"github.com/Cloud-Foundations/Dominator/lib/errors"
	lib_grpc "github.com/Cloud-Foundations/Dominator/lib/grpc"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
	pb "github.com/Cloud-Foundations/Dominator/proto/hypervisor/grpc"
)

// SRPC handler
func (t *srpcType) ChangeVmSize(conn *srpc.Conn,
	request hypervisor.ChangeVmSizeRequest,
	reply *hypervisor.ChangeVmSizeResponse) error {
	*reply = hypervisor.ChangeVmSizeResponse{
		errors.ErrorToString(
			t.manager.ChangeVmSize(conn.GetAuthInformation(), request))}
	return nil
}

// gRPC handler
func (s *grpcServer) ChangeVmSize(ctx context.Context,
	req *pb.ChangeVmSizeRequest) (*pb.ChangeVmSizeResponse, error) {
	ipAddr := net.IP(req.IpAddress)

	// Convert proto request to hypervisor request
	hypervisorReq := hypervisor.ChangeVmSizeRequest{
		IpAddress:   ipAddr,
		MemoryInMiB: req.MemoryInMib,
		MilliCPUs:   uint(req.MilliCpus),
		VirtualCPUs: uint(req.VirtualCpus),
	}

	conn := lib_grpc.GetConn(ctx)
	err := s.manager.ChangeVmSize(conn.GetAuthInformation(), hypervisorReq)
	if err != nil {
		return nil, lib_grpc.ErrorToStatus(err)
	}
	return &pb.ChangeVmSizeResponse{}, nil
}
