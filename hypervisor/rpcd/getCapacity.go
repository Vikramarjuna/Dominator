package rpcd

import (
	"context"

	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
	pb "github.com/Cloud-Foundations/Dominator/proto/hypervisor/grpc"
)

// SRPC handler
func (t *srpcType) GetCapacity(conn *srpc.Conn,
	request hypervisor.GetCapacityRequest,
	reply *hypervisor.GetCapacityResponse) error {
	*reply = t.manager.GetCapacity()
	return nil
}

// gRPC handler
func (s *grpcServer) GetCapacity(ctx context.Context,
	req *pb.GetCapacityRequest) (*pb.GetCapacityResponse, error) {
	capacity := s.manager.GetCapacity()

	// Note: The proto GetCapacityResponse has different fields than the SRPC version
	// SRPC has: MemoryInMiB, NumCPUs, TotalVolumeBytes
	// gRPC proto has: TotalMemoryInMib, AvailableMemoryInMib, TotalNumCpus, TotalVolumeBytes, AvailableVolumeBytes
	// For now, we'll map the available fields
	return &pb.GetCapacityResponse{
		TotalMemoryInMib: capacity.MemoryInMiB,
		TotalNumCpus:     uint32(capacity.NumCPUs),
		TotalVolumeBytes: capacity.TotalVolumeBytes,
		// AvailableMemoryInMib and AvailableVolumeBytes are not available in the current response
	}, nil
}
