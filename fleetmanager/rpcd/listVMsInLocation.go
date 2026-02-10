package rpcd

import (
	"context"
	"fmt"

	"github.com/Cloud-Foundations/Dominator/lib/errors"
	lib_grpc "github.com/Cloud-Foundations/Dominator/lib/grpc"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/fleetmanager"
	pb "github.com/Cloud-Foundations/Dominator/proto/fleetmanager/grpc"
)

// SRPC handler
func (t *srpcType) ListVMsInLocation(conn *srpc.Conn) error {
	var request proto.ListVMsInLocationRequest
	if err := conn.Decode(&request); err != nil {
		return err
	}
	addresses, err := t.hypervisorsManager.ListVMsInLocation(request)
	if err != nil {
		response := proto.ListVMsInLocationResponse{
			Error: errors.ErrorToString(err),
		}
		if err := conn.Encode(response); err != nil {
			return err
		}
		return nil
	}
	// TODO(rgooch): Chunk the response.
	response := proto.ListVMsInLocationResponse{IpAddresses: addresses}
	if err := conn.Encode(response); err != nil {
		return err
	}
	response.IpAddresses = nil // Send end-of-chunks message.
	return conn.Encode(response)
}

// gRPC handler (unary with pagination)
func (s *grpcServer) ListVMsInLocation(ctx context.Context,
	req *pb.ListVMsInLocationRequest) (*pb.ListVMsInLocationResponse, error) {

	// Convert to internal request type
	internalReq := proto.ListVMsInLocationRequest{
		HypervisorTagsToMatch: lib_grpc.MatchTagsFromProto(req.HypervisorTagsToMatch),
		Location:              req.Location,
		OwnerGroups:           req.OwnerGroups,
		OwnerUsers:            req.OwnerUsers,
		VmTagsToMatch:         lib_grpc.MatchTagsFromProto(req.VmTagsToMatch),
	}

	// Get all IP addresses
	ipAddrs, err := s.hypervisorsManager.ListVMsInLocation(internalReq)
	if err != nil {
		return nil, lib_grpc.ErrorToStatus(err)
	}

	// Handle pagination
	startIdx := 0
	if req.PageToken != "" {
		// Decode page token (simple integer offset for now)
		var offset int
		if _, err := fmt.Sscanf(req.PageToken, "%d", &offset); err != nil {
			return nil, lib_grpc.ErrorToStatus(
				fmt.Errorf("invalid page token: %s", req.PageToken))
		}
		startIdx = offset
		if startIdx >= len(ipAddrs) {
			// Past the end - return empty response
			return &pb.ListVMsInLocationResponse{}, nil
		}
	}

	// Determine page size
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		// Return all remaining results
		pageSize = len(ipAddrs) - startIdx
	}

	// Calculate end index
	endIdx := startIdx + pageSize
	var nextPageToken string
	if endIdx < len(ipAddrs) {
		// More results available
		nextPageToken = fmt.Sprintf("%d", endIdx)
		// Don't go past the end
	} else {
		endIdx = len(ipAddrs)
		// nextPageToken remains empty (no more results)
	}

	// Convert page of IPs to bytes
	pageIPs := ipAddrs[startIdx:endIdx]
	ipBytes := make([][]byte, len(pageIPs))
	for i, ip := range pageIPs {
		ipBytes[i] = []byte(ip)
	}

	return &pb.ListVMsInLocationResponse{
		IpAddresses:   ipBytes,
		NextPageToken: nextPageToken,
	}, nil
}
