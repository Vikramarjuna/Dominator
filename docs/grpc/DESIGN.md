# Design Document: Adding gRPC and REST APIs to Dominator Ecosystem

> Source: cleaned-up version of the Google Doc
> [Design Document: Adding gRPC and REST APIs to Dominator](https://docs.google.com/document/d/1LukJVDHB1KrGtMocEzsl2d8T89ZACpStFvK4em0rUIw).
> Retracted (strike-through) sections from the original — the @http annotation
> syntax, the GrpcToSrpcMethodMapping facility, and the response-enrichment view
> enums — have been omitted; the corresponding architectural decision is "use
> the same method name across all protocols" per the resolved review thread.

## Executive Summary

Dominator exposes only SRPC (a custom RPC protocol) for internal services. This
document proposes adding gRPC and REST APIs and explains the challenges
involved and how we propose to solve them.

Adding gRPC and REST support will enable:

- SDK users to integrate with Dominator using industry-standard protocols.
- Web UIs and ops teams to use REST/HTTP APIs.
- Cross-language client support (Python, Java, JavaScript, etc.).

The following points are kept in mind while adding the support:

- Maintain the RBAC mechanism used by SRPC while implementing gRPC and REST.
- Maintain backward compatibility with existing SRPC clients while avoiding
  duplicate type definitions.
- Adhere to gRPC and REST API development guidelines.

### Key Architectural Decisions

| Feature             | Decision                                  | Rationale                                                                                                     |
| ------------------- | ----------------------------------------- | ------------------------------------------------------------------------------------------------------------- |
| Authentication      | Reuse SRPC mTLS                           | No new infrastructure needed; leverages existing CA trust and OID extensions for identity.                    |
| Authorization       | Interceptor-based RBAC                    | Bridges the gRPC context to SRPC `AuthInformation` so business logic does not need to change.                 |
| Concurrency         | Explicit rate limiting                    | Counteracts gRPC's HTTP/2 multiplexing, which removes SRPC's implicit "one-at-a-time" safety.                 |
| Source of Truth     | Go structs (the existing SRPC types)      | Prevents breaking existing `encoding/gob` wire compatibility for internal SRPC clients.                       |
| Schema Management   | Auto-generated `.proto`                   | Eliminates manual type drift and lets the Go-native types (IPs/MACs) remain the primary definition.           |
| REST Strategy       | grpc-gateway                              | Provides a standard RESTful interface as a side-effect of gRPC with zero manual routing code.                 |

## Background

Dominator's internal communication is built on SRPC, using `encoding/gob` for
Go-to-Go traffic and a JSON endpoint for basic interoperability. `gob` encoding
is incredibly efficient for our internal Go services — handling native Go
types, pointers, and recursive structures with zero boilerplate and far less
overhead than Protobuf or JSON. It is secured via TLS with X.509 client certs.

Even with the existing JSON support, external integration is a manual,
error-prone process due to a few key bottlenecks:

- **Lack of a type contract**: There is no machine-readable schema. External
  developers have to manually audit Go source code in `messages.go` to
  reverse-engineer request/response structures.
- **Manual mapping of types**: Complex Go types (like `net.IP`) don't have a
  standard JSON representation. Clients have to map these types manually,
  leading to silent failures and "protocol drift" whenever the Go structs are
  updated.
- **No web browser support**: Because SRPC uses a custom framing protocol
  rather than standard HTTP, we cannot support modern Web UIs or
  browser-based management tools.

We are adding gRPC and REST to move toward an industry-standard API model.
This allows us to keep the internal efficiency of SRPC while providing a
strictly typed, auto-generated contract for everything else — enabling
type-safe SDKs, native browser support, and a "free" REST interface via
grpc-gateway.

## Alternatives Considered

We evaluated several paths for expanding Dominator's reach beyond Go. The goal
was to find a balance between developer experience and long-term maintenance
overhead.

### Rust-based SDKs with multi-language bindings

We considered expanding the existing Rust client (`rust/lib/srpc/client/`) and
using it as a shared core for other languages (e.g., via PyO3 for Python).

While this reuses our existing code, it doesn't solve the "type discovery"
problem. There is still no machine-readable schema, so every field mapping
remains a manual guess. Furthermore, we would become responsible for
maintaining the FFI/WASM glue code for every language, which is a significant
and brittle engineering tax.

### Native REST (no gRPC)

We considered adding native `net/http` handlers and generating an OpenAPI spec
from the Go source for type discovery. While this technically solves the
interoperability issue, it is a poor fit for Dominator's specific
requirements. JSON encoding is significantly heavier than Protobuf — a major
drawback since our primary use case remains high-performance communication
between internal services.

More importantly, many of our core APIs rely on **long-lived streaming**,
which is native to gRPC but awkward and inconsistent to implement in a pure
REST/HTTP 1.1 model. By using gRPC as the base and generating the REST layer
via grpc-gateway, we get the performance of a binary protocol for our streams
while getting the OpenAPI spec and REST boilerplate for free. It gives us
both protocols for less manual effort than writing a native REST-only
implementation.

### gRPC-only (no REST), or deferring REST

We could simplify the initial rollout by deferring (or omitting) the REST
gateway entirely. This would exclude web browsers and `curl`-style debugging
in the short term. Since grpc-gateway generates the REST layer automatically
from the `.proto` file, the marginal cost of supporting both is small once
gRPC is in place, so deferring REST is a reasonable interim option but not a
permanent one.

## Key Challenges

This section captures the key challenges in implementing gRPC and REST
support.

### RBAC

**Current RBAC model.** SRPC uses TLS client certificates for authentication
and authorization.

- **Certificate contains**: username (Common Name), groups (custom OID), and
  permitted methods (custom OID).
- **Authentication flow**: client connects with TLS cert → server verifies CA
  signature → server extracts metadata → server checks the requested method
  against the permitted list → handler receives a `*srpc.Conn` with auth
  information.

We want to reuse the existing certificate-based authentication without
duplicating the code or introducing a new authentication and authorization
mechanism.

### Connection Model and Concurrency

**SRPC connection model:**

- **Sequential**: handles one RPC at a time via a `callLock` mutex. No
  request multiplexing.
- **Connection limits**: global pool limit (~900 connections per process by
  default; configurable — Dominator, for example, raises this to ~64k).
- **Implicit protections**: natural rate limiting because of serialization.
- **Explicit per-user/per-method limits**: a handful of methods in
  fleet-manager and dominator are hard-coded to 1 concurrent call per user
  via `lib/srpc/serverutil`; everything else relies on the implicit
  serialization above.

**gRPC connection model.** gRPC uses HTTP/2 with stream multiplexing, where
each connection handles unlimited concurrent RPCs:

- Multiple concurrent requests on the same TCP connection.
- Each RPC gets its own HTTP/2 stream with a unique stream ID.
- No serialization — full concurrency.

**Connection limits:**

- **Client-side**: typically 1–4 connections per target (gRPC manages this
  internally).
- **Server-side**: no application-level limit (only OS resources).

**Removed protections:**

- ❌ No explicit connection-pool limit (gRPC manages connections internally).
- ❌ No serialization (unlimited concurrent RPCs per connection).
- ❌ No natural rate limiting.

**The challenge.** SRPC's serialization and connection pool provide implicit
rate limiting. gRPC removes these protections.

| Protection           | SRPC                              | gRPC                       |
| -------------------- | --------------------------------- | -------------------------- |
| Max concurrent RPCs  | ~900 (pool limit, configurable)   | Unlimited                  |
| Serialization        | Yes (1 RPC at a time)             | No (unlimited concurrent)  |
| Natural rate limit   | Yes (implicit)                    | No (must be explicit)      |

### Source of Truth for Method and Type Definitions

- **Option 1 (Reuse SRPC types)**: not possible (gRPC needs `.proto`).
- **Option 2 (Use proto types for SRPC)**: breaks backward compatibility (wire
  format incompatibility: `gob` vs. protobuf).
- **Option 3 (Manual maintenance)**: high burden and risk of drift.
- **Option 4 (Generate proto from SRPC types)**: **proposed approach.**

**Challenges with Option 4:**

- **Error fields**: SRPC has `Error string` fields; gRPC uses status codes.
- **Type mismatches**: SRPC uses `net.IP`; proto uses `bytes`.
- **Generic errors**: current logic returns `errors.New(...)`; gRPC needs
  structured codes.

## Proposed Solution

### Overall Architecture: Dual Servers with Shared Business Logic

We will have SRPC and gRPC servers coexist on the same port, sharing the same
business logic. Only thin client code is added for gRPC alongside SRPC.

- **Two servers run simultaneously**: SRPC and gRPC on the same port
  (different protocols).
- **Shared business logic**: written once, used by both servers.
- **Thin presentation layer**: SRPC and gRPC handlers are thin wrappers.
- **Use converters**: auto-generated converters between SRPC structs and
  `pb/` types.
- **REST**: grpc-gateway auto-generates REST from gRPC.

### Protocol Routing on a Shared Port

Each Dominator service already listens on a fixed port (e.g., hypervisor on
6976, fleet-manager on 6967, dominator on 6970). To avoid port proliferation
and to keep deployments where multiple services run on the same machine
simple, gRPC and REST will be served on the same port as SRPC. Routing is
done at the HTTP layer using Go's `http.ServeMux`.

The listener distinguishes the three protocols using standard HTTP semantics:

- **SRPC**: client issues `HTTP CONNECT /_goSRPC/` (or `/_SRPC/`); the
  connection is then upgraded and all subsequent traffic is SRPC framing over
  the upgraded TCP connection. Existing clients work unchanged.
- **gRPC**: identified by `Content-Type: application/grpc` to
  `/<package>.<Service>/<Method>`. The shared listener must therefore serve
  HTTP/2 (h2) over the same TLS endpoint.
- **REST**: any request matching the grpc-gateway mux paths (e.g.,
  `/v1/...`) is dispatched to the gateway, which translates it into an
  in-process gRPC call.

A single TLS configuration (the existing SRPC one) terminates the connection
for all three protocols, so the mTLS-based identity extraction described in
the security section applies uniformly regardless of which protocol the
client used.

### Reuse SRPC's TLS Certificates and Auth

We will continue to use the same certificates, the same CA trust, and the
same RBAC across all three protocols.

- Reuse SRPC's TLS certificates for gRPC and REST.
- Extract auth information using interceptors (gRPC) and middleware (REST).
- Reuse the existing RBAC logic in the business layer.
- The same permitted-methods list works for all protocols.

Certificates continue to be issued by Keymaster (not Vault).

**gRPC security implementation.**

1. **TLS configuration**:
   - Reuse SRPC's TLS config: `srpc.GetServerTlsConfig()`.
   - Wrap in gRPC credentials: `credentials.NewTLS(tlsConfig)`.
   - Pass to the gRPC server: `grpc.NewServer(grpc.Creds(creds), ...)`.
2. **Auth extraction via interceptors**:
   - Extract TLS peer info from context: `peer.FromContext(ctx)`.
   - Reuse SRPC's auth extraction: `srpc.GetAuthFromTLS(tlsInfo.State)`.
   - Extract permitted methods:
     `srpc.GetPermittedMethodsFromTLS(tlsInfo.State)`.
   - Store in context for handlers: `context.WithValue(ctx, connKey, conn)`.
3. **Handlers use auth**:
   - Extract connection from context: `lib_grpc.GetConn(ctx)`.
   - Get auth information: `conn.GetAuthInformation()`.
   - Pass to business logic (same as SRPC):
     `manager.ChangeMachineTags(hostname, authInfo, tags)`.

**REST security implementation.** The REST gateway uses mTLS:

- Reuse SRPC's TLS config (cloned).
- Enable mTLS: `ClientAuth = tls.RequireAndVerifyClientCert`.
- REST clients must present a valid certificate.

**Security flow.** REST client → TLS handshake (mTLS) → grpc-gateway →
internal gRPC server → business logic.

### Rate Limiting

Since gRPC removes SRPC's implicit rate limiting (connection pool +
serialization), we will enhance the current `lib/srpc/serverutil` to support
additional levels of rate limiting and use those in gRPC interceptors. The
existing per-user-method concurrency limits (described in the connection
model section above) will be preserved as-is for SRPC; the new per-user,
global, and per-method limiters introduced here apply uniformly across all
three protocols and act as guardrails on the gRPC/REST paths where the
implicit serialization no longer exists.

**Per-user rate limiting:**

- Limits each user **across all connections** (not per connection).
- Critical because gRPC connections multiplex unlimited concurrent requests
  via HTTP/2.
- Example: user "alice" gets 1,000 req/sec total, whether using 1 connection
  or 10.

**Global rate limiting:**

- Limits total server load across all users.
- Protects against aggregate traffic spikes.

**Per-method rate limiting:**

- Different limits for different methods (per user).
- Protects expensive operations (e.g., `CreateVm`, `DestroyVm`) with lower
  limits.
- Example: user "alice" can call `CreateVm` max 10 times/sec, but `ListVMs`
  100 times/sec.

**Example configuration.** Rate limits should be configurable without code
changes:

```yaml
rate_limits:
  global:
    requests_per_second: 10000
    burst: 20000
  per_user:
    requests_per_second: 1000
    burst: 2000
  per_method:
    CreateVm:  {requests_per_second:  10, burst:  20}
    DestroyVm: {requests_per_second:  10, burst:  20}
    ListVMs:   {requests_per_second: 100, burst: 200}
```

The implementation will use a token-bucket algorithm in-process; no external
dependency (Redis, etc.) is taken on, since much of the ecosystem is intended
to be the lowest software layer.

### Metrics and Monitoring

Dominator already exposes per-method RPC metrics through the tricorder
library — permitted call count, denied call count, successful call count, and
call duration distributions — currently emitted under the SRPC namespace. As
part of this work we will:

- Tag the existing metrics with a protocol dimension
  (`protocol={srpc|grpc|rest}`) so that overall QPS, per-protocol QPS, error
  rates, and latency distributions can all be queried from the same metric.
  This avoids duplicating dashboards per protocol and keeps a single source
  of truth for service health.
- Add a dedicated rate-limit metric:
  `rate_limit_denied_total{method, limit_type, protocol}` — the count of
  requests rejected by a rate limiter (i.e., not admitted for execution).
  `limit_type` is one of `global`, `per_user`, or `per_method`, matching the
  three levels described above.
- Log individual denials at debug level with the user identity, method, and
  limit type, to support investigation of specific clients hitting limits
  without inflating metric cardinality.

### Proto Generation from SRPC Messages

We will generate `.proto` files from the existing SRPC structs (SRPC types as
the source of truth). For an API/method that we want to expose via gRPC or
REST, add the `@grpc` tag to the types and methods. APIs without the tag
remain SRPC-only.

**Generation pipeline:**

```
SRPC proto structs (source of truth)
        ↓
   cmd/proto-gen
   ├─→ Parse types with @grpc tags
   ├─→ Parse methods with @grpc tags
   ├─→ Generate .proto files
   ├─→ Generate converters (SRPC ↔ pb/)
   └─→ Invoke protoc
        ↓
Generated: .proto, pb/, converters_gen.go
```

**Typical flow:**

1. Add `@grpc` tags to the SRPC method.
2. Add `@grpc` tag to the request/response types.
3. Run `make proto-gen`.
4. Implement the gRPC handler.
5. Commit source + generated files.

The same method name is used across SRPC and gRPC. Keeping names consistent
is more important than following gRPC naming conventions verbatim — different
names would be a continuous drag on developers. The default REST path is
derived from the service and method name (no hand-written `@http`
annotations), with the HTTP verb chosen from common conventions in the method
name.

### Converters: Bridging SRPC and gRPC Types

**Challenge.** Business logic uses SRPC structs; gRPC uses `pb/` types.

**Solution.** A custom proto-gen tool auto-generates bidirectional converters
(SRPC ↔ `pb/`) along with the proto files.

**Conversion strategy:**

| Go type (SRPC)             | Proto type (`pb/`)  | Conversion strategy                       |
| -------------------------- | ------------------- | ----------------------------------------- |
| `string`, `int`, `bool`    | Same                | Direct copy (auto-generated)              |
| `net.IP`, `net.HardwareAddr` | `bytes`           | Zero-cost cast: `[]byte(ip)`              |
| Embedded structs           | Flattened fields    | Auto-flattening logic (auto-generated)    |
| `map[string][]string`      | Nested message      | Hand-written extension methods            |

### Error Handling: Typed Errors for gRPC Status Codes

gRPC and REST need proper status codes (`NotFound` = 404, `InvalidArgument`
= 400, etc.), but business logic returns generic Go errors.

We will introduce typed errors in the `lib/errors` package and a new
interface that exposes a gRPC status code. We will provide a helper to
inspect an error: if it implements the interface, we extract the code from
it; otherwise, we fall back to parsing the error string to identify the
error type. The fallback is best-effort and may not be accurate. The intent
is that this is a stop-gap until the APIs are updated to return known error
types instead of generic errors. This approach addresses most cases and is
incremental.

The error field — a field named `Error` of type `string` — is ignored when
generating the proto.

### Committing Generated Files

Commit both the source (SRPC structs) and the generated files (`.proto`,
`pb/`, converters).

**Rationale:**

- Contributors don't need the proto-gen tool installed.
- CI can verify the generated files are up to date.
- Clear diffs when types change.
- Standard practice (similar to generated mocks and protobufs).

### Protocol-Specific APIs (Edge Cases)

The general principle is that all APIs should be available in both SRPC and
gRPC to maintain feature parity. SRPC-only APIs will exist transitionally
while we add support for existing methods incrementally — simply by not
adding the `@grpc` tag for a method. gRPC-only APIs are not recommended; if
truly required, they would be added via a hand-written proto file in a new
service (not on top of an existing service).

## Implementation Approach

We automate proto generation from existing SRPC APIs to avoid manual
`.proto` writing and to keep types in sync. While most types are
auto-generated, in a few cases it may be best to hand-write the converters.

**Approach:**

- **Method-first workflow**: tag SRPC methods with `@grpc` annotations; the
  tool discovers all referenced types automatically.
- **AST-based parsing**: reads Go source files to extract types and parse
  comment annotations.
- **Auto-generation**: produces `.proto` files, type converters
  (SRPC ↔ proto), and invokes `protoc`.

**Field numbering.** Supports explicit `proto:"N"` tags for stability. While
not needed in most cases, when introducing a new field in a struct,
developers will have to add the tag to existing fields to avoid breaking
backward compatibility. CI will catch unintended modifications.

**Generated artifacts:**

- `proto/*/grpc/*.proto` — protocol buffer definitions.
- `proto/*/grpc/*.pb.go` — generated by `protoc` (messages + service stubs).
- `proto/*/converters_gen.go` — bidirectional type converters.

**Migration strategy — per-API incremental migration:**

1. Tag the SRPC method with `@grpc` annotations.
2. Run `make proto-gen` to generate proto files and converters.
3. Implement the gRPC handler, using generated converters to call existing
   SRPC business logic.
4. Test that both SRPC and gRPC endpoints work.
5. Commit generated files alongside the handler implementation.

To avoid an unreviewable monster PR, the rollout is broken into a series of
small, well-contained PRs which can be easily reviewed. The sequencing
starts with adding/extending packages — rate limiters and other utility
code — before any service binary is touched. The detailed PR plan lives in
[`SPEC-KIT.md`](./SPEC-KIT.md).

## FAQ

**Q: How do we handle breaking changes in the `.proto` file?**

A: Since the Go structs in `messages.go` are the source of truth (SOT), any
breaking change there (like renaming a field) will trigger a change in the
generated `.proto` and converters. Because we commit the generated code, the
PR will clearly show the blast radius of the change across all three
protocols. External SDK users should be advised that the gRPC/REST API
follows the same stability guarantees as the underlying SRPC types.

**Q: What happens if the custom proto-gen fails or produces invalid code?**

A: The proto-gen tool will be a hermetic Go binary within the repo
(`cmd/proto-gen`). It uses the standard `go/ast` parser. If it fails, the
`make proto-gen` step in the Makefile will return a non-zero exit code,
blocking the build. Since the generated files are committed, a broken tool
won't break the build for other developers — only for the person trying to
modify the API.

**Q: Does the gRPC server increase memory overhead significantly?**

A: The overhead is minimal because the gRPC and SRPC servers share the same
underlying `Manager` instance. The primary increase in memory will come from
HTTP/2 header compression (HPACK) tables and internal gRPC buffers, which
are negligible.

**Q: Why not just use `protoc-gen-go` for everything?**

A: Standard `protoc` generation creates Go structs with internal protobuf
fields and specific getter/setter methods that are wire-incompatible with
Dominator's existing `encoding/gob` serialization. Our approach allows us to
keep the clean "Plain Old Go Objects" (POJO) that the rest of the ecosystem
relies on.
