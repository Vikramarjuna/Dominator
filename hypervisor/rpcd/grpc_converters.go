package rpcd

import (
	"net"

	"github.com/Cloud-Foundations/Dominator/lib/tags"
	proto "github.com/Cloud-Foundations/Dominator/proto/hypervisor"
	pb "github.com/Cloud-Foundations/Dominator/proto/hypervisor/grpc"
)

// ipToString converts net.IP to string
func ipToString(ip net.IP) string {
	if ip == nil {
		return ""
	}
	return ip.String()
}

// vmStateToProto converts internal State to proto VmState
func vmStateToProto(state proto.State) pb.VmState {
	switch state {
	case proto.StateStarting:
		return pb.VmState_VM_STATE_STARTING
	case proto.StateRunning:
		return pb.VmState_VM_STATE_RUNNING
	case proto.StateFailedToStart:
		return pb.VmState_VM_STATE_FAILED_TO_START
	case proto.StateStopping:
		return pb.VmState_VM_STATE_STOPPING
	case proto.StateStopped:
		return pb.VmState_VM_STATE_STOPPED
	case proto.StateMigrating:
		return pb.VmState_VM_STATE_MIGRATING
	case proto.StateExporting:
		return pb.VmState_VM_STATE_EXPORTING
	case proto.StateCrashed:
		return pb.VmState_VM_STATE_CRASHED
	case proto.StateDestroying:
		return pb.VmState_VM_STATE_DESTROYING
	case proto.StateDebugging:
		return pb.VmState_VM_STATE_DEBUGGING
	default:
		return pb.VmState_VM_STATE_UNSPECIFIED
	}
}

// vmInfoToProto converts internal VmInfo to proto Vm
func vmInfoToProto(vmInfo *proto.VmInfo, view pb.VmView) *pb.Vm {
	if vmInfo == nil {
		return nil
	}

	vm := &pb.Vm{
		IpAddress: ipToString(vmInfo.Address.IpAddress),
		State:     vmStateToProto(vmInfo.State),
		Hostname:  vmInfo.Hostname,
	}

	// For BASIC view, only return ip_address, hostname, state
	if view == pb.VmView_VM_VIEW_BASIC {
		return vm
	}

	// Full view includes all fields
	vm.MemoryInMib = uint32(vmInfo.MemoryInMiB)
	vm.MilliCpus = uint32(vmInfo.MilliCPUs)
	vm.ImageName = vmInfo.ImageName
	vm.OwnerUsers = vmInfo.OwnerUsers
	vm.Tags = tagsToMap(vmInfo.Tags)

	return vm
}

// tagsToMap converts tags.Tags to map[string]string
func tagsToMap(t tags.Tags) map[string]string {
	if t == nil {
		return nil
	}
	result := make(map[string]string, len(t))
	for k, v := range t {
		result[k] = v
	}
	return result
}

// mapToTags converts map[string]string to tags.Tags
func mapToTags(m map[string]string) tags.Tags {
	if m == nil {
		return nil
	}
	result := make(tags.Tags, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// vmFromProto converts proto Vm to internal CreateVmRequest.VmInfo
func vmFromProto(vm *pb.Vm) proto.VmInfo {
	if vm == nil {
		return proto.VmInfo{}
	}

	return proto.VmInfo{
		Hostname:    vm.Hostname,
		ImageName:   vm.ImageName,
		ImageURL:    vm.ImageUrl,
		MemoryInMiB: uint64(vm.MemoryInMib),
		MilliCPUs:   uint(vm.MilliCpus),
		OwnerUsers:  vm.OwnerUsers,
		Tags:        mapToTags(vm.Tags),
	}
}
