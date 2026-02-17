# gRPC Minimal Implementation Plan

## Goal
Create a minimal, clean gRPC implementation for easier review. Focus on **hypervisor service only** with just 5 essential APIs.

## Scope

### In Scope: Hypervisor APIs (5 APIs)
1. **CreateVmAsync** - Asynchronous VM creation (ImageName/ImageURL only, no streaming)
2. **StartVm** - Synchronous VM start
3. **DestroyVm** - Synchronous VM destruction
4. **ListVMs** - List all VMs
5. **GetVmInfo** - Get info for a specific VM

### Out of Scope (Deferred)
- ❌ FleetManager gRPC (defer to separate PR)
- ❌ Streaming APIs (CreateVm bidirectional streaming)
- ❌ Async variants (StartVmAsync, DestroyVmAsync)
- ❌ Other hypervisor APIs (GetUpdates, StopVm, RebootVm, etc.)
- ❌ REST/grpc-gateway support (defer to separate PR)
- ❌ Complex error handling patterns
- ❌ Background goroutine crash recovery

## Implementation Checklist

### Phase 1: Infrastructure
- [ ] Add gRPC dependencies to go.mod
- [ ] Create proto files structure
  - [ ] `proto/hypervisor/grpc/hypervisor.proto` (minimal)
  - [ ] `proto/common/grpc/common.proto` (shared types)
- [ ] Create code generation script
  - [ ] `scripts/generate-grpc.sh`
  - [ ] Add `make generate-grpc` target
- [ ] Create basic gRPC server setup
  - [ ] `lib/grpc/api.go` (minimal server setup)
  - [ ] `lib/grpc/errors.go` (basic error conversion)

### Phase 2: Proto Definitions (Minimal)
- [ ] Define 5 RPC methods in hypervisor.proto
- [ ] Define only the message types needed for these 5 APIs
- [ ] Reuse existing SRPC types where possible (no duplication)
- [ ] Generate Go code

### Phase 3: Hypervisor Handlers
- [ ] Implement gRPC server in `hypervisor/rpcd/api.go`
- [ ] Implement 5 handlers:
  - [ ] `CreateVmAsync` - Call existing Manager.CreateVmAsync()
  - [ ] `StartVm` - Call existing Manager.StartVm()
  - [ ] `DestroyVm` - Call existing Manager.DestroyVm()
  - [ ] `ListVMs` - Call existing Manager.ListVMs()
  - [ ] `GetVmInfo` - Call existing Manager.GetVmInfo()
- [ ] Add basic proto↔SRPC converters (inline, no separate file)

### Phase 4: Testing
- [ ] Build verification
- [ ] Basic smoke tests
- [ ] Update documentation

## Key Simplifications

### 1. No Async Variants (Except CreateVmAsync)
- Only CreateVmAsync (because it's the primary use case)
- StartVm and DestroyVm are synchronous only
- Simpler to review and understand

### 2. No Streaming
- CreateVmAsync only supports ImageName/ImageURL
- No bidirectional streaming CreateVm
- Can add later when needed

### 3. Minimal Error Handling
- Basic ErrorToStatus conversion
- No complex typed errors
- No custom error codes

### 4. No REST/grpc-gateway
- Pure gRPC only
- Can add REST layer later

### 5. Reuse SRPC Business Logic
- All handlers just call existing Manager methods
- No code duplication
- No new business logic

### 6. No FleetManager
- Focus on hypervisor only
- FleetManager can be separate PR

## File Structure

```
Dominator/
├── proto/
│   ├── common/grpc/
│   │   └── common.proto          # Shared types (IP, Timestamp, etc.)
│   └── hypervisor/grpc/
│       └── hypervisor.proto       # 5 RPC methods only
├── lib/grpc/
│   ├── api.go                     # Basic server setup
│   └── errors.go                  # Basic error conversion
├── hypervisor/rpcd/
│   ├── api.go                     # gRPC server registration
│   ├── createVm.go                # CreateVmAsync handler
│   ├── startVm.go                 # StartVm handler (add gRPC)
│   ├── destroyVm.go               # DestroyVm handler (add gRPC)
│   ├── listVMs.go                 # ListVMs handler (add gRPC)
│   └── getVmInfo.go               # GetVmInfo handler (add gRPC)
├── hypervisor/manager/
│   ├── api.go                     # Add CreateVmAsync() method
│   └── vm.go                      # Add createVmAsync() implementation
└── scripts/
    └── generate-grpc.sh           # Proto code generation
```

## Benefits of Minimal Approach

1. **Easier Review**: ~500 lines instead of ~5000 lines
2. **Faster Iteration**: Get feedback quickly
3. **Incremental**: Can add more APIs in follow-up PRs
4. **Lower Risk**: Smaller changes, easier to test
5. **Clear Scope**: Reviewers know exactly what to focus on

## Next Steps After Review

Once minimal implementation is approved:
1. Add more hypervisor APIs (GetUpdates, StopVm, etc.)
2. Add async variants (StartVmAsync, DestroyVmAsync)
3. Add streaming APIs (CreateVm bidirectional)
4. Add FleetManager gRPC
5. Add REST/grpc-gateway support
6. Add crash recovery for background goroutines

