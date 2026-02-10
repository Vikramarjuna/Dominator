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
func (t *srpcType) RebootVm(conn *srpc.Conn,
	request hypervisor.RebootVmRequest,
	reply *hypervisor.RebootVmResponse) error {
	dhcpTimedOut, err := t.manager.RebootVm(request.IpAddress,
		conn.GetAuthInformation(), request.DhcpTimeout)
	response := hypervisor.RebootVmResponse{dhcpTimedOut,
		errors.ErrorToString(err)}
	*reply = response
	return nil
}

// gRPC handler
func (s *grpcServer) RebootVm(ctx context.Context,
	req *pb.RebootVmRequest) (*pb.RebootVmResponse, error) {
	ipAddr := net.IP(req.IpAddress)
	dhcpTimeout := time.Duration(req.DhcpTimeoutNs) * time.Nanosecond

	conn := lib_grpc.GetConn(ctx)
	dhcpTimedOut, err := s.manager.RebootVm(ipAddr,
		conn.GetAuthInformation(), dhcpTimeout)
	if err != nil {
		return nil, lib_grpc.ErrorToStatus(err)
	}
	return &pb.RebootVmResponse{
		DhcpTimedOut: dhcpTimedOut,
	}, nil
}
