# gRPC Implementation Guide for Dominator Services

## Overview

This document describes the architectural patterns and design decisions for implementing gRPC support in Dominator services. It covers how gRPC coexists with SRPC, how types are organized, error handling, authentication/authorization, and streaming patterns.

**Key Principle:** gRPC is implemented as an **additional transport layer** alongside SRPC, reusing existing business logic and authentication infrastructure.

---

## Table of Contents

1. [Error Handling](#1-error-handling)
2. [Authentication & Authorization (RBAC)](#2-authentication--authorization-rbac)
3. [Streaming Patterns](#3-streaming-patterns)
4. [Type Organization](#4-type-organization)
5. [SRPC Code Reuse & Converters](#5-srpc-code-reuse--converters)

---

## 1. Error Handling

### 1.1 Hybrid Typed Error Approach

Dominator uses a **hybrid error handling strategy** that supports both typed errors (preferred) and pattern matching (legacy fallback).

**Architecture:**

```
Business Logic (hypervisors.Manager)
    ↓ returns error
gRPC Handler (rpcd)
    ↓ calls lib_grpc.ErrorToStatus()
lib_grpc.ErrorToStatus()
    ↓ checks typed errors first (errors.As)
    ↓ falls back to pattern matching
    ↓ returns gRPC status.Error()
Client
    ↓ receives proper gRPC status code
```

### 1.2 Typed Errors (Preferred)

**Define typed errors in `lib/errors/types.go`:**

```go
type NotFoundError struct {
    ResourceType string
    ResourceID   string
}

func (e *NotFoundError) Error() string {
    return fmt.Sprintf("%s %s not found", e.ResourceType, e.ResourceID)
}

func (e *NotFoundError) GRPCCode() codes.Code {
    return codes.NotFound
}
```

**Use in business logic:**

```go
func (m *Manager) GetMachineInfo(hostname string) (*MachineInfo, error) {
    machine := m.findMachine(hostname)
    if machine == nil {
        return nil, errors.NewNotFoundError("machine", hostname)
    }
    return machine, nil
}
```

### 1.3 gRPC Handler Pattern

**Always use `lib_grpc.ErrorToStatus()` in gRPC handlers:**

```go
func (s *grpcServer) GetMachineInfo(ctx context.Context,
    req *pb.GetMachineInfoRequest) (*pb.GetMachineInfoResponse, error) {

    info, err := s.manager.GetMachineInfo(req.Hostname)
    if err != nil {
        return nil, lib_grpc.ErrorToStatus(err)  // Converts to gRPC status
    }
    return machineInfoToProto(info), nil
}
```

### 1.4 Error Handling: Unary vs Streaming

**Unary RPCs:** Return errors via gRPC status codes

```go
func (s *grpcServer) GetMachineInfo(ctx context.Context,
    req *pb.GetMachineInfoRequest) (*pb.GetMachineInfoResponse, error) {

    info, err := s.manager.GetMachineInfo(req.Hostname)
    if err != nil {
        return nil, lib_grpc.ErrorToStatus(err)  // ✅ Return via status
    }
    return machineInfoToProto(info), nil
}
```

**Streaming RPCs:** Return errors via gRPC status codes, **NOT in message fields**

```go
func (s *grpcServer) GetUpdates(req *pb.GetUpdatesRequest,
    stream pb.FleetManager_GetUpdatesServer) error {

    for {
        update, ok := <-updateChannel
        if !ok {
            // ✅ Return error via gRPC status
            return lib_grpc.ErrorToStatus(fmt.Errorf("channel closed"))
        }
        if err := stream.Send(update); err != nil {
            return err  // ✅ gRPC handles this
        }
    }
}
```

**❌ Do NOT add error fields to streaming message types:**

```protobuf
// ❌ WRONG - Don't do this
message Update {
  repeated Machine changed_machines = 1;
  string error = 2;  // ❌ Remove this
}

// ✅ CORRECT - No error field
message Update {
  repeated Machine changed_machines = 1;
  map<string, VmInfo> changed_vms = 2;
}
```

### 1.5 SRPC Compatibility

SRPC handlers continue to use error fields in response structs:

```go
// SRPC handler - still uses error field
func (t *srpcType) GetMachineInfo(conn *srpc.Conn,
    request proto.GetMachineInfoRequest,
    reply *proto.GetMachineInfoResponse) error {

    if response, err := t.manager.GetMachineInfo(request.Hostname); err != nil {
        *reply = proto.GetMachineInfoResponse{
            Error: err.Error()}  // ✅ SRPC uses error field
    } else {
        *reply = response
    }
    return nil
}
```

**Key Point:** SRPC and gRPC error handling patterns can diverge. The internal `messages.go` types keep error fields for SRPC compatibility.

---

## 2. Authentication & Authorization (RBAC)

### 2.1 TLS Client Certificates

Dominator uses **TLS client certificates** for authentication, shared between SRPC and gRPC.

**Certificate contains:**
- Username
- Groups
- Permitted methods (optional - for method-level authorization)

### 2.2 gRPC Interceptors

**Interceptors extract auth from TLS certificates and inject into context:**

```go
// lib/grpc/interceptors.go

// UnaryAuthInterceptor for unary RPCs
func UnaryAuthInterceptor(ctx context.Context, req interface{},
    info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

    conn, err := extractConn(ctx)
    if err != nil {
        return nil, status.Error(codes.Unauthenticated, err.Error())
    }
    ctx = context.WithValue(ctx, connKey, conn)
    return handler(ctx, req)
}

// StreamAuthInterceptor for streaming RPCs
func StreamAuthInterceptor(srv interface{}, ss grpc.ServerStream,
    info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {

    conn, err := extractConn(ss.Context())
    if err != nil {
        return status.Error(codes.Unauthenticated, err.Error())
    }
    wrapped := &wrappedStream{
        ServerStream: ss,
        ctx:          context.WithValue(ss.Context(), connKey, conn),
    }
    return handler(srv, wrapped)
}
```

### 3.3 Auth Extraction from TLS

**Reuses SRPC's auth extraction logic:**

```go
func extractConn(ctx context.Context) (*Conn, error) {
    p, ok := peer.FromContext(ctx)
    if !ok {
        return nil, errors.New("no peer info in context")
    }
    tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
    if !ok {
        return nil, errors.New("peer auth is not TLS")
    }

    // ✅ Reuse SRPC's auth extraction from TLS certificates
    authInfo, err := srpc.GetAuthFromTLS(tlsInfo.State)
    if err != nil {
        return nil, err
    }

    // Also get permitted methods if available
    permittedMethods, _ := srpc.GetPermittedMethodsFromTLS(tlsInfo.State)

    return &Conn{
        authInfo:         authInfo,
        permittedMethods: permittedMethods,
    }, nil
}
```

### 3.4 Using Auth in Handlers

**Handlers retrieve auth from context:**

```go
func (s *grpcServer) ChangeMachineTags(ctx context.Context,
    req *pb.ChangeMachineTagsRequest) (*pb.ChangeMachineTagsResponse, error) {

    // ✅ Get auth from context
    conn := lib_grpc.GetConn(ctx)
    authInfo := conn.GetAuthInformation()

    // Pass to business logic for authorization checks
    err := s.manager.ChangeMachineTags(req.Hostname, authInfo, tags)
    if err != nil {
        return nil, lib_grpc.ErrorToStatus(err)
    }
    return &pb.ChangeMachineTagsResponse{}, nil
}
```

### 2.5 lib/grpc.Conn API

**Mirrors `srpc.Conn` for code reuse:**

```go
// lib/grpc/api.go
type Conn struct {
    authInfo         *srpc.AuthInformation
    permittedMethods map[string]struct{}
}

func (c *Conn) GetAuthInformation() *srpc.AuthInformation {
    return c.authInfo
}

func (c *Conn) GetPermittedMethods() map[string]struct{} {
    return c.permittedMethods
}
```

**This allows business logic to accept either `*srpc.Conn` or `*lib_grpc.Conn`** (or just `*srpc.AuthInformation` for maximum flexibility).

### 2.6 Server Setup with Interceptors

```go
// cmd/fleet-manager/main.go
import (
    lib_grpc "github.com/Cloud-Foundations/Dominator/lib/grpc"
)

grpcServer := grpc.NewServer(
    grpc.Creds(credentials.NewTLS(tlsConfig)),
    grpc.ChainUnaryInterceptor(
        lib_grpc.UnaryAuthInterceptor,  // ✅ Auth extraction
        // ... other interceptors (logging, metrics, etc.)
    ),
    grpc.ChainStreamInterceptor(
        lib_grpc.StreamAuthInterceptor,  // ✅ Auth extraction for streams
        // ... other interceptors
    ),
)
```

---

## 4. Type Organization

### 4.1 Proto File Structure

Proto files are organized by service ownership, mirroring the existing SRPC structure:

```
proto/
├── common/
│   └── grpc/
│       └── common.proto          # Truly cross-cutting types (Tags, MatchTags)
├── hypervisor/
│   ├── messages.go               # SRPC types (source of truth)
│   └── grpc/
│       └── hypervisor.proto      # gRPC types owned by hypervisor
├── fleetmanager/
│   ├── messages.go               # SRPC types (source of truth)
│   └── grpc/
│       └── fleetmanager.proto    # gRPC types, imports hypervisor types
└── <service>/
    ├── messages.go               # SRPC types
    └── grpc/
        └── <service>.proto       # gRPC types
```

### 4.2 Direct Import Pattern

**Pattern:** Services import proto types from each other when there's a clear ownership relationship.

**Example:** FleetManager imports types from Hypervisor:

```protobuf
// proto/fleetmanager/grpc/fleetmanager.proto
syntax = "proto3";
package fleetmanager;

import "hypervisor/grpc/hypervisor.proto";
import "common/grpc/common.proto";

message GetMachineInfoResponse {
  string location = 1;
  Machine machine = 2;
  repeated hypervisor.Subnet subnets = 3;  // Import from hypervisor
}
```

**Rationale:**
- ✅ Mirrors existing SRPC pattern (fleetmanager imports hypervisor types)
- ✅ Clear ownership: hypervisor owns `VmInfo`, `Subnet`, etc.
- ✅ Avoids "common dumping ground" anti-pattern
- ✅ Single source of truth for domain types

### 4.3 Common Types

**Only create `common/` types for truly cross-cutting concerns**, not domain models.

**Examples of appropriate common types:**
- `Tags` (map<string, string>) - Used by all services
- `MatchTags` (map<string, StringList>) - Used for filtering across services

**Examples of types that should NOT be in common:**
- `VmInfo` - Owned by hypervisor service
- `Machine` - Owned by fleetmanager service
- `Subnet` - Owned by hypervisor service

### 4.4 Go Package Naming

Use descriptive package names to avoid collisions:

```protobuf
option go_package = "github.com/Cloud-Foundations/Dominator/proto/hypervisor/grpc;hypervisor_grpc";
option go_package = "github.com/Cloud-Foundations/Dominator/proto/common/grpc;common_grpc";
```

**Pattern:** `<service>_grpc` suffix for clarity in imports.

---

## 5. SRPC Code Reuse & Converters

### 5.1 Transport Layer Consolidation

**Pattern:** Both SRPC and gRPC handlers live in the same `rpcd` package, side-by-side in the same files.

**File structure:**

```
fleetmanager/
└── rpcd/
    ├── api.go                    # Setup for both SRPC and gRPC
    ├── getMachineInfo.go         # Both SRPC and gRPC handlers
    ├── getUpdates.go             # Both SRPC and gRPC handlers
    └── changeMachineTags.go      # Both SRPC and gRPC handlers
```

**Example (`getMachineInfo.go`):**

```go
// SRPC handler
func (t *srpcType) GetMachineInfo(conn *srpc.Conn,
    request proto.GetMachineInfoRequest,
    reply *proto.GetMachineInfoResponse) error {

    // Call shared business logic
    info, err := t.manager.GetMachineInfo(request.Hostname)
    if err != nil {
        *reply = proto.GetMachineInfoResponse{Error: err.Error()}
    } else {
        *reply = *info
    }
    return nil
}

// gRPC handler (same file)
func (s *grpcServer) GetMachineInfo(ctx context.Context,
    req *pb.GetMachineInfoRequest) (*pb.GetMachineInfoResponse, error) {

    // Call same shared business logic
    info, err := s.manager.GetMachineInfo(req.Hostname)
    if err != nil {
        return nil, lib_grpc.ErrorToStatus(err)
    }
    return machineInfoToProto(info), nil
}
```

### 5.2 Converter Patterns

**Converters translate between internal types and gRPC proto types.**

#### 5.2.1 Converter Naming Convention

**Standard pattern:** `<type>ToProto` (converting TO proto format)

```go
func machineToProto(m *Machine) *pb.Machine { ... }
func vmInfoToProto(v *VmInfo) *hypervisor_grpc.VmInfo { ... }
func subnetToProto(s *Subnet) *hypervisor_grpc.Subnet { ... }
```

#### 5.2.2 Converter Location

**Co-locate converters with their usage:**

- **Single use:** Define converter in the same file where it's used
- **Multiple uses within same package:** Define in shared file (e.g., `getUpdates.go` if used by multiple streaming handlers)
- **Cross-service common types:** Define in `lib/grpc/common_converters.go`

**Example:**

```
lib/grpc/
└── common_converters.go          # TagsToProto, MatchTagsToProto

fleetmanager/rpcd/
├── getMachineInfo.go             # machineToProto (used here only)
└── getUpdates.go                 # vmInfoToProto (shared by multiple methods)
```

#### 4.2.3 Idiomatic Go Converter Pattern

**Return values, don't modify pointers:**

```go
// ✅ CORRECT - Return value pattern
func machineToProto(m *Machine) *pb.Machine {
    if m == nil {
        return nil
    }
    return &pb.Machine{
        Hostname: m.Hostname,
        Tags:     lib_grpc.TagsToProto(m.Tags),
    }
}

// ❌ WRONG - Don't modify pointer in-place
func machineToProto(m *Machine, pb *pb.Machine) {
    pb.Hostname = m.Hostname
    // ...
}
```

#### 4.2.4 Common Converters

**Shared converters in `lib/grpc/common_converters.go`:**

```go
// Tags conversion
func TagsToProto(t tags.Tags) *common_grpc.Tags {
    if t == nil {
        return nil
    }
    return &common_grpc.Tags{Tags: t}
}

func TagsFromProto(pb *common_grpc.Tags) tags.Tags {
    if pb == nil || pb.Tags == nil {
        return nil
    }
    return tags.Tags(pb.Tags)
}

// MatchTags conversion
func MatchTagsToProto(mt tags.MatchTags) *common_grpc.MatchTags { ... }
func MatchTagsFromProto(pb *common_grpc.MatchTags) tags.MatchTags { ... }
```

### 4.3 Shared Business Logic

**Business logic lives in service packages (e.g., `fleetmanager/hypervisors`), not in `rpcd`:**

```
fleetmanager/
├── hypervisors/
│   └── manager.go                # Business logic (transport-agnostic)
└── rpcd/
    ├── getMachineInfo.go         # SRPC + gRPC handlers (thin wrappers)
    └── getUpdates.go             # SRPC + gRPC handlers (thin wrappers)
```

**Handlers are thin wrappers:**

```go
// gRPC handler - thin wrapper
func (s *grpcServer) GetMachineInfo(ctx context.Context,
    req *pb.GetMachineInfoRequest) (*pb.GetMachineInfoResponse, error) {

    // 1. Extract auth
    conn := lib_grpc.GetConn(ctx)

    // 2. Call business logic
    info, err := s.manager.GetMachineInfo(req.Hostname)

    // 3. Convert and return
    if err != nil {
        return nil, lib_grpc.ErrorToStatus(err)
    }
    return machineInfoToProto(info), nil
}
```

---

## 3. Streaming Patterns

### 3.1 gRPC Watch Pattern (Standard)

**For real-time updates (like Kubernetes, etcd):**

```protobuf
// Standard gRPC watch pattern - client controls stream lifetime via context
rpc GetUpdates(GetUpdatesRequest) returns (stream Update) {}

message GetUpdatesRequest {
  bool ignore_missing_local_tags = 1;
  string location = 2;
  // Note: No max_updates field - use context cancellation to stop the stream
}

message Update {
  repeated Machine changed_machines = 1;
  map<string, VmInfo> changed_vms = 2;
  repeated string deleted_machines = 3;
  repeated string deleted_vms = 4;
}
```

**Server implementation:**

```go
func (s *grpcServer) GetUpdates(req *pb.GetUpdatesRequest,
    stream pb.FleetManager_GetUpdatesServer) error {

    updateChannel := s.manager.MakeUpdateChannel(req)
    defer s.manager.CloseUpdateChannel(updateChannel)

    // ✅ Stream indefinitely until client cancels context
    for {
        select {
        case <-stream.Context().Done():
            // Client canceled context - standard gRPC pattern
            return stream.Context().Err()

        case update, ok := <-updateChannel:
            if !ok {
                return lib_grpc.ErrorToStatus(fmt.Errorf("channel closed"))
            }
            if err := stream.Send(convertUpdate(&update)); err != nil {
                return err
            }
        }
    }
}
```

**Client usage:**

```go
// Watch for 5 minutes
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

stream, err := client.GetUpdates(ctx, &pb.GetUpdatesRequest{Location: "us-west"})
for {
    update, err := stream.Recv()
    if err == io.EOF {
        break  // Stream ended
    }
    if err != nil {
        // Handle error
    }
    // Process update
}
```

### 3.2 Pagination Pattern (Standard for List Operations)

**For list operations (like Google Cloud, AWS, Kubernetes):**

```protobuf
rpc ListVMsInLocation(ListVMsInLocationRequest) returns (ListVMsInLocationResponse) {}

message ListVMsInLocationRequest {
  string location = 1;
  uint32 page_size = 2;      // 0 = all results
  string page_token = 3;     // Empty for first page
}

message ListVMsInLocationResponse {
  repeated VmInfo vms = 1;
  string next_page_token = 2;  // Empty if no more pages
}
```

**Server implementation:**

```go
func (s *grpcServer) ListVMsInLocation(ctx context.Context,
    req *pb.ListVMsInLocationRequest) (*pb.ListVMsInLocationResponse, error) {

    vms, nextToken, err := s.manager.ListVMsPaginated(
        req.Location, req.PageSize, req.PageToken)
    if err != nil {
        return nil, lib_grpc.ErrorToStatus(err)
    }

    return &pb.ListVMsInLocationResponse{
        Vms:           vmsToProto(vms),
        NextPageToken: nextToken,
    }, nil
}
```

### 3.3 Differences from SRPC

| Aspect | SRPC | gRPC |
|--------|------|------|
| **Error handling** | Error field in response struct | gRPC status codes |
| **Streaming control** | `MaxUpdates` field | Context cancellation |
| **List operations** | Often streaming | Pagination (unary RPC) |
| **Auth extraction** | `conn.GetAuthInformation()` | `lib_grpc.GetConn(ctx).GetAuthInformation()` |
| **Interceptors** | N/A (built into SRPC) | Explicit interceptor chain |
| **Transport** | Custom (not HTTP) | HTTP/2 |

### 3.4 When to Use Streaming vs Pagination

**Use streaming (watch pattern) for:**
- ✅ Real-time updates (watch/subscribe)
- ✅ Large file transfers
- ✅ Long-running operations with progress updates

**Use pagination (unary RPC) for:**
- ✅ List operations (list machines, list VMs, list locations)
- ✅ Search results
- ✅ Any operation where client wants a snapshot, not continuous updates

---

## Summary

### Key Architectural Decisions

1. **Error Handling:** Hybrid typed errors with gRPC status codes, pattern matching fallback for legacy code
2. **RBAC:** TLS client certificates, gRPC interceptors extract auth, reuse SRPC's auth infrastructure
3. **Streaming:** Standard gRPC watch pattern (context cancellation), pagination for list operations
4. **Type Organization:** Direct import pattern - services import from each other, `common/` only for truly cross-cutting types
5. **Code Reuse:** Both SRPC and gRPC handlers in same `rpcd` package, share business logic, thin converter layer

### Benefits of This Approach

- ✅ **Minimal duplication:** Business logic written once, shared by SRPC and gRPC
- ✅ **Consistent auth:** Same TLS certificates and auth model for both protocols
- ✅ **Gradual migration:** SRPC and gRPC coexist, no big-bang rewrite
- ✅ **Industry standards:** Follows gRPC best practices (status codes, context cancellation, pagination)
- ✅ **Type safety:** Clear ownership, direct imports, no "common dumping ground"

### Example Service: FleetManager

See `fleetmanager/rpcd/` for a complete implementation following these patterns.

---

## References

- [gRPC Error Handling Guide](./grpc-error-handling.md) - Detailed error handling patterns
- [gRPC + grpc-gateway Architecture Decision](./grpc-rest-architecture-decision.md) - Why we chose gRPC + grpc-gateway
- [lib/grpc Package](../lib/grpc/) - Common gRPC infrastructure
- [FleetManager rpcd](../fleetmanager/rpcd/) - Example implementation

