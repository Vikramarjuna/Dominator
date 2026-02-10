# Hypervisor API Implementation Checklist

This document tracks all APIs exposed by the Hypervisor service across SRPC and gRPC transports.

## Legend

**Access:**
- ✅ Public - Available to all authenticated users
- � Private - Requires special permissions (not in PublicMethods list)

**SRPC Type:**
- Sync - Synchronous request/response
- Async - Asynchronous (returns immediately)
- Stream - Streaming (bidirectional or server-side)

**gRPC Type:**
- Sync - Synchronous/blocking
- Async - Asynchronous/non-blocking
- Stream - Server-side streaming
- BiStream - Bidirectional streaming
- ❌ - Not implemented

**REST:**
- GET, POST, PUT, DELETE - HTTP method
- Stream - Server-sent events / chunked transfer
- ❌ - Not implemented

---

## VM Information APIs

| API | Access | SRPC Type | gRPC Type | REST | Status |
|-----|--------|-----------|-----------|------|--------|
| GetVmAccessToken | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| GetVmCreateRequest | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| GetVmInfo | ✅ | Sync | ✅ Sync | ✅ GET | ✅ Complete |
| GetVmInfos | ✅ | Sync | ✅ Sync | ✅ POST | ✅ Complete |
| GetVmLastPatchLog | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| GetVmUserData | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| GetVmVolume | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| GetVmVolumeStorageConfiguration | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| ListVMs | ✅ | Sync | ✅ Sync | ✅ GET | ✅ Complete |

## VM Lifecycle APIs

| API | Access | SRPC Type | gRPC Type | REST | Status |
|-----|--------|-----------|-----------|------|--------|
| AcknowledgeVm | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| CommitImportedVm | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| CopyVm | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| CreateVm | ✅ | Stream | ❌ | ❌ | ⏳ TODO (BiStream for ImageDataSize) |
| CreateVmAsync | ✅ | Async | ✅ Async | ✅ POST | ✅ Complete (ImageName/URL only) |
| DestroyVm | ✅ | Sync | ✅ Sync | ✅ DELETE | ✅ Complete |
| DestroyVmAsync | ✅ | Async | ✅ Async | ✅ DELETE | ✅ Complete |
| ExportLocalVm | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| ImportLocalVm | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| MigrateVm | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| PrepareVmForMigration | � | Sync | ❌ | ❌ | ⏳ TODO |
| RebootVm | ✅ | Sync | ✅ Sync | ✅ POST | ✅ Complete |
| StartVm | ✅ | Sync (blocking) | ✅ Sync | ✅ POST | ✅ Complete |
| StartVmAsync | N/A | N/A | ✅ Async | ✅ POST | ✅ Complete |
| StopVm | ✅ | Sync | ✅ Sync | ✅ POST | ✅ Complete |

## VM Configuration APIs

| API | Access | SRPC Type | gRPC Type | REST | Status |
|-----|--------|-----------|-----------|------|--------|
| BecomePrimaryVmOwner | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| ChangeVmConsoleType | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| ChangeVmCpuPriority | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| ChangeVmDestroyProtection | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| ChangeVmHostname | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| ChangeVmMachineType | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| ChangeVmNumNetworkQueues | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| ChangeVmOwnerGroups | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| ChangeVmOwnerUsers | ✅ | Sync | ✅ Sync | ✅ PATCH | ✅ Complete |
| ChangeVmSize | ✅ | Sync | ✅ Sync | ✅ PATCH | ✅ Complete |
| ChangeVmSubnet | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| ChangeVmTags | ✅ | Sync | ✅ Sync | ✅ PATCH | ✅ Complete |

## VM Volume Management APIs

| API | Access | SRPC Type | gRPC Type | REST | Status |
|-----|--------|-----------|-----------|------|--------|
| AddVmVolumes | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| ChangeVmVolumeInterfaces | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| ChangeVmVolumeSize | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| ChangeVmVolumeStorageIndex | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| DeleteVmVolume | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| ReorderVmVolumes | ✅ | Sync | ❌ | ❌ | ⏳ TODO |

## VM Image Management APIs

| API | Access | SRPC Type | gRPC Type | REST | Status |
|-----|--------|-----------|-----------|------|--------|
| DebugVmImage | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| DiscardVmOldImage | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| PatchVmImage | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| ReplaceVmImage | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| RestoreVmImage | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| ScanVmRoot | ✅ | Sync | ❌ | ❌ | ⏳ TODO |

## VM Snapshot APIs

| API | Access | SRPC Type | gRPC Type | REST | Status |
|-----|--------|-----------|-----------|------|--------|
| DiscardVmSnapshot | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| RestoreVmFromSnapshot | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| SnapshotVm | ✅ | Sync | ❌ | ❌ | ⏳ TODO |

## VM User Data APIs

| API | Access | SRPC Type | gRPC Type | REST | Status |
|-----|--------|-----------|-----------|------|--------|
| DiscardVmOldUserData | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| ReplaceVmUserData | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| RestoreVmUserData | ✅ | Sync | ❌ | ❌ | ⏳ TODO |

## VM Credentials & Identity APIs

| API | Access | SRPC Type | gRPC Type | REST | Status |
|-----|--------|-----------|-----------|------|--------|
| DiscardVmAccessToken | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| ReplaceVmCredentials | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| ReplaceVmIdentity | ✅ | Sync | ❌ | ❌ | ⏳ TODO |

## VM Console & Serial Port APIs

| API | Access | SRPC Type | gRPC Type | REST | Status |
|-----|--------|-----------|-----------|------|--------|
| ConnectToVmConsole | ✅ | Stream (BiDir) | ❌ | ❌ | ⏳ TODO |
| ConnectToVmManager | � | Stream (BiDir) | ❌ | ❌ | ⏳ TODO |
| ConnectToVmSerialPort | ✅ | Stream (BiDir) | ❌ | ❌ | ⏳ TODO |

## Monitoring & Debugging APIs

| API | Access | SRPC Type | gRPC Type | REST | Status |
|-----|--------|-----------|-----------|------|--------|
| GetUpdates | ✅ | Stream (Server) | ✅ Stream | ✅ Stream | ✅ Complete |
| ProbeVmPort | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| TraceVmMetadata | ✅ | Stream (Server) | ❌ | ❌ | ⏳ TODO |
| WatchDhcp | � | Stream (Server) | ❌ | ❌ | ⏳ TODO |

## Subnet & Network Management APIs

| API | Access | SRPC Type | gRPC Type | REST | Status |
|-----|--------|-----------|-----------|------|--------|
| ChangeAddressPool | � | Sync | ❌ | ❌ | ⏳ TODO |
| ListSubnets | ✅ | Sync | ✅ Sync | ✅ GET | ✅ Complete |
| NetbootMachine | � | Sync | ❌ | ❌ | ⏳ TODO |
| RegisterExternalLeases | � | Sync | ❌ | ❌ | ⏳ TODO |
| UpdateSubnets | � | Sync | ❌ | ❌ | ⏳ TODO |

## Hypervisor Management APIs

| API | Access | SRPC Type | gRPC Type | REST | Status |
|-----|--------|-----------|-----------|------|--------|
| GetCapacity | ✅ | Sync | ✅ Sync | ✅ GET | ✅ Complete |
| GetIdentityProvider | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| GetPublicKey | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| GetRootCookiePath | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| HoldLock | � | Sync | ❌ | ❌ | ⏳ TODO |
| HoldVmLock | � | Sync | ❌ | ❌ | ⏳ TODO |
| ListVolumeDirectories | ✅ | Sync | ❌ | ❌ | ⏳ TODO |
| PowerOff | � | Sync | ❌ | ❌ | ⏳ TODO |
| SetDisabledState | � | Sync | ❌ | ❌ | ⏳ TODO |

---

## Summary Statistics

**Total APIs:** 79

**By Access Level:**
- ✅ Public: 65 (82%)
- � Private: 14 (18%)

**By Implementation Status:**
- ✅ Complete (gRPC + REST): 14 (18%)
- ⏳ TODO: 65 (82%)

**gRPC Implementation:**
- Sync: 11
- Async: 1 (StartVmAsync)
- Stream: 1 (GetUpdates)
- Not Implemented: 66

**REST Implementation:**
- GET: 4
- POST: 5
- PATCH: 3
- DELETE: 1
- Stream: 1
- Not Implemented: 65

---

## Implementation Notes

### Completed APIs (14)

1. **GetVmInfo** - Get single VM information
2. **GetVmInfos** - Get multiple VMs with filters
3. **ListVMs** - List all VM IP addresses
4. **StartVm** - Synchronous VM start (SRPC-compatible)
5. **StartVmAsync** - Asynchronous VM start (AWS EC2 pattern)
6. **StopVm** - Stop a running VM
7. **RebootVm** - Reboot a VM
8. **DestroyVm** - Permanently destroy a VM
9. **ChangeVmTags** - Modify VM tags
10. **ChangeVmOwnerUsers** - Change VM owner users
11. **ChangeVmSize** - Resize VM (memory, CPUs)
12. **ListSubnets** - List available subnets
13. **GetUpdates** - Streaming watch for VM changes
14. **GetCapacity** - Get hypervisor capacity

### High Priority TODOs

**VM Lifecycle (Critical):**
- CreateVm (bidirectional streaming) - For ImageDataSize mode (rare use case)
- CopyVm - Common operation
- MigrateVm - Important for maintenance

**VM Configuration (High):**
- ChangeVmOwnerGroups
- ChangeVmConsoleType
- ChangeVmDestroyProtection
- ChangeVmHostname

**VM Image Management (High):**
- ReplaceVmImage
- PatchVmImage

**Monitoring (Medium):**
- TraceVmMetadata
- ProbeVmPort

**Console Access (Medium):**
- ConnectToVmConsole
- ConnectToVmSerialPort

### Streaming APIs

**Bidirectional Streaming (Complex):**
- ConnectToVmConsole - Interactive console access
- ConnectToVmSerialPort - Serial port access
- ConnectToVmManager - Management interface

**Server-Side Streaming:**
- GetUpdates - ✅ Implemented
- TraceVmMetadata - TODO
- WatchDhcp - TODO (private)

### Private APIs (Admin Only)

These APIs are not in the PublicMethods list and require special permissions:
- ConnectToVmManager
- PrepareVmForMigration
- UpdateSubnets
- ChangeAddressPool
- RegisterExternalLeases
- NetbootMachine
- SetDisabledState
- PowerOff
- HoldLock
- HoldVmLock
- WatchDhcp

---

## Summary Statistics

- **Total APIs:** 80 (79 original + 1 new async variant)
- **Completed:** 16 (20%)
- **In Progress:** 0
- **TODO:** 64 (80%)
- **Public APIs:** 65 (81%)
- **Private APIs:** 15 (19%)

## Next Steps

1. **Implement CreateVm bidirectional streaming** - For ImageDataSize mode (when needed)
2. **Implement VM Image APIs** - ReplaceVmImage, PatchVmImage
3. **Implement remaining VM configuration APIs**
4. **Implement VM volume management APIs**
5. **Implement snapshot APIs**
6. **Implement console/serial port streaming APIs** (complex)
7. **Add REST annotations** to proto file for all implemented APIs
8. **Update client libraries** with new gRPC methods

---

**Last Updated:** 2026-02-09
**Maintained By:** Auto-generated from SRPC PublicMethods list

