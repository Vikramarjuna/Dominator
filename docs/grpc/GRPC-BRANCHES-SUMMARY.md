# gRPC Implementation Branches Summary

**Date:** 2026-03-26  
**Current Branch:** master (with proto-gen tool)

---

## Overview

Multiple experimental gRPC implementation branches exist in the Dominator repository. Each explores different approaches and features. This document catalogs them to avoid re-implementing existing work.

---

## Branch Inventory

### 1. `grpc-minimal`
**Status:** Hand-coded proto and converters for hypervisor

**What it contains:**
- ✅ Hand-written proto files (`proto/hypervisor/grpc/hypervisor.proto`)
- ✅ Hand-written converters (`hypervisor/rpcd/grpc_converters.go`)
- ✅ gRPC server implementation (`hypervisor/grpcd/`)
- ✅ Working gRPC handlers for some hypervisor methods
- ✅ Enum conversions (State → VmState, etc.)
- ✅ Complex type conversions (VmInfo, Address, etc.)

**Key learnings:**
- Demonstrates converter patterns for manual implementation
- Shows enum conversion approach
- Provides gRPC server setup example

---

### 2. `grpc-auth-core`
**Focus:** Authentication and authorization for gRPC

**What it explores:**
- RBAC integration with gRPC
- Method-level authorization
- Token-based auth

---

### 3. `grpc-copy-0`
**Focus:** Unknown (snapshot/backup branch?)

**Status:** Needs investigation

---

### 4. `grpc-errors`
**Focus:** Error handling patterns

**What it explores:**
- SRPC errors → gRPC status codes
- Error enrichment
- Custom error details

---

### 5. `grpc-infra`
**Focus:** Infrastructure and deployment

**What it explores:**
- Service discovery
- Load balancing
- Deployment patterns

---

### 6. `grpc-metrics-rest`
**Focus:** Metrics and REST gateway

**What it explores:**
- Prometheus metrics for gRPC
- HTTP/REST gateway (grpc-gateway)
- Dual protocol support

---

### 7. `grpc-streaming`
**Focus:** Streaming RPCs

**What it explores:**
- Server-side streaming
- Client-side streaming
- Bidirectional streaming

---

## Current Master Branch (Proto-Gen Approach)

**What we implemented:**
- ✅ **Proto-gen tool** - Auto-generates proto and converters from SRPC
- ✅ **AST-based parsing** - Reads Go source, extracts types
- ✅ **Recursive type discovery** - Finds all dependencies automatically
- ✅ **Cross-package imports** - Services can use types from other services
- ✅ **Tooling & CI** - Makefile targets, CI check script
- ✅ **@grpc and @http annotations** - Tag methods, not types
- ✅ **Custom method names** - `@grpc MyCustomName`

**Advantages over hand-coded approach:**
- No manual proto writing
- Converters auto-generated
- Types stay in sync automatically
- Less maintenance burden

---

## Feature Comparison

| Feature | grpc-minimal (hand-coded) | master (proto-gen) |
|---------|---------------------------|-------------------|
| Proto files | ✅ Hand-written | ✅ Auto-generated |
| Converters | ✅ Hand-written | ✅ Auto-generated |
| gRPC server | ✅ Implemented | ⚠️ TODO |
| Handlers | ✅ Some methods | ⚠️ TODO |
| @http support | ❌ No | ✅ Parsed (not generated yet) |
| Custom method names | ❌ No | ✅ Yes (`@grpc CustomName`) |
| Cross-package types | ❌ Manual | ✅ Auto-import |
| Type sync | ❌ Manual | ✅ Automatic |
| CI check | ❌ No | ✅ Yes |

---

## @http Support Status

**In proto-gen (master):**
- ✅ **Parsing implemented** - Reads `@http GET /path` annotations
- ✅ **MethodInfo populated** - HttpMethod, HttpPath, HttpBody fields
- ❌ **Generation disabled** - Requires googleapis proto files
- ❌ **grpc-gateway** - Not set up yet

**Example usage:**
```go
// @grpc
// @http GET /v1/vms/{ip_address}
func (t *srpcType) GetVmInfo(...) error
```

**Parsed to:**
```go
MethodInfo{
    Name:       "GetVmInfo",
    GrpcName:   "GetVmInfo",
    HttpMethod: "GET",
    HttpPath:   "/v1/vms/{ip_address}",
}
```

**To enable HTTP generation:**
1. Install googleapis proto files
2. Uncomment HTTP annotation generation in `generator.go`
3. Set up grpc-gateway

---

## Custom Method Name Support

**Status:** ✅ Fully implemented

**Usage:**
```go
// Use different name in gRPC
// @grpc GetVm
// @http GET /v1/vms/{ip_address}
func (t *srpcType) GetVmInfo(...) error
```

**Generated proto:**
```protobuf
service Hypervisor {
  rpc GetVm(GetVmInfoRequest) returns (GetVmInfoResponse);
}
```

**Implementation:**
- Parser extracts custom name from `@grpc CustomName`
- If no custom name, uses function name
- MethodInfo.GrpcName vs MethodInfo.Name tracks the difference

---

## Recommendations

### For New gRPC Work

**Use master branch (proto-gen approach):**
- Reduces manual work
- Keeps types in sync
- Better maintainability

**Reference grpc-minimal for:**
- gRPC server setup patterns
- Handler implementation examples
- Enum conversion patterns

### Migration Path

1. **Start with proto-gen** - Generate proto and converters automatically
2. **Copy server setup from grpc-minimal** - Adapt to generated types
3. **Copy handler patterns from grpc-minimal** - Adapt to generated converters
4. **Merge auth from grpc-auth-core** - When needed
5. **Merge metrics from grpc-metrics-rest** - When needed

### What to Avoid

**Don't re-implement:**
- Manual proto writing (use proto-gen)
- Manual converter writing (use proto-gen)
- Server setup (copy from grpc-minimal)

---

## Next Steps

**Immediate:**
1. Implement gRPC server in master (copy pattern from grpc-minimal)
2. Implement handlers using generated converters
3. Test end-to-end

**Future:**
1. Enable HTTP gateway support
2. Merge auth patterns from grpc-auth-core
3. Merge metrics from grpc-metrics-rest
4. Merge streaming patterns from grpc-streaming

---

## Branch Status Legend

- ✅ **Implemented** - Working code exists
- ⚠️ **TODO** - Planned but not done
- ❌ **Not supported** - Not implemented
- 🔍 **Needs investigation** - Unknown status

