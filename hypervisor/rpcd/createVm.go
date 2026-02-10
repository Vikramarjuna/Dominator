package rpcd

import (
	"context"
	"errors"
	"net"
	"time"

	lib_grpc "github.com/Cloud-Foundations/Dominator/lib/grpc"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/hypervisor"
	pb "github.com/Cloud-Foundations/Dominator/proto/hypervisor/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SRPC handler
func (t *srpcType) CreateVm(conn *srpc.Conn) error {
	return t.manager.CreateVm(conn)
}

// gRPC streaming handler - bidirectional streaming for ImageDataSize mode
// TODO: Not yet implemented
func (s *grpcServer) CreateVm(stream pb.Hypervisor_CreateVmServer) error {
	return lib_grpc.ErrorToStatus(errors.New("CreateVm streaming not yet implemented - use CreateVmAsync for ImageName/ImageURL modes"))
}

// validateNoStreamingFields validates that streaming-only fields are not used in CreateVmAsync.
// Returns Unimplemented status error if any streaming fields are set.
func validateNoStreamingFields(request *proto.CreateVmRequest) error {
	if request.ImageDataSize > 0 {
		return status.Error(codes.Unimplemented,
			"ImageDataSize streaming not supported in CreateVmAsync - use ImageName or ImageURL instead, or use CreateVm streaming API")
	}
	if request.UserDataSize > 0 {
		return status.Error(codes.Unimplemented,
			"UserDataSize streaming not supported in CreateVmAsync - use CreateVm streaming API")
	}
	if request.SecondaryVolumesData {
		return status.Error(codes.Unimplemented,
			"SecondaryVolumesData streaming not supported in CreateVmAsync - use CreateVm streaming API")
	}
	return nil
}

// gRPC async handler - returns immediately with VM in StateStarting or StateStopped
// Only supports ImageName and ImageURL modes (no ImageDataSize streaming)
func (s *grpcServer) CreateVmAsync(ctx context.Context,
	req *pb.CreateVmAsyncRequest) (*pb.CreateVmAsyncResponse, error) {

	// Get auth info
	conn := lib_grpc.GetConn(ctx)
	authInfo := conn.GetAuthInformation()

	s.logger.Debugf(1, "CreateVmAsync(%s) starting via gRPC\n", authInfo.Username)

	// Convert proto request to SRPC CreateVmRequest
	request, err := createVmAsyncRequestFromProto(req)
	if err != nil {
		s.logger.Debugf(1, "CreateVmAsync(%s) failed to convert request: %s\n",
			authInfo.Username, err)
		return nil, lib_grpc.ErrorToStatus(err)
	}

	// Validate that streaming modes are not used
	// Note: validateNoStreamingFields returns gRPC status errors directly,
	// so we don't call ErrorToStatus on it
	if err := validateNoStreamingFields(request); err != nil {
		s.logger.Debugf(1, "CreateVmAsync(%s) validation failed: %s\n", authInfo.Username, err)
		return nil, err // Already a gRPC status error
	}

	// Call the async CreateVm business logic
	vmInfo, err := s.manager.CreateVmAsync(*request, authInfo)
	if err != nil {
		s.logger.Debugf(1, "CreateVmAsync(%s) failed: %s\n", authInfo.Username, err)
		return nil, lib_grpc.ErrorToStatus(err)
	}

	s.logger.Debugf(1, "CreateVmAsync(%s) initiated, hostname=%s, IP=%s, state=%s\n",
		authInfo.Username, vmInfo.Hostname, vmInfo.Address.IpAddress, vmInfo.State)

	pbVmInfo := vmInfoToProto(vmInfo)
	return &pb.CreateVmAsyncResponse{VmInfo: pbVmInfo}, nil
}

// Helper function to convert proto request to SRPC CreateVmRequest
func createVmAsyncRequestFromProto(req *pb.CreateVmAsyncRequest) (*proto.CreateVmRequest, error) {
	request := &proto.CreateVmRequest{
		DhcpTimeout:         time.Duration(req.DhcpTimeoutNs),
		DoNotStart:          req.DoNotStart,
		EnableNetboot:       req.EnableNetboot,
		IdentityCertificate: req.IdentityCertificate,
		IdentityKey:         req.IdentityKey,
		ImageTimeout:        time.Duration(req.ImageTimeoutNs),
		MinimumFreeBytes:    req.MinimumFreeBytes,
		OverlayDirectories:  req.OverlayDirectories,
		OverlayFiles:        req.OverlayFiles,
		RoundupPower:        req.RoundupPower,
		SkipBootloader:      req.SkipBootloader,
		SkipMemoryCheck:     req.SkipMemoryCheck,
	}

	// Convert storage indices
	request.StorageIndices = make([]uint, len(req.StorageIndices))
	for i, idx := range req.StorageIndices {
		request.StorageIndices[i] = uint(idx)
	}

	// Convert VmInfo fields
	request.VmInfo = proto.VmInfo{
		ConsoleType:        proto.ConsoleType(req.ConsoleType),
		CpuPriority:        int(req.CpuPriority),
		DestroyOnPowerdown: req.DestroyOnPowerdown,
		DestroyProtection:  req.DestroyProtection,
		DisableVirtIO:      req.DisableVirtIo,
		ExtraKernelOptions: req.ExtraKernelOptions,
		FirmwareType:       proto.FirmwareType(req.FirmwareType),
		Hostname:           req.Hostname,
		ImageName:          req.ImageName,
		ImageURL:           req.ImageUrl,
		MachineType:        proto.MachineType(req.MachineType),
		MemoryInMiB:        req.MemoryInMib,
		MilliCPUs:          uint(req.MilliCpus),
		OwnerGroups:        req.OwnerGroups,
		OwnerUsers:         req.OwnerUsers,
		SpreadVolumes:      req.SpreadVolumes,
		SubnetId:           req.SubnetId,
		Tags:               lib_grpc.TagsFromProto(req.Tags),
		VirtualCPUs:        uint(req.VirtualCpus),
		WatchdogAction:     proto.WatchdogAction(req.WatchdogAction),
		WatchdogModel:      proto.WatchdogModel(req.WatchdogModel),
	}

	// Convert network entries
	if len(req.NetworkEntries) > 0 {
		request.VmInfo.NetworkEntries = make([]proto.NetworkEntry, len(req.NetworkEntries))
		for i, ne := range req.NetworkEntries {
			request.VmInfo.NetworkEntries[i] = networkEntryFromProto(ne)
		}
	}

	// Convert secondary addresses
	if len(req.SecondaryAddresses) > 0 {
		request.VmInfo.SecondaryAddresses = make([]proto.Address, len(req.SecondaryAddresses))
		for i, addr := range req.SecondaryAddresses {
			request.VmInfo.SecondaryAddresses[i] = addressFromProto(addr)
		}
	}

	// Convert secondary subnet IDs
	request.VmInfo.SecondarySubnetIDs = req.SecondarySubnetIds

	// Convert volumes
	if len(req.Volumes) > 0 {
		request.VmInfo.Volumes = make([]proto.Volume, len(req.Volumes))
		for i, vol := range req.Volumes {
			request.VmInfo.Volumes[i] = volumeFromProto(vol)
		}
	}

	// Convert secondary volumes
	if len(req.SecondaryVolumes) > 0 {
		request.SecondaryVolumes = make([]proto.Volume, len(req.SecondaryVolumes))
		for i, vol := range req.SecondaryVolumes {
			request.SecondaryVolumes[i] = volumeFromProto(vol)
		}
	}

	// Convert secondary volumes init
	if len(req.SecondaryVolumesInit) > 0 {
		request.SecondaryVolumesInit = make([]proto.VolumeInitialisationInfo, len(req.SecondaryVolumesInit))
		for i, vinit := range req.SecondaryVolumesInit {
			request.SecondaryVolumesInit[i] = proto.VolumeInitialisationInfo{
				BytesPerInode:            vinit.BytesPerInode,
				Label:                    vinit.Label,
				ReservedBlocksPercentage: uint16(vinit.ReservedBlocksPercentage),
			}
		}
	}

	return request, nil
}

// Helper functions to convert from proto to SRPC types
func addressFromProto(a *pb.Address) proto.Address {
	if a == nil {
		return proto.Address{}
	}
	return proto.Address{
		IpAddress:  net.IP(a.IpAddress),
		MacAddress: a.MacAddress,
	}
}

func networkEntryFromProto(n *pb.NetworkEntry) proto.NetworkEntry {
	if n == nil {
		return proto.NetworkEntry{}
	}
	// Note: proto.NetworkEntry has more fields than hypervisor.NetworkEntry
	// hypervisor.NetworkEntry only has NumQueues field
	return proto.NetworkEntry{}
}

func volumeFromProto(v *pb.Volume) proto.Volume {
	if v == nil {
		return proto.Volume{}
	}
	return proto.Volume{
		Format:      proto.VolumeFormat(v.Format),
		Interface:   proto.VolumeInterface(v.Interface),
		Size:        v.Size,
		Snapshots:   v.Snapshots,
		Type:        proto.VolumeType(v.Type),
		VirtualSize: v.VirtualSize,
	}
}
