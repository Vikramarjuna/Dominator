# gRPC/Protobuf Architecture Refactor Plan

## Problem Statement

### Current Issues

1. **Duplicate type definitions**: `proto/fleetmanager/messages.go` (Go structs for SRPC) and `proto/grpc/fleetmanager/fleetmanager.proto` (protobuf definitions) define the same concepts twice.

2. **Duplicate business logic**: Both `rpcd/getMachineInfo.go` and `grpcd/getMachineInfo.go` contain nearly identical business logic with different type conversions.

3. **Manual conversion layer**: `grpcd/converters.go` has hand-written conversion functions that must be maintained for every type.

4. **Separate proto directories**: Having `proto/grpc/fleetmanager/` separate from `proto/fleetmanager/` creates unnecessary complexity.

5. **No REST support**: Adding REST would require yet another handler layer and more converters.

## Proposed Solution

**Transport-Agnostic Architecture with grpc-gateway**

### Key Principles

1. **`messages.go` is the source of truth** - Go structs define the canonical types
2. **Generate `.proto` from Go** - No manual proto maintenance (with HTTP annotations for REST)
3. **Generate converters automatically** - No hand-written conversion code
4. **Shared service layer** - Business logic written once
5. **grpc-gateway for REST** - REST API is auto-generated from gRPC, zero additional code

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              Clients                                     │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                   │
│  │ SRPC Client  │  │ gRPC Client  │  │ REST Client  │                   │
│  │(hyper-control)│  │  (grpcurl)   │  │ (curl, web)  │                   │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘                   │
└─────────┼─────────────────┼─────────────────┼───────────────────────────┘
          │                 │                 │
          │                 │                 ▼
          │                 │        ┌─────────────────┐
          │                 │        │  grpc-gateway   │
          │                 │        │  (HTTP→gRPC)    │
          │                 │        └────────┬────────┘
          │                 │                 │
          ▼                 ▼                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          fleet-manager                                   │
│                                                                          │
│  ┌──────────────────┐    ┌──────────────────┐                           │
│  │ rpcd/            │    │ grpcd/           │◄──── REST goes through    │
│  │ SRPC Handlers    │    │ gRPC Handlers    │      gRPC (same handlers) │
│  │ (thin wrappers)  │    │ (thin wrappers)  │                           │
│  └────────┬─────────┘    └────────┬─────────┘                           │
│           │                       │                                      │
│           │            ┌──────────▼──────────┐                          │
│           │            │ converters_gen.go   │                          │
│           │            │ AUTO-GENERATED      │                          │
│           │            └──────────┬──────────┘                          │
│           │                       │                                      │
│           └───────────┬───────────┘                                      │
│                       ▼                                                  │
│           ┌─────────────────────────┐                                   │
│           │ service/                │                                   │
│           │ Shared Business Logic   │                                   │
│           └───────────┬─────────────┘                                   │
│                       ▼                                                  │
│           ┌─────────────────────────┐                                   │
│           │ hypervisors.Manager     │                                   │
│           └─────────────────────────┘                                   │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────┐
│                        Code Generation Flow                              │
│                                                                          │
│  messages.go ──► cmd/proto-gen ──┬──► fleetmanager.proto                │
│  (SOURCE)        (AST Parser)    │    (with HTTP annotations)           │
│                                  │                                       │
│                                  └──► converters_gen.go                  │
│                                                                          │
│  fleetmanager.proto ──► protoc ──┬──► fleetmanager.pb.go                │
│                          │       ├──► fleetmanager_grpc.pb.go           │
│                          │       └──► fleetmanager.pb.gw.go (REST)      │
│                          │                                               │
│                          └──► openapi/fleetmanager.swagger.json         │
└─────────────────────────────────────────────────────────────────────────┘
```

## REST via grpc-gateway

grpc-gateway is a plugin for protoc that generates a reverse-proxy server which translates RESTful HTTP API into gRPC. This means:

- **Zero additional handler code** for REST
- **Automatic OpenAPI/Swagger spec** generation
- **Battle-tested** approach used by Google, Uber, and many others

### HTTP Annotations in Proto

```protobuf
import "google/api/annotations.proto";

service FleetManager {
  rpc GetMachineInfo(GetMachineInfoRequest) returns (GetMachineInfoResponse) {
    option (google.api.http) = {
      get: "/v1/machines/{hostname}"
    };
  }

  rpc ListHypervisorLocations(ListHypervisorLocationsRequest) returns (ListHypervisorLocationsResponse) {
    option (google.api.http) = {
      get: "/v1/locations"
    };
  }

  rpc ListHypervisorsInLocation(ListHypervisorsInLocationRequest) returns (ListHypervisorsInLocationResponse) {
    option (google.api.http) = {
      get: "/v1/locations/{location}/hypervisors"
    };
  }

  rpc ChangeMachineTags(ChangeMachineTagsRequest) returns (ChangeMachineTagsResponse) {
    option (google.api.http) = {
      patch: "/v1/machines/{hostname}/tags"
      body: "*"
    };
  }
}
```

### REST Endpoints (Auto-Generated)

| Method | Endpoint | gRPC Method |
|--------|----------|-------------|
| GET | `/v1/machines/{hostname}` | GetMachineInfo |
| GET | `/v1/locations` | ListHypervisorLocations |
| GET | `/v1/locations/{location}/hypervisors` | ListHypervisorsInLocation |
| GET | `/v1/locations/{location}/vms` | ListVMsInLocation |
| GET | `/v1/vms/{ip_address}/hypervisor` | GetHypervisorForVM |
| PATCH | `/v1/machines/{hostname}/tags` | ChangeMachineTags |
| POST | `/v1/machines/{hostname}/power-on` | PowerOnMachine |
| GET | `/v1/updates` | GetUpdates (SSE/WebSocket) |

## Directory Structure

### Before
```
proto/
  fleetmanager/
    messages.go          # Go structs
    methods.go           # Helper methods
  grpc/fleetmanager/
    fleetmanager.proto   # DUPLICATE definitions
    fleetmanager.pb.go
    fleetmanager_grpc.pb.go

fleetmanager/
  rpcd/                  # Business logic + SRPC handling
  grpcd/                 # Business logic (duplicated) + gRPC handling + manual converters
```

### After
```
proto/
  fleetmanager/
    messages.go              # SOURCE OF TRUTH - Go structs
    methods.go               # Helper methods
    fleetmanager.proto       # GENERATED from messages.go
    fleetmanager.pb.go       # GENERATED by protoc
    fleetmanager_grpc.pb.go  # GENERATED by protoc

fleetmanager/
  service/                   # NEW: Shared business logic
    api.go
    getMachineInfo.go
    listHypervisors.go
    ...
  rpcd/                      # Thin SRPC wrappers
  grpcd/                     # Thin gRPC wrappers
    converters_gen.go        # GENERATED converters
```

## Type Mapping Rules

| Go Type | Protobuf Type | Notes |
|---------|---------------|-------|
| `string` | `string` | |
| `bool` | `bool` | |
| `uint`, `uint32` | `uint32` | |
| `uint64` | `uint64` | |
| `int`, `int32` | `int32` | |
| `int64` | `int64` | |
| `[]byte` | `bytes` | |
| `net.IP` | `bytes` | Converter handles IP parsing |
| `net.HardwareAddr` | `bytes` | Converter handles MAC parsing |
| `[]T` | `repeated T` | |
| `map[K]V` | `map<K, V>` | |
| `*T` | `T` | Optional in proto3 |
| `tags.Tags` | `map<string, string>` | Direct mapping |
| `tags.MatchTags` | `map<string, string>` | Note: loses `[]string` values |

## Implementation Phases

### Phase 1: Code Generator Tool
- [ ] 1.1 Design type mapping rules
- [ ] 1.2 Create `cmd/proto-gen/main.go` - parses Go AST, generates .proto files
- [ ] 1.3 Add converter generation - ToProto() and FromProto() functions
- [ ] 1.4 Extract service methods from rpcd/api.go to generate gRPC service definition
- [ ] 1.5 Add `//go:generate proto-gen` directive to messages.go

### Phase 2: Reorganize Proto Structure
- [ ] 2.1 Remove `proto/grpc/fleetmanager/` directory
- [ ] 2.2 Generate proto in `proto/fleetmanager/`
- [ ] 2.3 Run protoc to generate `.pb.go` files
- [ ] 2.4 Update all import paths

### Phase 3: Shared Service Layer
- [ ] 3.1 Create `fleetmanager/service/api.go` - Service struct and interface
- [ ] 3.2 Extract GetMachineInfo logic
- [ ] 3.3 Extract GetHypervisorForVM logic
- [ ] 3.4 Extract ListHypervisorLocations logic
- [ ] 3.5 Extract ListHypervisorsInLocation logic
- [ ] 3.6 Extract GetHypervisorsInLocation logic
- [ ] 3.7 Extract ListVMsInLocation logic
- [ ] 3.8 Extract GetUpdates logic
- [ ] 3.9 Extract ChangeMachineTags logic
- [ ] 3.10 Extract PowerOnMachine logic
- [ ] 3.11 Extract GetIpInfo logic
- [ ] 3.12 Extract MoveIpAddresses logic

### Phase 4: Refactor grpcd Handlers
- [ ] 4.1 Update `grpcd/api.go` to use service layer
- [ ] 4.2 Generate `converters_gen.go`
- [ ] 4.3 Simplify handlers: convert request → call service → convert response
- [ ] 4.4 Remove manual `converters.go`

### Phase 5: Refactor rpcd Handlers
- [ ] 5.1 Update `rpcd/api.go` to use service layer
- [ ] 5.2 Simplify handlers to thin wrappers
- [ ] 5.3 Remove duplicate business logic

### Phase 6: Testing & Validation
- [ ] 6.1 Build and fix compilation errors
- [ ] 6.2 Test SRPC endpoints with hyper-control
- [ ] 6.3 Test gRPC endpoints with grpcurl
- [ ] 6.4 Run existing unit tests
- [ ] 6.5 Add service layer tests
- [ ] 6.6 Document the architecture

## Design Decisions

### Streaming RPC Handling

`GetUpdates` and `ListVMsInLocation` are streaming RPCs. The service layer will return channels or iterators that both rpcd and grpcd can consume and stream to their respective clients.

```go
// Service layer returns a channel
func (s *Service) GetUpdates(ctx context.Context, req fm_proto.GetUpdatesRequest) (<-chan fm_proto.Update, error)

// rpcd consumes and streams via SRPC
func (t *srpcType) GetUpdates(conn *srpc.Conn, req fm_proto.GetUpdatesRequest) error {
    updates, err := t.service.GetUpdates(ctx, req)
    for update := range updates {
        conn.Encode(update)
    }
}

// grpcd consumes, converts, and streams via gRPC
func (s *server) GetUpdates(req *pb.GetUpdatesRequest, stream pb.FleetManager_GetUpdatesServer) error {
    updates, err := s.service.GetUpdates(ctx, FromProtoGetUpdatesRequest(req))
    for update := range updates {
        stream.Send(ToProtoUpdate(update))
    }
}
```

### Generator Approach

The `cmd/proto-gen` tool will:

1. Parse `proto/fleetmanager/messages.go` using `go/ast`
2. Extract all exported struct types
3. Generate `.proto` file with:
   - Field numbers based on struct field order
   - Appropriate type mappings
   - Service definition from rpcd's PublicMethods list
4. Generate `converters_gen.go` with:
   - `ToProto<Type>()` for each struct
   - `FromProto<Type>()` for each struct
   - Special handling for net.IP, HardwareAddr, tags

### Example Generated Code

**Input: messages.go**
```go
type GetMachineInfoRequest struct {
    Hostname               string
    IgnoreMissingLocalTags bool
}
```

**Output: fleetmanager.proto**
```protobuf
message GetMachineInfoRequest {
  string hostname = 1;
  bool ignore_missing_local_tags = 2;
}
```

**Output: converters_gen.go**
```go
func ToProtoGetMachineInfoRequest(r fm_proto.GetMachineInfoRequest) *pb.GetMachineInfoRequest {
    return &pb.GetMachineInfoRequest{
        Hostname:               r.Hostname,
        IgnoreMissingLocalTags: r.IgnoreMissingLocalTags,
    }
}

func FromProtoGetMachineInfoRequest(r *pb.GetMachineInfoRequest) fm_proto.GetMachineInfoRequest {
    return fm_proto.GetMachineInfoRequest{
        Hostname:               r.Hostname,
        IgnoreMissingLocalTags: r.IgnoreMissingLocalTags,
    }
}
```

## Benefits

1. **Single source of truth** - Go structs in messages.go
2. **No manual converter maintenance** - Generated from type definitions
3. **Business logic written once** - In the service layer
4. **Type safety** - Converters are compile-time checked
5. **Easy to add new methods** - Add to messages.go, regenerate, implement in service
6. **Easy to add new transports** - Just add thin wrapper + use generated converters

---

## Current Implementation Status

### grpc-gateway Integration (Completed)

The grpc-gateway has been successfully integrated to provide REST API support. Here's what was implemented:

#### Files Modified/Added

1. **`proto/grpc/fleetmanager/fleetmanager.proto`** - Added HTTP annotations for all RPC methods
2. **`proto/grpc/fleetmanager/fleetmanager.pb.gw.go`** - Generated gateway code
3. **`proto/google/api/annotations.proto`** - Google API proto files for HTTP annotations
4. **`proto/google/api/http.proto`** - HTTP annotation definitions
5. **`cmd/fleet-manager/main.go`** - Added REST gateway server startup

#### How to Use

Start the fleet-manager with REST gateway enabled:

```bash
./fleet-manager \
  -certFile=/tmp/ssl/fleet-manager/cert.pem \
  -keyFile=/tmp/ssl/fleet-manager/key.pem \
  -topologyDir=/tmp/test-topology \
  -stateDir=/tmp/fresh-fleet-manager \
  -grpcPortNum=6878 \
  -restPortNum=8080 \
  -portNum=6977
```

Test REST endpoints:

```bash
# List hypervisor locations
curl http://localhost:8080/v1/locations

# Get machine info
curl http://localhost:8080/v1/machines/{hostname}

# List hypervisors in location
curl http://localhost:8080/v1/locations/SFO4/hypervisors
```

#### Architecture

```
REST Client (curl)
      │
      ▼
 HTTP :8080
      │
      ▼
grpc-gateway (translates HTTP→gRPC)
      │
      ▼
 gRPC :7878 (internal, localhost only, no TLS)
      │
      ▼
 grpcd handlers
      │
      ▼
 hypervisors.Manager
```

The gateway runs on a separate port and connects to an internal gRPC server
(without TLS) running on localhost only. The external gRPC server on port 6878
still uses TLS with client certificate authentication.

#### Regenerating Gateway Code

After modifying `fleetmanager.proto`, regenerate with:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
protoc \
  -I proto/grpc/fleetmanager \
  -I proto \
  --go_out=proto/grpc/fleetmanager --go_opt=paths=source_relative \
  --go-grpc_out=proto/grpc/fleetmanager --go-grpc_opt=paths=source_relative \
  --grpc-gateway_out=proto/grpc/fleetmanager --grpc-gateway_opt=paths=source_relative \
  fleetmanager.proto
```

