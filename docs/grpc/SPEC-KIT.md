# Dominator gRPC Spec Kit

**Status:** Draft for review
**Owner:** @Vikramarjuna
**Last updated:** 2026-05-04

This spec kit replaces the bundled "infra" approach with a libraries-first,
small-PR roadmap. It catalogues what already exists across the `grpc-*`
branches, picks the canonical pieces, and sequences the work so that every PR
is reviewable in isolation.

---

## 1. Goals & Non-Goals

### Goals
- Land gRPC support for Dominator without disrupting SRPC clients.
- Reuse SRPC's authorization, RBAC, TLS, and method-power semantics — do not
  re-implement them.
- Keep every PR ≤ ~300 net lines of production code (excluding generated files
  and vendored deps) so reviewers can finish in one sitting.
- Establish a code-generation pipeline so proto definitions and converters stay
  in lockstep with `proto/<svc>/messages.go`.
- Pilot the architecture on Hypervisor with 5 read/lifecycle APIs before
  expanding.

### Non-Goals (deferred, not rejected)
- REST/grpc-gateway exposure (Phase E).
- Bidirectional streaming variants of `CreateVm` (Phase E).
- FleetManager / ImageServer / Imaginator gRPC surfaces (Phase F).
- Migrating SRPC clients to gRPC. SRPC stays the primary protocol.
- Wire-level compatibility between SRPC gob and gRPC protobuf (different
  protocols, intentional).

---

## 2. Guiding Principles

1. **Libraries first.** No service can take a gRPC dependency until the
   underlying library is merged. Phases A–C must precede Phase D.
2. **One concern per PR.** Auth, errors, streaming, metrics, REST, and codegen
   each ship in their own PR. The `grpc-infra` superset branch is rejected as
   a merge target.
3. **Reuse over duplicate.** gRPC handlers are thin wrappers that call existing
   `Manager`/service methods. No business logic in `grpcd/`.
4. **Generate, don't hand-write.** Proto files and SRPC↔proto converters are
   generated from `proto/<svc>/messages.go`. CI fails if regeneration produces
   a diff.
5. **Backward compatible at every step.** Every PR must keep `make` green,
   every existing SRPC test passing, and all CLIs working.
6. **Net-additive infrastructure.** Library PRs only add files under
   `lib/grpc/`, `lib/errors/`, `lib/server/`, `cmd/proto-gen/`. They do not
   touch service binaries.

---

## 3. Inventory of Existing Work

### Open PRs (rebase, don't reopen)

| PR | Branch | Scope | Status |
|----|--------|-------|--------|
| #226 | `grpc-auth-core` | `lib/grpc/api.go`, `lib/srpc` exports | Open, no review. **Now depends on Phase 0.** |
| #227 | `grpc-errors` | `lib/errors/types.go`, `lib/grpc/errors.go` | Open, no review |

### Greenfield (no existing branch)

| Need | Target PR | Notes |
|------|-----------|-------|
| In-repo design doc + spec kit | PR-0.0 | Ports the Google Doc design and this spec kit into `docs/grpc/` per Richard's review comment. Must land first so subsequent PRs can reference it. |
| Global + per-method limiters in `lib/srpc/serverutil` | PR-0.1 | Extends existing `PerUserMethodLimiter`. Per design-doc §"Rate Limiting" and Richard's review comment. |
| gRPC rate-limit interceptors | PR-0.2 | Wraps any `srpc.MethodBlocker`. See §5 Phase 0. |
| `lib/errors` typed-error catalogue | PR-A2a | Splits PR #227. `CodedError` interface + standard errors, no gRPC import. |
| Protocol-dimension on existing tricorder call metrics | PR-B0 | Touches `lib/srpc/server.go` so SRPC and gRPC dashboards can union. |
| Hand-written common converters (`Tags`, `net.IP`, `net.HardwareAddr`) | PR-B3 | Runtime helpers referenced by generated converters in Phase C. |

### Branches with reusable code (not yet PR'd)

| Branch | Reusable artifact | Target PR |
|--------|-------------------|-----------|
| `grpc-streaming` | `lib/grpc/streaming.go` (`StreamingConn`) | PR-B1 |
| `grpc-metrics-rest` | `lib/grpc/metrics.go` | PR-B2 |
| `grpc-metrics-rest` | `lib/grpc/rest.go` | PR-E2 (deferred to Phase E) |
| `grpc-metrics-rest` | `lib/server/api.go` (combined listener) | PR-B4 |
| `grpc-proto-gen-poc` | `cmd/proto-gen/{scanner,parser}.go` | PR-C1 |
| `grpc-proto-gen-poc` | `cmd/proto-gen/generator.go` | PR-C2 |
| `grpc-proto-gen-poc` | `cmd/proto-gen/converters.go` | PR-C3 |
| `grpc-proto-gen-poc` | `Makefile.proto`, `scripts/check-proto-gen.sh` | PR-C4 |
| `grpc-minimal` | `hypervisor/grpcd/*.go` handler patterns | Reference for Phase D |
| `grpc-copy-0` | Design docs in `docs/` | Already analysed; archive |

### Branches to retire after spec is approved
- `grpc-infra` — bundled scope, superseded by Phases A+B.
- `grpc-copy-0` — snapshot, content lifted into this doc.

---

## 4. Target Architecture

See `design-docs/Dominator/ArchitecturalOverview.md` for the broader system
context. The gRPC layering target is:

```
        SRPC client                gRPC client                (REST client — Phase E)
            │                          │                              │
            ▼                          ▼                              ▼
   ┌─────────────────┐        ┌─────────────────┐          ┌──────────────────┐
   │  srpc.Receiver  │        │ grpc.Server +   │          │ grpc-gateway mux │
   │ (existing)      │        │  interceptors   │          │ (deferred)       │
   └────────┬────────┘        └────────┬────────┘          └────────┬─────────┘
            │                          │                              │
            │       ┌──────────────────▼──────────────────┐           │
            │       │  lib/srpc/serverutil (Phase 0)      │           │
            │       │  • PerUserMethodLimiter (existing)  │           │
            │       │  • GlobalLimiter        (PR-0.1)    │           │
            │       │  • PerMethodLimiter     (PR-0.1)    │           │
            │       └──────────────────┬──────────────────┘           │
            │                          │ shared MethodBlocker         │
            │       ┌──────────────────▼──────────────────┐           │
            │       │      lib/grpc (this spec, Phase 0–B)│           │
            │       │  • RateLimitInterceptor (PR-0.2)    │           │
            │       │  • UnaryAuthInterceptor             │           │
            │       │  • StreamAuthInterceptor            │           │
            │       │  • ErrorToStatus / FromStatus       │           │
            │       │  • StreamingConn (srpc adapter)     │           │
            │       │  • metrics (tricorder, protocol=grpc)│          │
            │       │  • common_converters (Tags/IP/MAC)  │           │
            │       └──────────────────┬──────────────────┘           │
            │                          │                              │
            │            ┌─────────────▼──────────────┐               │
            └────────────►   <svc>/rpcd  (existing)   ◄───────────────┘
                         │   thin SRPC handlers       │
                         └─────────────┬──────────────┘
                                       │
                                       ▼
                              <svc>/manager (or service/)
                              all business logic lives here
```

The new `<svc>/grpcd/` package contains only:
- `api.go` — `RegisterServer`, calls `grpc.RegisterServiceOptions(...)`.
- One file per RPC, each ≤ 30 lines: convert request → call manager → convert
  response → return `grpc.ErrorToStatus(err)`.

Converters live in `proto/<svc>/converters_gen.go` (generated). Hand-written
converters for irreducible types (e.g. `net.IP`, `net.HardwareAddr`) live in
`proto/<svc>/converters.go` and are referenced by the generator.

### Code-generation pipeline (Phase C)

```
proto/<svc>/messages.go ─┐
                         ├─► cmd/proto-gen ─► proto/<svc>/grpc/<svc>.proto
<svc>/rpcd/*.go (tags)  ─┘                  ├─► proto/<svc>/converters_gen.go
                                            └─► proto/<svc>/grpc/*.pb.go (via protoc)
```

`@grpc` and `@http` annotations on SRPC method comments drive what gets
exposed. CI runs `scripts/check-proto-gen.sh` and fails the build on diff.



---

## 5. Phased PR Plan

Phases are gates: no PR in Phase N may merge until all PRs in Phase N-1 are
merged. Within a phase, PRs may proceed in parallel unless noted.

### Phase 0 — Foundations (must merge first)

The first PR in this plan ports the design document into the repo. Richard's
review comment on PR #226 explicitly asks for the design to live next to the
code so reviewers can reference it without leaving GitHub. Once that is in,
Phase 0 then lands the rate-limiter foundation: a precondition for exposing
gRPC at all, because SRPC's implicit serialisation (one RPC at a time per
connection) and connection-pool cap provide a natural ceiling that gRPC's
HTTP/2 multiplexing removes. Today `lib/srpc/serverutil` ships exactly one
limiter (`PerUserMethodLimiter`) and a handful of methods in `fleet-manager`
and `dominator` are hard-coded to one concurrent call per user. The design doc
("Rate Limiting" section) and Richard's review comment direct us to
**expand `lib/srpc/serverutil`** with the additional limiters rather than
introducing a parallel limiter package under `lib/grpc`. The gRPC interceptor
is then a thin adapter over the existing `srpc.MethodBlocker` interface,
guaranteeing one policy implementation across both protocols.

| PR | Title | Files | Net LoC | Depends on |
|----|-------|-------|---------|------------|
| 0.0 | `docs/grpc: design doc + spec kit` | `docs/grpc/DESIGN.md`, `docs/grpc/SPEC-KIT.md` | docs only | none |
| 0.1 | `serverutil: global + per-method token-bucket limiters` | `lib/srpc/serverutil/{api,globalLimiter,perMethodLimiter}.go` + tests | ~300 | 0.0 |
| 0.2 | `grpc: rate-limit interceptors over MethodBlocker` | `lib/grpc/ratelimit.go`, `lib/grpc/ratelimit_test.go` | ~200 | 0.1 |

**Scope of 0.0**:

- Port the Google Doc design (`Adding gRPC and REST APIs to Dominator`) into
  `docs/grpc/DESIGN.md` as Markdown. Preserve section structure; add a header
  noting the original Google Doc URL as the source of truth for inline
  comments and revision history.
- Add `docs/grpc/SPEC-KIT.md` (this document) as the executable roadmap.
- No code changes. Reviewable in one sitting; primarily content review.
- Subsequent PR descriptions reference the in-repo path so the design intent
  travels with the diff.

**Scope of 0.1**:

- New `GlobalLimiter` (token-bucket; `requests_per_second`, `burst`) — applies
  to all callers regardless of identity.
- New `PerMethodLimiter` (token-bucket per `(user, method)`) — refines the
  existing concurrency-only `PerUserMethodLimiter` with rate semantics.
- New `MultiLimiter` that composes any number of `MethodBlocker`s in order,
  short-circuiting on first denial. Lets services stack `global → per_user →
  per_method` without writing glue.
- New `rate_limit_denied_total{method, limit_type, protocol}` tricorder counter
  registered by the limiters themselves; `limit_type ∈ {global, per_user,
  per_method}`.
- Existing `PerUserMethodLimiter` API remains unchanged; new limiters all
  satisfy `srpc.MethodBlocker` so they're drop-in for `RegisterNameWithOptions`.
- No `lib/grpc` import; no service binary changed.

**Scope of 0.2**:

- `UnaryRateLimitInterceptor(srpc.MethodBlocker)` and
  `StreamRateLimitInterceptor(srpc.MethodBlocker)`. Accepts any
  implementation: the existing `PerUserMethodLimiter`, the new limiters from
  0.1, or a `MultiLimiter` stack.
- Interceptor ordering contract (documented in code): rate-limit runs **after**
  auth (so the principal is known) but **before** the handler. Denial maps to
  `codes.ResourceExhausted` via `lib/errors` (the typed error lands in A2a).
- The `BlockMethod` cleanup function is invoked in `defer`, including on panic
  recovery.
- No new limiter implementations live here — that responsibility stays in
  `lib/srpc/serverutil`.

**Exit criteria**: both PRs merged. `go test ./lib/srpc/serverutil/... ./lib/grpc/...`
green. The exported interceptor chain helper that A1 will introduce can accept
a `MethodBlocker` slot (zero-value = no-op) without further refactor. SRPC
receivers in `cmd/mdbd` keep working unchanged.

### Phase A — Auth & errors foundation (in flight)

PR #227 is being split: the typed-error catalogue is foundational (touches
`lib/errors` only, no gRPC import) and several other PRs in this plan depend
on it, so it lands as A2a. The `lib/grpc` translator follows as A2b.

| PR | Title | Files | Net LoC | Depends on |
|----|-------|-------|---------|------------|
| A1 | `grpc: auth core (interceptors, Conn)` (#226, rebase) | `lib/grpc/api.go`, `lib/srpc/{exports,auth}.go` | ~340 | **0.2** |
| A2a | `errors: CodedError interface + standard typed errors` | `lib/errors/{api,types}.go` + tests | ~180 | none |
| A2b | `grpc: ErrorToStatus / FromStatus` (split from #227) | `lib/grpc/errors.go` + tests | ~150 | A2a |

A1 must be rebased after 0.2 to include the rate-limit interceptor in its
chain helper. A2a is fully self-contained (no `lib/grpc` import) and can be
opened as soon as Phase 0 lands. A2b imports A2a and the typed
`RateLimitedError` defined there.

**Exit criteria:** all three PRs merged to master. `go build ./...` and
`go test ./lib/...` green. No service binary changed.

### Phase B — Library completion

| PR | Title | Files | Net LoC | Depends on |
|----|-------|-------|---------|------------|
| B0 | `srpc: add protocol label to tricorder call metrics` | `lib/srpc/server.go` (metric registration) + tests | ~150 | none |
| B1 | `grpc: streaming connection adapter` | `lib/grpc/streaming.go`, `lib/grpc/streaming_test.go` | ~250 | A1 |
| B2 | `grpc: tricorder metrics interceptors` | `lib/grpc/metrics.go`, `lib/grpc/metrics_test.go` | ~200 | A1, B0 |
| B3 | `grpc: hand-written common converters (Tags, IP, MAC)` | `lib/grpc/common_converters.go` + tests | ~180 | A1 |
| B4 | `server: combined HTTP + gRPC listener` | `lib/server/api.go`, `lib/server/listener.go` | ~250 | A1 |

B0 is the prerequisite refactor for the design doc's "Tag existing metrics
with a protocol dimension" requirement: existing SRPC counters
(`numPermittedCalls`, `numDeniedCalls`, `failedCallsDistribution`,
`successfulCallsDistribution`) gain a `protocol` label set to `"srpc"`. B2
then registers the same counters under `"grpc"` so dashboards union cleanly.

B3 ships the runtime helpers that auto-generated converters in Phase C will
call into for `net.IP ↔ bytes`, `net.HardwareAddr ↔ bytes`, `tags.Tags`, and
`tags.MatchTags`. Hand-written, not generated, because these involve Go-stdlib
types proto-gen cannot reason about.

REST middleware (originally drafted as B3 on `grpc-metrics-rest`) is renamed
PR-E2 and deferred to Phase E so it ships together with grpc-gateway.

**Exit criteria:** `lib/grpc` is feature-complete for unary, server-streaming,
client-streaming, and bidi RPCs. A combined listener serves SRPC, HTTP, and
gRPC on one port (mux'd). Tricorder metrics carry a uniform `protocol` label
across both stacks. Each PR has its own unit tests.

### Phase C — Code generation tooling

| PR | Title | Files | Net LoC | Depends on |
|----|-------|-------|---------|------------|
| C1 | `proto-gen: AST scanner + annotation parser` | `cmd/proto-gen/{main,scanner,parser}.go`, fixtures | ~400 | none |
| C2 | `proto-gen: .proto file generator` | `cmd/proto-gen/generator.go`, golden tests | ~350 | C1 |
| C3 | `proto-gen: Go converter generator` | `cmd/proto-gen/converters.go`, golden tests | ~400 | C2 |
| C4 | `build: Makefile.proto + check-proto-gen.sh CI hook` | `Makefile.proto`, `scripts/check-proto-gen.sh`, CI workflow | ~150 | C3 |

C1–C3 use golden-file testing (`testdata/`) to lock generator output. C4
extends `.github/workflows/` (or equivalent) to run regen + diff in CI.

**Exit criteria:** Running `make -f Makefile.proto generate` on a clean tree
produces zero diff. CI fails if a contributor edits `messages.go` without
regenerating. No service uses the generated output yet.

### Phase D — Hypervisor pilot service

Each PR exposes a single SRPC method through gRPC, end-to-end. Reviewers can
read the SRPC method, the proto, and the gRPC handler in one sitting.

| PR | Title | Files | Net LoC |
|----|-------|-------|---------|
| D0 | `hypervisor/grpcd: skeleton + ListVMs` | `proto/hypervisor/{messages.go tags,grpc/}`, `hypervisor/grpcd/{api,list_vms}.go`, `hypervisor/main.go` (wire-up) | ~400 |
| D1 | `hypervisor/grpcd: GetVmInfo` | `proto/.../grpc/`, `hypervisor/grpcd/get_vm_info.go` | ~150 |
| D2 | `hypervisor/grpcd: StartVm` | as above | ~150 |
| D3 | `hypervisor/grpcd: DestroyVm` | as above | ~150 |
| D4 | `hypervisor/grpcd: CreateVmAsync (unary)` | as above | ~250 |

D0 carries the wiring cost; D1–D4 are pure additions. After D4, the 5 pilot
APIs are live behind a feature flag (`-grpcAddress=` flag, off by default).

**Exit criteria:** End-to-end integration test (`hypervisor/grpcd/e2e_test.go`)
exercises all 5 RPCs against an in-process gRPC server backed by the real
manager. SRPC parity confirmed: same auth, same RBAC, same errors.

### Phase E — Streaming & REST

| PR | Title | Notes |
|----|-------|-------|
| E1 | `hypervisor/grpcd: CreateVm (server-streaming)` | Uses `lib/grpc/streaming.go`. |
| E2 | `lib/grpc: REST auth middleware` | The deferred B3. |
| E3 | `hypervisor: grpc-gateway proto annotations + mux` | Adds `google.api.http` annotations via `@http` tags. |

### Phase F — Additional services

One PR per service to introduce its `grpcd/` directory + first RPC, then
incremental RPC PRs as in Phase D.

- F1: `fleetmanager/grpcd` skeleton + `ListMachines`
- F2: `imageserver/grpcd` skeleton + `GetImage`
- F3: `dominator/grpcd` skeleton + status RPCs
- F4+: per-RPC additions, prioritised by consumer demand


---

## 6. Cross-Cutting Decisions

### Naming
- gRPC service name: `dominator.<svc>.v1.<Svc>Service` (e.g.
  `dominator.hypervisor.v1.HypervisorService`).
- gRPC method names: PascalCase, identical to the SRPC method where possible.
  Where SRPC uses `Get*`/`List*` returning a slice, the proto stays singular
  for `Get*` and plural for `List*`.
- Proto package: `dominator.<svc>.v1`. Go package suffix: `pb`.
- Generated files: `*_gen.go` for converters, `*.pb.go` for protoc output.
  `.gitignore` does **not** exclude either — they are committed.

### Authentication & authorization
- TLS: same `lib/srpc/setupclient` and `lib/srpc/setupserver` cert material.
  gRPC uses `credentials.NewTLS(...)` over the same `*tls.Config`.
- Identity extraction: peer certs → SRPC-equivalent `AuthInformation`, attached
  to the context by `UnaryAuthInterceptor` / `StreamAuthInterceptor`.
- Method-power check: the interceptor consults the same RBAC table (exposed by
  `lib/srpc.CheckAuthorization`). gRPC handlers must NOT re-check.
- Per-request method tag: a single `lib/grpc.MethodInfo` registry maps
  `/dominator.svc.v1.Svc/Method` → `srpc.MethodPower` so existing power tiers
  apply unchanged.

### Rate limiting

- Three levels, all in `lib/srpc/serverutil`, all satisfying
  `srpc.MethodBlocker`: **global** (cap aggregate QPS), **per-user**
  (cap a single principal across all their connections), **per-method**
  (cap an expensive method per principal). The design doc's example config
  maps directly: `rate_limits.{global,per_user,per_method}`.
- Parity with SRPC: every gRPC method that has an SRPC twin must enforce the
  same `MethodBlocker` chain. Services build a `MultiLimiter` once and pass it
  to both `srpc.RegisterNameWithOptions` and the gRPC interceptor — no
  per-protocol policy drift, no protocol-switch bypass.
- Interceptor order: `auth → ratelimit → metrics → handler`. Auth must run
  first so the limiter has a principal to key on; metrics observe the post-
  limiter latency.
- Default key: the auth principal (peer cert CN). The per-method limiter keys
  on `(principal, method)`. The global limiter keys on the singleton bucket.
- Failure mode: `codes.ResourceExhausted` with a `lib/errors.RateLimitedError`
  payload (defined in A2a) carrying `retry_after` and `limit_type` fields.
- The `BlockMethod` cleanup function is invoked in a `defer` after the handler
  returns, including on panic recovery.
- Streams: rate limiting is applied at stream-open. Per-message limits inside
  a stream are out of scope for Phase 0 (tracked as Q11).
- No-op default: services that do not pass a `MethodBlocker` get a no-op
  limiter (matches SRPC behaviour); explicit opt-in keeps existing services
  unaffected when they later expose gRPC.
- Configuration: `requests_per_second` and `burst` per level, supplied via the
  service's existing config-loading mechanism (no new config framework).

### Errors

- `lib/errors.CodedError` (interface, defined in A2a) is the contract handlers
  return. It exposes `Error() string` and `GRPCCode() codes.Code`. The
  `lib/errors` package itself does **not** import `google.golang.org/grpc`;
  it depends only on `google.golang.org/grpc/codes`, which is a tiny leaf
  package, so business-logic packages can return typed errors without pulling
  in the full gRPC stack.
- Standard typed errors shipped in A2a: `NotFoundError`, `AlreadyExistsError`,
  `InvalidArgumentError`, `PermissionDeniedError`, `RateLimitedError`,
  `UnavailableError`, `InternalError`. Each implements `CodedError`.
- `grpc.ErrorToStatus(err)` (A2b) translates any `CodedError` into the matching
  `status.Status` plus `status.WithDetails` carrying typed payloads. For
  unknown error types it falls back to a best-effort string-pattern match
  (per design-doc "stop gap arrangement"), defaulting to `codes.Internal`
  with the message redacted to `"internal error"` and the original logged.
- Clients use `lib/errors.FromStatus(status.Code, status.Details)` to round-trip
  back to typed errors.

### Streaming
- `lib/grpc.StreamingConn` adapts `grpc.ServerStream` to `srpc.StreamingConn`,
  so existing manager code that takes `srpc.StreamingConn` (e.g. `CreateVm`)
  needs no changes.
- Client-streaming and bidi follow the same pattern: handler builds a
  `StreamingConn` and hands it to the existing manager method.

### Transport / serving
- One process serves both protocols. `lib/server.Listen` returns a struct
  exposing `SRPCServer`, `HTTPServer`, and `GRPCServer`. Internally it uses
  `cmux` (or HTTP/2 ALPN switching) on a single TCP port.
- gRPC max message size: 16 MiB request, 16 MiB response (override per-method
  via interceptor where necessary).
- Keepalive defaults: server `MinTime=30s`, `Time=2m`, `Timeout=20s`.

### Observability

- **Unified metric surface across protocols.** SRPC's existing tricorder
  counters in `lib/srpc/server.go` (`numPermittedCalls`, `numDeniedCalls`,
  `failedCallsDistribution`, `successfulCallsDistribution`, plus the per-method
  histograms) are refactored in PR-B0 to carry a `protocol` label
  (`"srpc"` initially). PR-B2 then registers the same metric families under
  the `"grpc"` value via `lib/grpc/metrics.go`. A future REST adapter would
  add `"rest"`. This avoids the dashboard fragmentation that would otherwise
  result from having parallel `/srpc/...` and `/grpc/...` metric trees.
- Per-method counters/histograms live under
  `/{srpc,grpc}/<service>/<method>/{requests,errors,latency_ns}`; the
  `protocol` label lets dashboards `sum by (method)` across both stacks.
- Rate-limit denials are reported by the limiters themselves (PR-0.1) under
  `rate_limit_denied_total{method, limit_type, protocol}` so the cause is
  attributable without joining across services.
- Logs go through the existing `log.DebugLogger` — no separate gRPC logger.
  Interceptors prefix log lines with `grpc:` for grep-friendliness.

### Versioning
- Proto package is `v1`. Breaking changes require a `v2` package and a parallel
  `*_v2.go` handler file. Field-additive changes stay in `v1`.
- Generated files include a `// Code generated by proto-gen. DO NOT EDIT.`
  header; CI rejects manual edits.

---

## 7. Per-PR Acceptance Criteria (template)

Every PR in this plan must satisfy:

1. **Scope**: changes only the files listed in the PR row above. Out-of-scope
   touches → split.
2. **Tests**: new code has unit tests in the same package; coverage delta is
   non-negative for that package.
3. **Backward compatibility**: `go test ./...` passes on the full tree; no
   existing SRPC test changes.
4. **Build artefacts**: for codegen PRs, `make -f Makefile.proto generate`
   produces zero diff after merge.
5. **Docs**: PR description references the spec-kit phase and PR id (e.g.
   "Phase B / B1"). No prose-doc changes required unless the public API in
   `lib/grpc` changes shape.
6. **Reviewer checklist** (in PR body):
   - [ ] No business logic moved into `*/grpcd/`
   - [ ] Errors returned via `lib/errors` types only
   - [ ] No `panic` on bad input — return `codes.InvalidArgument`
   - [ ] Generated files match `proto-gen` output
   - [ ] For any PR exposing a new gRPC method: the same `MethodBlocker`
     implementation is wired to both the SRPC receiver and the gRPC chain

---

## 8. Open Questions

These are decisions that should be made before the affected PR is opened. They
do not block earlier phases.

| # | Question | Affects | Owner | Default if unanswered |
|---|----------|---------|-------|------------------------|
| Q1 | Do we ship grpc-gateway in the same binary or as a sidecar? | Phase E | TBD | In-binary, mounted on the combined listener. |
| Q2 | Do we adopt `buf` for proto linting/breaking-change detection, or stick with `protoc` + custom checks? | Phase C | TBD | `protoc` only; revisit after C4. |
| Q3 | Should `proto-gen` emit an OpenAPI spec alongside the `.proto`? | Phase C/E | TBD | No; defer to grpc-gateway's own emitter. |
| Q4 | Do we vendor `google.golang.org/grpc` and `google.golang.org/protobuf`, or rely on `go mod` only? | Phase A | TBD | `go mod` only; vendor only if a release-engineering need arises. |
| Q5 | Per-service gRPC port vs. shared port via `cmux`? | Phase B/D | TBD | Shared port via `cmux`; service flags override. |
| Q6 | Wire-level compression (gzip)? | Phase B | TBD | Off by default; opt-in per call. |
| Q7 | Where does the `MethodInfo` registry live — generated or hand-maintained? | Phase A/C | TBD | Generated by `proto-gen` from `@grpc` tags in C2. |
| Q8 | Default rate-limit backend: per-user token bucket (in-memory) or pluggable only with no default? | Phase 0 | TBD | Ship a per-user token-bucket default keyed on auth principal; configurable burst/rate per service. |
| Q9 | Are gRPC and SRPC limits shared (one bucket per principal) or independent (separate buckets per protocol)? | Phase 0 | TBD | Shared: one `MethodBlocker` instance passed to both stacks so a misbehaving client cannot double its allowance by switching protocol. |
| Q10 | Should rate limits be runtime-tunable via an admin RPC, or static at startup? | Phase 0 / D | TBD | Static at startup for Phase 0; admin RPC deferred until a service requests it. |
| Q11 | Per-message rate limiting inside long-lived streams? | Phase E | TBD | Out of scope; revisit if a streaming service shows abuse. |
| Q12 | When PR-B0 adds a `protocol` label to existing SRPC counters, do we keep the old un-labelled metric paths as aliases for one release, or break dashboards immediately? | Phase B | TBD | Break immediately; tricorder paths are internal and dashboards are version-controlled in this repo. Coordinate with anyone running ad-hoc queries. |
| Q13 | Do hand-written common converters (Tags, IP, MAC) live in `lib/grpc/common_converters.go` or alongside the generated converters in `proto/<svc>/`? | Phase B/C | TBD | `lib/grpc/common_converters.go` so every service shares one implementation; generated code in `proto/<svc>/` calls into it. |

---

## 9. Branch Hygiene Plan

Once this spec is approved:

1. **Keep open**: `grpc-auth-core` (#226), `grpc-errors` (#227). Rebase onto
   master and push.
2. **Reslice into smaller PRs**: cherry-pick from `grpc-streaming`,
   `grpc-metrics-rest`, `grpc-proto-gen-poc` into the PRs listed in §5. Use
   one branch per PR named `grpc-<phase><n>-<slug>` (e.g. `grpc-b1-streaming`,
   `grpc-c1-proto-gen-scanner`).
3. **Archive**: `grpc-infra`, `grpc-minimal`, `grpc-copy-0`. Tag the heads as
   `archive/<branch>-2026-05-04` and delete the branches. Their content is
   preserved in this spec and in git history.
4. **Document**: update `docs/grpc/README.md` (if present) or create a short
   index pointing at this spec kit and the per-phase PRs.

---

## 10. References

- `design-docs/Dominator/ArchitecturalOverview.md`
- `docs/converter-generation-analysis.md`
- `docs/grpc/GRPC-BRANCHES-SUMMARY.md` (on `grpc-proto-gen-poc`)
- `docs/PROTO-GEN-TOOLING.md` (on `grpc-proto-gen-poc`)
- Design doc:
  https://docs.google.com/document/d/1LukJVDHB1KrGtMocEzsl2d8T89ZACpStFvK4em0rUIw/edit
- PRs: #226 (auth core), #227 (errors)
