package rpcd

import (
	"fmt"
	"net"
	"time"

	lib_grpc "github.com/Cloud-Foundations/Dominator/lib/grpc"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	fm_proto "github.com/Cloud-Foundations/Dominator/proto/fleetmanager"
	proto "github.com/Cloud-Foundations/Dominator/proto/fleetmanager"
	pb "github.com/Cloud-Foundations/Dominator/proto/fleetmanager/grpc"
	hyper_proto "github.com/Cloud-Foundations/Dominator/proto/hypervisor"
	hypervisor_grpc "github.com/Cloud-Foundations/Dominator/proto/hypervisor/grpc"
)

const flushDelay = time.Millisecond * 10

func (t *srpcType) GetUpdates(conn *srpc.Conn) error {
	var request proto.GetUpdatesRequest
	if err := conn.Decode(&request); err != nil {
		return err
	}
	closeChannel := conn.GetCloseNotifier()
	updateChannel := t.hypervisorsManager.MakeUpdateChannel(request)
	defer t.hypervisorsManager.CloseUpdateChannel(updateChannel)
	flushTimer := time.NewTimer(flushDelay)
	var numToFlush uint
	maxUpdates := request.MaxUpdates
	for count := uint64(0); maxUpdates < 1 || count < maxUpdates; {
		select {
		case update, ok := <-updateChannel:
			if !ok {
				return fmt.Errorf(
					"error sending update to: %s for: %s: receiver not keeping up with updates",
					conn.RemoteAddr(), conn.Username())
			}
			if err := conn.Encode(update); err != nil {
				return fmt.Errorf("error sending update: %s", err)
			}
			if update.Error != "" {
				return nil
			}
			count++
			numToFlush++
			if !flushTimer.Stop() {
				select {
				case <-flushTimer.C:
				default:
				}
			}
			if len(updateChannel) < 1 {
				flushTimer.Reset(flushDelay)
			}
		case <-flushTimer.C:
			if numToFlush > 1 {
				t.logger.Debugf(0, "flushing %d events\n", numToFlush)
			}
			numToFlush = 0
			if err := conn.Flush(); err != nil {
				return fmt.Errorf("error flushing update(s): %s", err)
			}
		case err := <-closeChannel:
			if err == nil {
				t.logger.Debugf(0, "update client disconnected: %s\n",
					conn.RemoteAddr())
				return nil
			}
			return err
		}
	}
	return nil
}

// gRPC handler (streaming)
// Standard gRPC watch pattern - streams updates until client cancels context
func (s *grpcServer) GetUpdates(req *pb.GetUpdatesRequest,
	stream pb.FleetManager_GetUpdatesServer) error {

	// Convert to internal request type
	// Note: SRPC layer still uses MaxUpdates, but gRPC clients control via context
	internalReq := proto.GetUpdatesRequest{
		IgnoreMissingLocalTags: req.IgnoreMissingLocalTags,
		Location:               req.Location,
		MaxUpdates:             0, // 0 = infinite for SRPC layer
	}

	updateChannel := s.hypervisorsManager.MakeUpdateChannel(internalReq)
	defer s.hypervisorsManager.CloseUpdateChannel(updateChannel)

	// Stream updates indefinitely until client cancels context
	for {
		select {
		case <-stream.Context().Done():
			// Client canceled context - standard gRPC pattern
			s.logger.Debugf(0, "gRPC update client disconnected\n")
			return stream.Context().Err()

		case update, ok := <-updateChannel:
			if !ok {
				// Channel closed - client not keeping up
				// Return error via gRPC status (standard practice)
				return lib_grpc.ErrorToStatus(fmt.Errorf("receiver not keeping up with updates"))
			}

			// Check for error in internal update (from SRPC layer)
			if update.Error != "" {
				// Return error via gRPC status (standard practice)
				return lib_grpc.ErrorToStatus(fmt.Errorf("%s", update.Error))
			}

			pbUpdate := convertUpdate(&update)
			if err := stream.Send(pbUpdate); err != nil {
				return err
			}
		}
	}
}

// convertUpdate converts internal Update to gRPC Update.
// Note: Errors are handled separately via gRPC status codes, not in the message.
func convertUpdate(u *proto.Update) *pb.Update {
	pbUpdate := &pb.Update{}

	// Convert changed machines
	if u.ChangedMachines != nil {
		pbUpdate.ChangedMachines = make([]*pb.Machine, len(u.ChangedMachines))
		for i, m := range u.ChangedMachines {
			pbMachine := machineToProto(m)
			pbMachine.Hostname = m.Hostname
			pbUpdate.ChangedMachines[i] = pbMachine
		}
	}

	// Convert changed VMs
	if u.ChangedVMs != nil {
		pbUpdate.ChangedVms = make(map[string]*hypervisor_grpc.VmInfo)
		for ip, vm := range u.ChangedVMs {
			pbVm := vmInfoToProto(vm)
			pbVm.IpAddress = []byte(vm.Address.IpAddress)
			pbUpdate.ChangedVms[ip] = pbVm
		}
	}

	// Copy deleted lists
	pbUpdate.DeletedMachines = u.DeletedMachines
	pbUpdate.DeletedVms = u.DeletedVMs

	// Convert VM to hypervisor map
	pbUpdate.VmToHypervisor = u.VmToHypervisor

	return pbUpdate
}

func machineToProto(t *fm_proto.Machine) *pb.Machine {
	if t == nil {
		return nil
	}
	machine := &pb.Machine{
		GatewaySubnetId:  t.GatewaySubnetId,
		Ipmi:             networkEntryToProto(t.IPMI),
		Location:         t.Location,
		MemoryInMib:      t.MemoryInMiB,
		NetworkEntry:     networkEntryToProto(t.NetworkEntry),
		NumCpus:          uint32(t.NumCPUs),
		OwnerGroups:      t.OwnerGroups,
		OwnerUsers:       t.OwnerUsers,
		Tags:             lib_grpc.TagsToProto(t.Tags),
		TotalVolumeBytes: t.TotalVolumeBytes,
		Hostname:         t.Hostname,
	}
	machine.SecondaryNetworkEntries = make([]*pb.NetworkEntry, len(t.SecondaryNetworkEntries))
	for i := range t.SecondaryNetworkEntries {
		machine.SecondaryNetworkEntries[i] = networkEntryToProto(t.SecondaryNetworkEntries[i])
	}
	return machine
}

func vmInfoToProto(t *hyper_proto.VmInfo) *hypervisor_grpc.VmInfo {
	if t == nil {
		return nil
	}
	return &hypervisor_grpc.VmInfo{
		Hostname:    t.Hostname,
		ImageName:   t.ImageName,
		ImageUrl:    t.ImageURL,
		MemoryInMib: t.MemoryInMiB,
		MilliCpus:   uint32(t.MilliCPUs),
		OwnerGroups: t.OwnerGroups,
		OwnerUsers:  t.OwnerUsers,
		State:       uint32(t.State),
		SubnetId:    t.SubnetId,
		Tags:        lib_grpc.TagsToProto(t.Tags),
	}
}

func networkEntryToProto(e fm_proto.NetworkEntry) *pb.NetworkEntry {
	return &pb.NetworkEntry{
		Hostname:       e.Hostname,
		HostIpAddress:  ipToBytes(e.HostIpAddress),
		HostMacAddress: hardwareAddrToBytes(e.HostMacAddress),
		SubnetId:       e.SubnetId,
		VlanTrunk:      e.VlanTrunk,
	}
}

func ipToBytes(ip net.IP) []byte {
	if ip == nil {
		return nil
	}
	return []byte(ip)
}

func hardwareAddrToBytes(addr fm_proto.HardwareAddr) []byte {
	if addr == nil {
		return nil
	}
	return []byte(addr)
}
