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
func (t *srpcType) GetVmInfo(conn *srpc.Conn,
	request hypervisor.GetVmInfoRequest,
	reply *hypervisor.GetVmInfoResponse) error {
	info, err := t.manager.GetVmInfo(request.IpAddress)
	response := hypervisor.GetVmInfoResponse{
		VmInfo: info,
		Error:  errors.ErrorToString(err),
	}
	*reply = response
	return nil
}

// gRPC handler
func (s *grpcServer) GetVmInfo(ctx context.Context,
	req *pb.GetVmInfoRequest) (*pb.GetVmInfoResponse, error) {
	ipAddr := net.IP(req.IpAddress)
	info, err := s.manager.GetVmInfo(ipAddr)
	if err != nil {
		return nil, lib_grpc.ErrorToStatus(err)
	}
	return &pb.GetVmInfoResponse{
		VmInfo: vmInfoToProto(&info),
	}, nil
}

// vmInfoToProto converts hypervisor.VmInfo to proto VmInfo
func vmInfoToProto(v *hypervisor.VmInfo) *pb.VmInfo {
	if v == nil {
		return nil
	}

	pbVmInfo := &pb.VmInfo{
		Address:             addressToProto(&v.Address),
		ChangedStateOn:      v.ChangedStateOn.Unix(),
		ConsoleType:         uint32(v.ConsoleType),
		CreatedOn:           v.CreatedOn.Unix(),
		CpuPriority:         int32(v.CpuPriority),
		DestroyOnPowerdown:  v.DestroyOnPowerdown,
		DestroyProtection:   v.DestroyProtection,
		DisableVirtIo:       v.DisableVirtIO,
		ExtraKernelOptions:  v.ExtraKernelOptions,
		FirmwareType:        uint32(v.FirmwareType),
		Hostname:            v.Hostname,
		IdentityExpires:     v.IdentityExpires.Unix(),
		IdentityName:        v.IdentityName,
		ImageName:           v.ImageName,
		ImageUrl:            v.ImageURL,
		MachineType:         uint32(v.MachineType),
		MemoryInMib:         v.MemoryInMiB,
		MilliCpus:           uint32(v.MilliCPUs),
		RootFileSystemLabel: v.RootFileSystemLabel,
		SpreadVolumes:       v.SpreadVolumes,
		State:               uint32(v.State),
		SubnetId:            v.SubnetId,
		Tags:                lib_grpc.TagsToProto(v.Tags),
		Uncommitted:         v.Uncommitted,
		VirtualCpus:         uint32(v.VirtualCPUs),
		WatchdogAction:      uint32(v.WatchdogAction),
		WatchdogModel:       uint32(v.WatchdogModel),
		IpAddress:           []byte(v.Address.IpAddress),
	}

	// Convert slices
	for _, entry := range v.NetworkEntries {
		pbVmInfo.NetworkEntries = append(pbVmInfo.NetworkEntries, networkEntryToProto(&entry))
	}
	for _, group := range v.OwnerGroups {
		pbVmInfo.OwnerGroups = append(pbVmInfo.OwnerGroups, group)
	}
	for _, user := range v.OwnerUsers {
		pbVmInfo.OwnerUsers = append(pbVmInfo.OwnerUsers, user)
	}
	for _, addr := range v.SecondaryAddresses {
		pbVmInfo.SecondaryAddresses = append(pbVmInfo.SecondaryAddresses, addressToProto(&addr))
	}
	for _, subnetID := range v.SecondarySubnetIDs {
		pbVmInfo.SecondarySubnetIds = append(pbVmInfo.SecondarySubnetIds, subnetID)
	}
	for _, vol := range v.Volumes {
		pbVmInfo.Volumes = append(pbVmInfo.Volumes, volumeToProto(&vol))
	}

	return pbVmInfo
}

func addressToProto(a *hypervisor.Address) *pb.Address {
	if a == nil {
		return nil
	}
	return &pb.Address{
		IpAddress:  []byte(a.IpAddress),
		MacAddress: a.MacAddress,
	}
}

func networkEntryToProto(n *hypervisor.NetworkEntry) *pb.NetworkEntry {
	if n == nil {
		return nil
	}
	// Note: hypervisor.NetworkEntry only has NumQueues field
	// The proto NetworkEntry has more fields but they're not populated from this type
	return &pb.NetworkEntry{}
}

func volumeToProto(v *hypervisor.Volume) *pb.Volume {
	if v == nil {
		return nil
	}
	return &pb.Volume{
		Format:      uint32(v.Format),
		Interface:   uint32(v.Interface),
		Size:        v.Size,
		Snapshots:   v.Snapshots,
		Type:        uint32(v.Type),
		VirtualSize: v.VirtualSize,
	}
}
