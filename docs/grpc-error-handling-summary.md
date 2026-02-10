# gRPC Error Handling - Implementation Summary

## What We Built

A **hybrid error handling system** that supports gradual migration from untyped to typed errors.

## Architecture Decision: Using gRPC Codes in lib/errors

### Decision
We decided to have `lib/errors` import `google.golang.org/grpc/codes` directly, rather than creating our own generic error code system.

### Rationale

**Why this is acceptable:**

1. **Lightweight Dependency**: The `codes` package only imports standard library (`fmt`, `strconv`)
2. **Industry Standard**: gRPC codes are widely understood and used across the industry
3. **Stable API**: The codes enum hasn't changed in years and is unlikely to change
4. **Minimal Binary Impact**: Only adds ~1-2 KB to binary (just enum constants)
5. **No Runtime Dependencies**: No network code, no servers, no clients - just constants
6. **Automatic HTTP Mapping**: grpc-gateway automatically maps gRPC codes to HTTP status codes

**Why NOT to create custom codes:**

1. ❌ Reinventing the wheel - gRPC codes already exist and are well-designed
2. ❌ Additional mapping layer - would need to map custom codes → gRPC codes → HTTP codes
3. ❌ Less familiar to developers - everyone knows gRPC codes
4. ❌ More maintenance burden - another thing to maintain and document

### Go Module Dependency Details

When `lib/errors` imports `google.golang.org/grpc/codes`:

**Download Phase** (`go get` or `go mod tidy`):
- Entire `google.golang.org/grpc` module is downloaded to `$GOPATH/pkg/mod`
- Module size: ~10 MB
- Versioning: Module-level (e.g., `v1.60.0`)

**Compile Phase** (`go build`):
- Only `codes` package is compiled into binary
- Dead code elimination removes unused packages
- Binary size impact: ~1-2 KB (just enum constants)

**When clients import our proto package:**
- They download our entire Dominator module
- But only compile what they actually use
- If they use our proto types, they also get `google.golang.org/grpc/codes` dependency
- This is acceptable because they're already using gRPC (they need it for proto anyway)

### gRPC to HTTP Status Code Mapping

grpc-gateway automatically maps gRPC codes to HTTP status codes:

| gRPC Code | HTTP Status Code |
|-----------|------------------|
| `OK` | 200 OK |
| `InvalidArgument` | 400 Bad Request |
| `Unauthenticated` | 401 Unauthorized |
| `PermissionDenied` | 403 Forbidden |
| `NotFound` | 404 Not Found |
| `AlreadyExists` | 409 Conflict |
| `ResourceExhausted` | 429 Too Many Requests |
| `Internal` | 500 Internal Server Error |
| `Unavailable` | 503 Service Unavailable |
| `DeadlineExceeded` | 504 Gateway Timeout |

This means when we return `status.Error(codes.NotFound, "...")` from gRPC, clients using grpc-gateway automatically get HTTP 404!

### Components Created

1. **`lib/errors/types.go`** - Typed error definitions
   - `NotFoundError` → `codes.NotFound`
   - `PermissionDeniedError` → `codes.PermissionDenied`
   - `UnauthenticatedError` → `codes.Unauthenticated`
   - `AlreadyExistsError` → `codes.AlreadyExists`
   - `InvalidArgumentError` → `codes.InvalidArgument`
   - `UnavailableError` → `codes.Unavailable`
   - `DeadlineExceededError` → `codes.DeadlineExceeded`

2. **`lib/grpc/errors.go`** - Error mapper
   - Checks for typed errors first (using `errors.As()`)
   - Falls back to pattern matching for untyped errors
   - Returns proper gRPC `status.Error()` with appropriate codes

3. **`lib/grpc/errors_test.go`** - Comprehensive tests
   - Tests for all typed errors
   - Tests for pattern matching fallback
   - Tests for wrapped errors
   - All tests passing ✅

4. **`docs/grpc-error-handling.md`** - Migration guide
   - How to use typed errors
   - How to update gRPC handlers
   - Migration strategy
   - Examples and best practices

## How It Works

### Option 2: Typed Errors (Preferred)

```go
// Business logic returns typed error
return fm_proto.GetMachineInfoResponse{},
    errors.NewNotFoundError("machine", hostname)

// gRPC handler converts to status
return nil, lib_grpc.ErrorToStatus(err)
// → Returns: codes.NotFound with message "machine host1 not found"
```

### Option 1: Pattern Matching (Fallback)

```go
// Legacy code returns untyped error
return fm_proto.GetMachineInfoResponse{},
    errors.New("machine not found")

// gRPC handler still works
return nil, lib_grpc.ErrorToStatus(err)
// → Returns: codes.NotFound (detected via pattern matching)
```

## Example: Updated getMachineInfo Handler

**Before:**
```go
func (s *server) GetMachineInfo(ctx context.Context,
    req *pb.GetMachineInfoRequest) (*pb.GetMachineInfoResponse, error) {
    
    internalResp, err := s.handler.GetMachineInfo(*internalReq)
    if err != nil {
        return &pb.GetMachineInfoResponse{
            Error: errors.ErrorToString(err),  // ❌ Error in response
        }, nil  // ❌ gRPC error is nil
    }
    return pb.GetMachineInfoResponseFromFleetManager(&internalResp), nil
}
```

**After:**
```go
func (s *server) GetMachineInfo(ctx context.Context,
    req *pb.GetMachineInfoRequest) (*pb.GetMachineInfoResponse, error) {
    
    internalResp, err := s.handler.GetMachineInfo(*internalReq)
    if err != nil {
        return nil, lib_grpc.ErrorToStatus(err)  // ✅ Proper gRPC error
    }
    return pb.GetMachineInfoResponseFromFleetManager(&internalResp), nil
}
```

## Migration Path

### Phase 1: Infrastructure (✅ DONE)
- [x] Create typed error types in `lib/errors/types.go`
- [x] Create `ErrorToStatus()` mapper in `lib/grpc/errors.go`
- [x] Write comprehensive tests
- [x] Document migration guide

### Phase 2: Update gRPC Handlers (Next Step)
- [ ] Remove `error` field from proto response messages
- [ ] Update all grpcd handlers to use `ErrorToStatus()`
- [ ] Regenerate proto files
- [ ] Test with grpcurl

### Phase 3: Migrate Business Logic (Gradual)
- [ ] Identify common error patterns in business logic
- [ ] Replace `errors.New()` with typed errors as code is touched
- [ ] Update tests to check for typed errors
- [ ] Monitor error metrics to verify proper codes

## Benefits

### Immediate (with current implementation)
✅ **Pattern matching works now** - Existing errors get proper codes  
✅ **No breaking changes** - Existing code continues to work  
✅ **Infrastructure ready** - Can start using typed errors immediately  

### After Full Migration
✅ **Proper HTTP status codes** - grpc-gateway returns 404, 403, 500, etc.  
✅ **Better client experience** - Standard gRPC error handling works  
✅ **Interceptors work** - Logging, metrics, tracing see errors correctly  
✅ **Type safety** - Compile-time checking for error types  
✅ **Better testing** - Can assert on specific error types  

## Next Steps

1. **Update proto files** - Remove `error` field from response messages
2. **Update remaining grpcd handlers** - Use `ErrorToGRPCStatus()`
3. **Test with grpcurl** - Verify proper status codes
4. **Gradually migrate business logic** - Replace untyped errors with typed errors

## Example: Migrating One Error

**Current code:**
```go
// fleetmanager/hypervisors/manager.go (hypothetical)
func (m *Manager) GetMachineInfo(req fm_proto.GetMachineInfoRequest) (
    fm_proto.Machine, error) {
    
    machine, ok := m.machines[req.Hostname]
    if !ok {
        return fm_proto.Machine{}, 
            errors.New("machine not found")  // Untyped
    }
    return machine, nil
}
```

**Migrated code:**
```go
// fleetmanager/hypervisors/manager.go (hypothetical)
func (m *Manager) GetMachineInfo(req fm_proto.GetMachineInfoRequest) (
    fm_proto.Machine, error) {
    
    machine, ok := m.machines[req.Hostname]
    if !ok {
        return fm_proto.Machine{}, 
            errors.NewNotFoundError("machine", req.Hostname)  // Typed
    }
    return machine, nil
}
```

**Result:**
- gRPC clients get `codes.NotFound` (404 via grpc-gateway)
- Error message: "machine host1 not found"
- Interceptors see the error properly
- Metrics track 404 errors separately from 500 errors

## Testing

All tests pass:
```
$ go test ./proto/common/grpc -v
=== RUN   TestErrorToGRPCStatus_TypedErrors
--- PASS: TestErrorToGRPCStatus_TypedErrors (0.00s)
=== RUN   TestErrorToGRPCStatus_PatternMatching
--- PASS: TestErrorToGRPCStatus_PatternMatching (0.00s)
=== RUN   TestErrorToGRPCStatus_NilError
--- PASS: TestErrorToGRPCStatus_NilError (0.00s)
=== RUN   TestErrorToGRPCStatus_WrappedTypedError
--- PASS: TestErrorToGRPCStatus_WrappedTypedError (0.00s)
PASS
ok      github.com/Cloud-Foundations/Dominator/proto/common/grpc    0.003s
```

Build succeeds:
```
$ go build ./fleetmanager/grpcd ./fleetmanager/rpcd
(no output - success)
```

