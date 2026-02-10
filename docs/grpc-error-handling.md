# gRPC Error Handling Guide

## Overview

This document describes the error handling strategy for gRPC services in Dominator. We use a **hybrid approach** that supports both:

1. **Typed errors** (preferred, new code) - Custom error types in `lib/errors`
2. **Pattern matching** (fallback, legacy code) - Inspects error messages

This allows gradual migration from untyped to typed errors without breaking existing code.

## Architecture

### Error Flow

```
Business Logic (rpcd.Handler)
    ↓ returns error
gRPC Handler (grpcd)
    ↓ calls ErrorToGRPCStatus()
ErrorToGRPCStatus()
    ↓ checks typed errors first (errors.As)
    ↓ falls back to pattern matching
    ↓ returns gRPC status.Error()
Client
    ↓ receives proper gRPC status code
```

### Key Components

1. **`lib/errors/types.go`** - Typed error definitions
2. **`lib/grpc/errors.go`** - Error to gRPC status mapper
3. **grpcd handlers** - Use `ErrorToStatus()` instead of returning error strings

## Using Typed Errors (Preferred)

### In Business Logic

```go
// fleetmanager/rpcd/getMachineInfo.go
import "github.com/Cloud-Foundations/Dominator/lib/errors"

func (h *Handler) GetMachineInfo(request fm_proto.GetMachineInfoRequest) (
    fm_proto.GetMachineInfoResponse, error) {
    
    machine, err := h.hypervisorsManager.GetMachineInfo(request)
    if err != nil {
        // Return typed error instead of generic error
        return fm_proto.GetMachineInfoResponse{}, 
            errors.NewNotFoundError("machine", request.Hostname)
    }
    // ...
}
```

### In gRPC Handlers

```go
// fleetmanager/grpcd/getMachineInfo.go
import lib_grpc "github.com/Cloud-Foundations/Dominator/lib/grpc"

func (s *server) GetMachineInfo(ctx context.Context,
    req *pb.GetMachineInfoRequest) (*pb.GetMachineInfoResponse, error) {

    internalReq := pb.GetMachineInfoRequestToFleetManager(req)
    internalResp, err := s.handler.GetMachineInfo(*internalReq)
    if err != nil {
        // Convert to proper gRPC status code
        return nil, lib_grpc.ErrorToStatus(err)
    }

    return pb.GetMachineInfoResponseFromFleetManager(&internalResp), nil
}
```

## Available Typed Errors

| Error Type | gRPC Code | Use Case |
|------------|-----------|----------|
| `NotFoundError` | `NotFound` | Resource doesn't exist |
| `PermissionDeniedError` | `PermissionDenied` | Caller lacks permission |
| `UnauthenticatedError` | `Unauthenticated` | Caller not authenticated |
| `AlreadyExistsError` | `AlreadyExists` | Resource already exists |
| `InvalidArgumentError` | `InvalidArgument` | Invalid input parameter |
| `UnavailableError` | `Unavailable` | Service temporarily down |
| `DeadlineExceededError` | `DeadlineExceeded` | Operation timed out |

### Creating Typed Errors

```go
// NotFoundError
errors.NewNotFoundError("machine", "host1")
// → "machine host1 not found" → codes.NotFound

// PermissionDeniedError
errors.NewPermissionDeniedError("machine", "update", "not owner")
// → "permission denied: update on machine: not owner" → codes.PermissionDenied

// InvalidArgumentError
errors.NewInvalidArgumentError("hostname", "cannot be empty")
// → "invalid argument hostname: cannot be empty" → codes.InvalidArgument
```

## Pattern Matching Fallback (Legacy)

For existing code that uses `errors.New()` or `fmt.Errorf()`, the mapper automatically detects common patterns:

| Pattern | gRPC Code |
|---------|-----------|
| "not found", "does not exist" | `NotFound` |
| "permission denied", "access denied" | `PermissionDenied` |
| "unauthenticated", "not authenticated" | `Unauthenticated` |
| "already exists", "duplicate" | `AlreadyExists` |
| "invalid", "malformed" | `InvalidArgument` |
| "unavailable", "service down" | `Unavailable` |
| "timeout", "deadline exceeded" | `DeadlineExceeded` |
| (anything else) | `Internal` |

## Migration Strategy

### Phase 1: Update gRPC Handlers (Immediate)

1. Remove `error` field from proto response messages
2. Update grpcd handlers to use `ErrorToStatus()`
3. Existing errors will be mapped via pattern matching

### Phase 2: Migrate to Typed Errors (Gradual)

As you touch business logic code:

1. Replace `errors.New()` with typed errors
2. Update error messages to use typed error constructors
3. Tests will automatically benefit from proper error codes

### Example Migration

**Before:**
```go
if machine == nil {
    return fm_proto.GetMachineInfoResponse{}, 
        errors.New("machine not found")  // Untyped
}
```

**After:**
```go
if machine == nil {
    return fm_proto.GetMachineInfoResponse{}, 
        errors.NewNotFoundError("machine", hostname)  // Typed
}
```

## Benefits

✅ **Proper HTTP status codes** - grpc-gateway returns 404, 403, etc.  
✅ **Better client experience** - Standard gRPC error handling works  
✅ **Interceptors work** - Logging, metrics, tracing see errors  
✅ **Gradual migration** - No big-bang rewrite needed  
✅ **Type safety** - Typed errors are easier to test and maintain  

## Testing

```go
func TestGetMachineInfo_NotFound(t *testing.T) {
    // Business logic returns typed error
    err := handler.GetMachineInfo(request)
    
    var notFound *errors.NotFoundError
    if !errors.As(err, &notFound) {
        t.Fatal("expected NotFoundError")
    }
    
    // gRPC handler converts to status
    grpcErr := lib_grpc.ErrorToStatus(err)
    st, _ := status.FromError(grpcErr)
    if st.Code() != codes.NotFound {
        t.Errorf("expected NotFound, got %v", st.Code())
    }
}
```

## SRPC Compatibility

SRPC handlers continue to work as before - they put errors in the response struct:

```go
func (t *srpcType) GetMachineInfo(conn *srpc.Conn,
    request fm_proto.GetMachineInfoRequest,
    reply *fm_proto.GetMachineInfoResponse) error {
    
    if response, err := t.Handler.GetMachineInfo(request); err != nil {
        *reply = fm_proto.GetMachineInfoResponse{
            Error: errors.ErrorToString(err)}  // Still works
    } else {
        *reply = response
    }
    return nil
}
```

The internal `GetMachineInfoResponse` struct keeps its `Error` field for SRPC compatibility.

