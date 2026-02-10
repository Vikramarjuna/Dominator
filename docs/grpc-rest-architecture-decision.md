# gRPC + grpc-gateway Architecture Decision

## Decision Summary

**Chosen Approach:** Standard gRPC + grpc-gateway  
**Alternative Considered:** Connect-Go  
**Date:** 2026-02-04  
**Status:** Implemented

---

## Context

Fleet-manager is being implemented with both gRPC and REST API support. We evaluated two approaches:

1. **Standard gRPC + grpc-gateway** (chosen)
2. **Connect-Go** (alternative)

---

## Decision: Use gRPC + grpc-gateway

### Primary Reasons

#### 1. **RESTful Paths for Human Users**

Fleet-manager will have:
- **Web UI** - Requires clean, RESTful HTTP endpoints
- **Manual API users** - Ops teams using curl/Postman for debugging and operations
- **Non-SDK integrations** - Scripts and tools that prefer REST over gRPC

**RESTful paths are important for these use cases:**
```
GET  /v1/machines/{hostname}
GET  /v1/locations/{location}/hypervisors
POST /v1/machines/{hostname}/power-on
```

vs. Connect-Go's default RPC-style paths:
```
POST /fleetmanager.FleetManager/GetMachineInfo
POST /fleetmanager.FleetManager/ListHypervisorsInLocation
POST /fleetmanager.FleetManager/PowerOnMachine
```

#### 2. **Auto-Generated RESTful Routing**

With grpc-gateway, we write **declarative HTTP annotations** in proto files:

```protobuf
rpc GetMachineInfo(GetMachineInfoRequest) returns (GetMachineInfoResponse) {
  option (google.api.http) = {
    get: "/v1/machines/{hostname}"
  };
}
```

**Code we write:** ~3 lines per method = ~24 lines total  
**Code auto-generated:** ~1000+ lines of routing, marshaling, validation

**Alternative with Connect-Go:** Would require ~225 lines of hand-written routing code to achieve RESTful paths.

#### 3. **Mature, Battle-Tested Solution**

- ✅ Used by Google, Kubernetes, and many large-scale systems
- ✅ Extensive documentation and community support
- ✅ Well-understood patterns and best practices
- ✅ Proven at scale

#### 4. **Clear Separation of Concerns**

- **gRPC** - For SDK users (Go, Python, Java clients)
- **REST/JSON** - For Web UI, curl, Postman, scripts

This aligns with how clients naturally want to consume the API.

---

## What We Gave Up (Connect-Go Benefits)

### 1. **Simpler Server Setup**

**Connect-Go:**
```go
// ~15 lines total
mux := http.NewServeMux()
path, handler := fleetmanagerconnect.NewFleetManagerHandler(svc)
mux.Handle(path, handler)
http.ListenAndServeTLS(":6979", certFile, keyFile, mux)
```

**gRPC + grpc-gateway:**
```go
// ~60 lines total
// External gRPC server (TLS, auth)
externalGrpc := grpc.NewServer(...)
pb.RegisterFleetManagerServer(externalGrpc, svc)

// Internal gRPC server (for gateway)
internalGrpc := grpc.NewServer()
pb.RegisterFleetManagerServer(internalGrpc, svc)

// Gateway
gwMux := runtime.NewServeMux()
pb.RegisterFleetManagerHandlerServer(ctx, gwMux, svc)

// Start servers
go externalGrpc.Serve(externalListener)
http.ListenAndServe(":6980", gwMux)
```

**Trade-off:** 4x more server setup code, but we get RESTful paths automatically.

### 2. **Native HTTP/1.1 Support**

Connect-Go natively supports HTTP/1.1 for gRPC-like calls, making it easier to use in environments without HTTP/2.

**Impact:** Low - Our infrastructure supports HTTP/2, and REST clients can use HTTP/1.1 via grpc-gateway.

### 3. **Better HTTP Integration**

Connect-Go is built on `net/http`, making it easier to:
- Use standard Go middleware
- Add custom HTTP headers
- Integrate with existing HTTP tooling

**Impact:** Medium - We can still do this with grpc-gateway, just requires more setup.

### 4. **Simpler Interceptor Model**

Connect-Go uses standard Go middleware patterns instead of gRPC interceptors.

**Impact:** Low - gRPC interceptors are well-understood and work fine for our auth needs.

### 5. **Modern, Future-Proof Approach**

Connect-Go is the modern direction for RPC in Go, actively developed by Buf (the protobuf company).

**Impact:** Medium - gRPC is still the industry standard and will be supported for years.

---

## Comparison Table

| Aspect | gRPC + grpc-gateway (Chosen) | Connect-Go (Alternative) |
|--------|------------------------------|--------------------------|
| **RESTful paths** | ✅ Auto-generated from annotations | ❌ Requires manual routing (~225 lines) |
| **Server setup code** | 🔴 ~60 lines | 🟢 ~15 lines |
| **Proto annotations** | 🔴 ~3 lines per method | 🟢 None needed |
| **Number of servers** | 🔴 2 (external + internal gRPC) | 🟢 1 |
| **Dependencies** | 🔴 3 (grpc, gateway, annotations) | 🟢 1 (connect) |
| **Maturity** | 🟢 Very mature, battle-tested | 🟡 Newer (2022+) |
| **HTTP/1.1 support** | 🟡 Via gateway only | 🟢 Native |
| **Middleware** | 🟡 gRPC interceptors | 🟢 Standard Go middleware |
| **Community/ecosystem** | 🟢 Large | 🟡 Growing |
| **For Web UI** | 🟢 Perfect (RESTful) | 🔴 Poor (non-RESTful) |
| **For SDK users** | 🟢 Standard gRPC | 🟢 gRPC-compatible |
| **For debugging** | 🟢 Easy curl commands | 🟡 Verbose curl commands |

---

## When Would We Reconsider Connect-Go?

We would switch to Connect-Go if:

1. **No Web UI or manual API users** - If all clients use SDKs, RESTful paths don't matter
2. **Simplicity becomes critical** - If maintaining 60 lines of server code becomes a burden
3. **HTTP/1.1 becomes a requirement** - If we need to support environments without HTTP/2
4. **Connect-Go adds RESTful routing** - If a code generator emerges that reads `google.api.http` annotations and generates Connect routing code

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│  Current Architecture: gRPC + grpc-gateway                   │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  SDK Clients (Go, Python)                                   │
│  gRPC Protocol                                              │
│     │                                                        │
│     └──→ External gRPC Server (TLS, auth, port 6979)       │
│              ↓                                               │
│         fleet-manager service handlers                      │
│                                                              │
│  Web UI / curl / Postman                                    │
│  REST/JSON                                                  │
│     │                                                        │
│     └──→ grpc-gateway (port 6980)                           │
│              ↓                                               │
│         Internal gRPC Server (localhost, no TLS)            │
│              ↓                                               │
│         fleet-manager service handlers                      │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## Conclusion

**We chose gRPC + grpc-gateway because:**
- ✅ RESTful paths are important for Web UI and manual API users
- ✅ Auto-generated routing saves us from writing ~225 lines of manual routing code
- ✅ Mature, proven solution used by industry leaders
- ✅ Clear separation: gRPC for SDKs, REST for humans

**We accepted the trade-offs:**
- ❌ More complex server setup (~60 lines vs ~15 lines)
- ❌ Running 2 gRPC servers instead of 1
- ❌ More dependencies

**The decision prioritizes developer experience for API consumers (Web UI, ops teams) over simplicity of server implementation.**

---

## References

- [grpc-gateway documentation](https://grpc-ecosystem.github.io/grpc-gateway/)
- [Connect-Go documentation](https://connectrpc.com/docs/go/getting-started)
- [Google API HTTP annotations](https://github.com/googleapis/googleapis/blob/master/google/api/http.proto)
- [Architecture refactor document](./grpc-architecture-refactor.md)

