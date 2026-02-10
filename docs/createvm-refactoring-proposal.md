# CreateVm Refactoring Proposal

## Problems Identified

### 1. Code Duplication
The `createVmAsync` background goroutine duplicates ~200 lines of code from the synchronous `createVm` function. This includes:
- Image fetching (ImageName, ImageURL modes)
- Volume setup
- Secondary volume creation
- Memory allocation checks
- Error handling

**Impact**: 
- Maintenance burden (bugs need to be fixed in two places)
- Risk of divergence between sync and async implementations
- Harder to add new features

### 2. Crash Recovery
If the hypervisor process crashes while a background goroutine is creating a VM:
- The VM is left in `StateStarting` forever
- Resources (IP address, volumes) may be partially allocated
- No automatic recovery on restart

**Impact**:
- Resource leaks
- Orphaned VMs that can't be cleaned up
- Manual intervention required

## Proposed Solutions

### Solution 1: Extract Common VM Creation Logic

**Approach**: Create a shared internal function that both sync and async can use.

```go
// Internal function that does the actual VM creation work
// progressCallback is optional - if nil, no progress updates are sent
type vmCreationProgressCallback func(message string) error

func (m *Manager) createVmInternal(
    vm *vmInfoType,
    request proto.CreateVmRequest,
    progressCallback vmCreationProgressCallback,
) error {
    // All the common logic:
    // - Memory allocation check
    // - Directory creation
    // - Key pair writing
    // - Image fetching (ImageName/ImageURL/MinimumFreeBytes)
    // - Volume setup
    // - Secondary volumes
    // - Starting the VM
    
    // Calls progressCallback("message") at key points if non-nil
}

// Synchronous version (SRPC streaming)
func (m *Manager) createVm(conn *srpc.Conn) error {
    // ... setup and validation ...
    vm, err := m.allocateVm(request, conn.GetAuthInformation())
    // ... 
    
    // Use the shared function with progress callback
    progressCallback := func(msg string) error {
        return sendUpdate(conn, msg)
    }
    
    return m.createVmInternal(vm, request, progressCallback)
}

// Asynchronous version (gRPC/SRPC async)
func (m *Manager) createVmAsyncBackground(vm *vmInfoType, request proto.CreateVmRequest) {
    // Use the shared function without progress callback
    err := m.createVmInternal(vm, request, nil)
    if err != nil {
        vm.setState(proto.StateFailedToStart)
    }
}
```

**Benefits**:
- Single source of truth for VM creation logic
- Easier to maintain and test
- Sync version gets progress updates, async version doesn't
- Both use identical creation logic

### Solution 2: Crash Recovery via State Persistence

**Approach**: Persist VM state immediately after allocation, detect incomplete creations on restart.

#### Step 1: Persist State Early
```go
func (m *Manager) createVmAsync(...) (*proto.VmInfo, error) {
    // ... validation ...
    
    vm, err := m.allocateVm(request, authInfo)
    if err != nil {
        return nil, err
    }
    
    // Set initial state
    if request.DoNotStart {
        vm.setState(proto.StateStopped)
    } else {
        vm.setState(proto.StateStarting)
    }
    
    // CRITICAL: Write state to disk BEFORE starting background work
    // This ensures we can detect incomplete VMs on restart
    if err := vm.writeInfo(); err != nil {
        vm.cleanup()
        return nil, err
    }
    
    // Now start background work
    go m.createVmAsyncBackground(vm, request)
    
    return &vm.VmInfo, nil
}
```

#### Step 2: Detect and Recover on Startup
```go
// In manager startup (start.go), after loading VMs from disk:
func (m *Manager) recoverIncompleteVms() {
    for ipAddr, vm := range m.vms {
        if vm.State == proto.StateStarting {
            // VM was being created when hypervisor crashed
            m.Logger.Printf("Detected incomplete VM creation: %s, marking as failed\n", ipAddr)
            vm.setState(proto.StateFailedToStart)
            vm.writeAndSendInfo()
        }
    }
}
```

**Benefits**:
- Automatic recovery from crashes
- No resource leaks
- Clear failure state for operators
- Existing cleanup mechanisms can handle failed VMs

**Trade-offs**:
- Adds one extra disk write per async VM creation
- VMs in progress during crash are marked as failed (safe, conservative approach)
- Operators can manually retry creation if needed

## Implementation Plan

1. **Phase 1**: Extract common logic (Solution 1)
   - Create `createVmInternal()` function
   - Refactor `createVm()` to use it
   - Refactor `createVmAsyncBackground()` to use it
   - Test both sync and async paths

2. **Phase 2**: Add crash recovery (Solution 2)
   - Add early `writeInfo()` call in `createVmAsync()`
   - Add `recoverIncompleteVms()` to startup
   - Test crash scenarios

3. **Phase 3**: Testing
   - Test sync CreateVm still works
   - Test async CreateVmAsync still works
   - Test crash recovery (kill hypervisor during VM creation)
   - Verify no resource leaks

## Questions for Review

1. **Progress callback approach**: Is the callback pattern acceptable, or would you prefer a different approach?

2. **Crash recovery strategy**: Should we mark incomplete VMs as `StateFailedToStart`, or try to resume them?
   - **Recommendation**: Mark as failed (safer, simpler)
   - **Alternative**: Try to resume (complex, risky)

3. **Scope**: Should we also refactor other async operations (StartVmAsync, DestroyVmAsync) to use similar patterns?

4. **Testing**: Do you have a test environment where we can test crash scenarios?

