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

func (t *srpcType) AcknowledgeVm(conn *srpc.Conn,
	request hypervisor.AcknowledgeVmRequest,
	reply *hypervisor.AcknowledgeVmResponse) error {
	response := hypervisor.AcknowledgeVmResponse{
		errors.ErrorToString(t.manager.AcknowledgeVm(request.IpAddress,
			conn.GetAuthInformation()))}
	*reply = response
	return nil
}

// AcknowledgeVm is the gRPC handler. It returns an error if the caller is not
// authenticated or if the manager refuses the acknowledgement.
func (s *grpcServer) AcknowledgeVm(ctx context.Context,
	req *pb.AcknowledgeVmRequest) (*pb.AcknowledgeVmResponse, error) {
	authInfo := lib_grpc.AuthInfoFromContext(ctx).GetAuthInformation()
	if authInfo == nil {
		return nil, errors.NewUnauthenticatedError("no auth information")
	}
	s.logger.Debugf(1, "AcknowledgeVm(%s) starting, IP=%s\n",
		authInfo.Username, req.IpAddress)
	err := s.manager.AcknowledgeVm(net.ParseIP(req.IpAddress), authInfo)
	if err != nil {
		s.logger.Debugf(1, "AcknowledgeVm(%s) failed, IP=%s, error: %s\n",
			authInfo.Username, req.IpAddress, err)
		return nil, lib_grpc.ErrorToStatus(err)
	}
	s.logger.Debugf(1, "AcknowledgeVm(%s) finished, IP=%s\n",
		authInfo.Username, req.IpAddress)
	return &pb.AcknowledgeVmResponse{}, nil
}
