package rpcd

import (
	"context"
	"net"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/errors"
	lib_grpc "github.com/Cloud-Foundations/Dominator/lib/grpc"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
	pb "github.com/Cloud-Foundations/Dominator/proto/hypervisor/grpc"
)

// SRPC handler
func (t *srpcType) StartVm(conn *srpc.Conn,
	request hypervisor.StartVmRequest,
	reply *hypervisor.StartVmResponse) error {
	dhcpTimedOut, err := t.manager.StartVm(request.IpAddress,
		conn.GetAuthInformation(), request.AccessToken, request.DhcpTimeout)
	response := hypervisor.StartVmResponse{dhcpTimedOut,
		errors.ErrorToString(err)}
	*reply = response
	return nil
}

// gRPC handler - Synchronous/blocking (SRPC-compatible)
// Blocks until VM starts and DHCP completes (or times out).
// This provides the same behavior as SRPC for easy migration.
func (s *grpcServer) StartVm(ctx context.Context,
	req *pb.StartVmRequest) (*pb.StartVmResponse, error) {
	ipAddr := net.IP(req.IpAddress)
	dhcpTimeout := time.Duration(req.DhcpTimeoutNs) * time.Nanosecond

	conn := lib_grpc.GetConn(ctx)
	dhcpTimedOut, err := s.manager.StartVm(ipAddr,
		conn.GetAuthInformation(), req.AccessToken, dhcpTimeout)
	if err != nil {
		return nil, lib_grpc.ErrorToStatus(err)
	}
	return &pb.StartVmResponse{
		DhcpTimedOut: dhcpTimedOut,
	}, nil
}

// gRPC handler - Asynchronous/non-blocking (AWS EC2 pattern)
// Returns immediately with VM in StateStarting. Clients should use GetVmInfo
// or GetUpdates to monitor state transitions.
func (s *grpcServer) StartVmAsync(ctx context.Context,
	req *pb.StartVmAsyncRequest) (*pb.StartVmAsyncResponse, error) {
	ipAddr := net.IP(req.IpAddress)
	dhcpTimeout := time.Duration(req.DhcpTimeoutNs) * time.Nanosecond

	conn := lib_grpc.GetConn(ctx)
	vmInfo, err := s.manager.StartVmAsync(ipAddr,
		conn.GetAuthInformation(), req.AccessToken, dhcpTimeout)
	if err != nil {
		return nil, lib_grpc.ErrorToStatus(err)
	}

	// Convert VmInfo to proto
	pbVmInfo := vmInfoToProto(vmInfo)

	return &pb.StartVmAsyncResponse{
		VmInfo: pbVmInfo,
	}, nil
}
