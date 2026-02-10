package rpcd

import (
	"context"

	"github.com/Cloud-Foundations/Dominator/lib/errors"
	lib_grpc "github.com/Cloud-Foundations/Dominator/lib/grpc"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/fleetmanager"
	pb "github.com/Cloud-Foundations/Dominator/proto/fleetmanager/grpc"
)

// SRPC handler
func (t *srpcType) PowerOnMachine(conn *srpc.Conn,
	request fleetmanager.PowerOnMachineRequest,
	reply *fleetmanager.PowerOnMachineResponse) error {
	*reply = fleetmanager.PowerOnMachineResponse{
		errors.ErrorToString(t.hypervisorsManager.PowerOnMachine(
			request.Hostname, conn.GetAuthInformation()))}
	return nil
}

// gRPC handler
func (s *grpcServer) PowerOnMachine(ctx context.Context,
	req *pb.PowerOnMachineRequest) (*pb.PowerOnMachineResponse, error) {

	conn := lib_grpc.GetConn(ctx)

	err := s.hypervisorsManager.PowerOnMachine(
		req.Hostname,
		conn.GetAuthInformation())

	if err != nil {
		return nil, lib_grpc.ErrorToStatus(err)
	}

	return &pb.PowerOnMachineResponse{}, nil
}
