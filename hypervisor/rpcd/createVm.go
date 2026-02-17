package rpcd

import (
	"context"
	"errors"
	"time"

	liberrors "github.com/Cloud-Foundations/Dominator/lib/errors"
	lib_grpc "github.com/Cloud-Foundations/Dominator/lib/grpc"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/lib/tags"
	proto "github.com/Cloud-Foundations/Dominator/proto/hypervisor"
	pb "github.com/Cloud-Foundations/Dominator/proto/hypervisor/grpc"
)

func (t *srpcType) CreateVm(conn *srpc.Conn) error {
	return t.manager.CreateVm(conn)
}

func validateCreateVmRequest(vm *pb.Vm) error {
	if vm == nil {
		return liberrors.NewInvalidArgumentError("vm", "required")
	}
	if vm.Hostname == "" {
		return liberrors.NewInvalidArgumentError("vm.hostname", "required")
	}
	if vm.MemoryInMib == 0 {
		return liberrors.NewInvalidArgumentError("vm.memory_in_mib", "required")
	}
	if vm.MilliCpus == 0 {
		return liberrors.NewInvalidArgumentError("vm.milli_cpus", "required")
	}
	if vm.ImageName == "" && vm.ImageUrl == "" {
		return liberrors.NewInvalidArgumentError("vm.image", "either image_name or image_url must be specified")
	}
	if vm.ImageName != "" && vm.ImageUrl != "" {
		return liberrors.NewInvalidArgumentError("vm.image", "cannot specify both image_name and image_url")
	}
	return nil
}

// CreateVm gRPC handler (async/non-blocking)
// Returns immediately with VM in starting state
// Client polls GetVm to check completion
// For minimal implementation, this returns Unimplemented
// TODO: Implement async VM creation in follow-up PR
func (s *grpcServer) CreateVm(ctx context.Context, req *pb.CreateVmRequest) (*pb.Vm, error) {
	username := "unknown"
	if conn := lib_grpc.ConnFromContext(ctx); conn != nil {
		if authInfo := conn.GetAuthInformation(); authInfo != nil {
			username = authInfo.Username
		}
	}

	vm := req.GetVm()
	hostname := ""
	if vm != nil {
		hostname = vm.Hostname
	}

	s.logger.Debugf(1, "CreateVm(%s) called for hostname=%s\n", username, hostname)

	// Validate request
	if err := validateCreateVmRequest(vm); err != nil {
		return nil, err
	}

	// For minimal implementation, return Unimplemented
	// The full implementation would create VM asynchronously
	return nil, liberrors.NewUnimplementedError("CreateVm async - use CreateVmStreaming API")
}

// CreateVmStreaming gRPC handler - bidirectional streaming
// Client sends: initial metadata request, then optionally streams image data chunks
// Server sends: multiple progress messages during VM creation
// This reuses the manager's CreateVm implementation by wrapping the gRPC stream
// in an SRPC-compatible conn interface
func (s *grpcServer) CreateVmStreaming(stream pb.Hypervisor_CreateVmStreamingServer) error {
	conn := lib_grpc.ConnFromContext(stream.Context())
	if conn == nil {
		return liberrors.NewUnauthenticatedError("no connection in context")
	}
	authInfo := conn.GetAuthInformation()
	if authInfo == nil {
		return liberrors.NewUnauthenticatedError("no auth information")
	}

	s.logger.Debugf(1, "CreateVmStreaming(gRPC) called\n")

	// Receive first message (metadata)
	firstMsg, err := stream.Recv()
	if err != nil {
		s.logger.Debugf(1, "CreateVmStreaming(gRPC) failed to receive metadata: %v\n", err)
		return err
	}

	metadata := firstMsg.GetMetadata()
	if metadata == nil {
		return liberrors.NewInvalidArgumentError("metadata", "first message must contain metadata")
	}

	hostname := ""
	if metadata.Vm != nil {
		hostname = metadata.Vm.Hostname
	}
	s.logger.Debugf(1, "CreateVmStreaming(gRPC) for hostname=%s\n", hostname)

	// Create streaming conn wrapper with conversion functions
	streamConn := lib_grpc.NewStreamingConn(
		stream.Context(),
		createVmStreamingRequestDecoder(metadata, authInfo),
		createVmStreamingResponseEncoder(stream),
		lib_grpc.WithReadFunc(createImageDataReader(stream)),
	)

	// Call the manager's CreateVm - it will use conn.Decode(), conn.Read(), and conn.Encode()
	err = s.manager.CreateVm(streamConn)
	if err != nil {
		s.logger.Debugf(1, "CreateVmStreaming(gRPC) failed: %v\n", err)
		return lib_grpc.ErrorToStatus(err)
	}

	s.logger.Debugf(1, "CreateVmStreaming(gRPC) finished\n")
	return nil
}

// createVmStreamingRequestDecoder returns a function that converts gRPC CreateVmStreamingMetadata to SRPC CreateVmRequest.
// This is called once by the manager when it calls Decode().
// Image data streaming is handled separately via the Read() method.
func createVmStreamingRequestDecoder(metadata *pb.CreateVmStreamingMetadata, authInfo *srpc.AuthInformation) func(v any) error {
	return func(v any) error {
		reqPtr, ok := v.(*proto.CreateVmRequest)
		if !ok {
			return liberrors.NewInternalError("unexpected decode type")
		}

		vm := metadata.GetVm()
		if vm == nil {
			return liberrors.NewInvalidArgumentError("metadata.vm", "required")
		}

		// Convert gRPC Vm to SRPC proto request
		vmTags := make(tags.Tags)
		for k, v := range vm.Tags {
			vmTags[k] = v
		}

		ownerUsers := make([]string, 1, len(vm.OwnerUsers)+1)
		ownerUsers[0] = authInfo.Username
		if ownerUsers[0] == "" {
			return liberrors.NewUnauthenticatedError("no authentication data")
		}
		ownerUsers = append(ownerUsers, vm.OwnerUsers...)

		dhcpTimeout := time.Duration(metadata.DhcpTimeout)

		*reqPtr = proto.CreateVmRequest{
			VmInfo: proto.VmInfo{
				Hostname:    vm.Hostname,
				ImageName:   vm.ImageName,
				ImageURL:    vm.ImageUrl,
				MemoryInMiB: uint64(vm.MemoryInMib),
				MilliCPUs:   uint(vm.MilliCpus),
				OwnerUsers:  ownerUsers,
				Tags:        vmTags,
			},
			DoNotStart:    vm.DoNotStart,
			DhcpTimeout:   dhcpTimeout,
			ImageDataSize: metadata.ImageDataSize,
			UserDataSize:  metadata.UserDataSize,
		}
		return nil
	}
}

// createImageDataReader returns a function that reads image data chunks from the gRPC stream.
// This implements the io.Reader interface for StreamingConn.
// The manager calls conn.Read() to stream image data after decoding the initial request.
func createImageDataReader(stream pb.Hypervisor_CreateVmStreamingServer) func(p []byte) (int, error) {
	var buffer []byte // Buffer for partial chunk data

	return func(p []byte) (int, error) {
		// If we have buffered data, use it first
		if len(buffer) > 0 {
			n := copy(p, buffer)
			buffer = buffer[n:]
			return n, nil
		}

		// Receive next message from stream
		msg, err := stream.Recv()
		if err != nil {
			return 0, err // io.EOF when client closes stream
		}

		// Get image data chunk
		chunk := msg.GetImageDataChunk()
		if chunk == nil {
			// No more image data
			return 0, nil
		}

		// Copy chunk to output buffer
		n := copy(p, chunk)
		if n < len(chunk) {
			// Save remaining data for next Read() call
			buffer = chunk[n:]
		}
		return n, nil
	}
}

// createVmStreamingResponseEncoder returns a function that converts SRPC CreateVmResponse to gRPC CreateVmStreamingResponse.
// This is called multiple times by the manager for streaming progress updates.
//
// Error handling: SRPC sends errors in the response's Error field. For gRPC, we convert
// these to gRPC status errors so the client receives them properly.
//
// Final flag: SRPC uses a Final flag to indicate the last response, but gRPC uses stream completion.
// The manager will send Final=true in the last response, then return. We just send all responses
// (including the final one), and the stream closes naturally when the handler returns.
func createVmStreamingResponseEncoder(stream pb.Hypervisor_CreateVmStreamingServer) func(v any) error {
	return func(v any) error {
		// Handle both pointer and value types
		var resp *proto.CreateVmResponse
		if respPtr, ok := v.(*proto.CreateVmResponse); ok {
			resp = respPtr
		} else if respVal, ok := v.(proto.CreateVmResponse); ok {
			resp = &respVal
		} else {
			return liberrors.NewInternalError("unexpected encode type")
		}

		// If the response contains an error, return it as a gRPC status error
		// This is how SRPC sends errors - via the Error field in the response
		// Use ErrorToStatus to map to appropriate gRPC status codes
		if resp.Error != "" {
			return lib_grpc.ErrorToStatus(errors.New(resp.Error))
		}

		// Convert SRPC response to gRPC response
		// Note: We ignore the Final flag - gRPC doesn't need it
		grpcResp := &pb.CreateVmStreamingResponse{
			ProgressMessage: resp.ProgressMessage,
			Vm: &pb.Vm{
				IpAddress: ipToString(resp.IpAddress),
			},
			DhcpTimedOut: resp.DhcpTimedOut,
		}

		// Send the response (including final response if Final=true)
		return stream.Send(grpcResp)
	}
}
