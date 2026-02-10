# Converter Generation Analysis

## Current Situation

We have manual converter functions in `fleetmanager/grpcd/converters.go` that convert between:
- **SRPC types** (`proto/fleetmanager/messages.go`) - Hand-written Go structs
- **gRPC types** (`proto/fleetmanager/grpc/*.pb.go`) - Auto-generated from proto files

**Current converter code:** ~143 lines of hand-written conversion logic

---

## Problem

Maintaining two sets of type definitions creates duplication:
1. **SRPC types** - `proto/fleetmanager/messages.go`
2. **gRPC types** - `proto/fleetmanager/grpc/fleetmanager.proto`

When we add/modify fields, we must update:
- ✅ SRPC message definition
- ✅ gRPC proto definition  
- ✅ Converter functions (both directions)
- ✅ Tests

This is error-prone and tedious.

---

## Evaluated Solution: mog

**mog** is HashiCorp's tool for auto-generating converters between protobuf types and Go structs.

### How mog Works

1. Add annotations to proto files:
```protobuf
// mog annotation:
//
// target=github.com/Cloud-Foundations/Dominator/proto/fleetmanager.Machine
// output=converters.gen.go
// name=FleetManager
message Machine {
  string gateway_subnet_id = 1;
  // mog: func-to=NetworkEntryToProto func-from=NetworkEntryFromProto
  NetworkEntry ipmi = 2;
  // ... more fields
}
```

2. Run mog to generate converters:
```bash
mog -source ./proto/fleetmanager/grpc
```

3. Get auto-generated `converters.gen.go` with functions like:
```go
func MachineToProto(s *fleetmanager.Machine) *pb.Machine { ... }
func MachineFromProto(s *pb.Machine) *fleetmanager.Machine { ... }
```

### Challenges with mog

#### 1. **Complex Type Conversions**

Our converters have non-trivial type conversions:

| SRPC Type | gRPC Type | Conversion |
|-----------|-----------|------------|
| `net.IP` | `bytes` | `[]byte(ip)` / `net.IP(bytes)` |
| `net.HardwareAddr` | `bytes` | `[]byte(addr)` / `net.HardwareAddr(bytes)` |
| `uint` | `uint32` | `uint32(v)` / `uint(v)` |
| `tags.Tags` | `common.Tags` | Wrapper struct conversion |
| `tags.MatchTags` | `common.MatchTags` | Map with nested struct |
| `hyper_proto.State` (enum) | `string` | `.String()` / parse |

**mog requires helper functions for each of these**, which we'd still need to write manually.

#### 2. **Embedded Structs**

SRPC `Machine` has embedded `NetworkEntry`:
```go
type Machine struct {
    NetworkEntry  // Embedded
    IPMI NetworkEntry
    // ...
}
```

gRPC proto doesn't support embedding:
```protobuf
message Machine {
    NetworkEntry network_entry = 5;  // Not embedded
    NetworkEntry ipmi = 2;
}
```

**mog struggles with embedded vs non-embedded field mapping.**

#### 3. **Nested Conversions**

Many types reference other types that also need conversion:
- `Machine` → contains `NetworkEntry`, `Tags`
- `NetworkEntry` → contains `net.IP`, `net.HardwareAddr`
- `VmInfo` → contains `Tags`, `State` enum, nested `Address.IpAddress`

**mog requires annotations for every level**, making proto files verbose.

#### 4. **Learning Curve & Maintenance**

- Team needs to learn mog annotation syntax
- Proto files become cluttered with annotations
- Debugging generated code is harder than reading manual converters
- mog is HashiCorp-specific, not widely adopted

---

## Alternative: Keep Manual Converters

### Pros

✅ **Simple & Readable** - Clear, straightforward Go code  
✅ **Full Control** - Handle edge cases easily  
✅ **No Dependencies** - No external code generation tools  
✅ **Easy to Debug** - Step through conversion logic  
✅ **Team Familiarity** - Standard Go patterns  

### Cons

❌ **Manual Maintenance** - Must update when types change  
❌ **Potential for Errors** - Forgetting to update converters  
❌ **Boilerplate Code** - ~143 lines of repetitive code  

---

## Recommendation

### **Keep Manual Converters (for now)**

**Reasons:**

1. **Small Scale** - Only ~143 lines of converter code for 8 RPC methods
2. **Complex Conversions** - Many non-trivial type mappings that mog can't auto-generate anyway
3. **Embedded Structs** - SRPC uses embedding, which mog doesn't handle well
4. **Simplicity** - Manual code is easier to understand and debug
5. **No Benefit** - mog would still require ~50-70 lines of helper functions, saving only ~70 lines

### **When to Reconsider mog:**

Reconsider if:
- ✅ Number of RPC methods grows to 20+ (converter code becomes 300+ lines)
- ✅ Types become simpler (fewer net.IP, embedded structs, etc.)
- ✅ Team becomes familiar with mog from other projects
- ✅ We standardize on proto as single source of truth (deprecate SRPC)

---

## Better Long-Term Solution

Instead of auto-generating converters, consider:

### **Option: Deprecate SRPC, Use Proto as Single Source of Truth**

1. Make `.proto` files the canonical type definitions
2. Generate `.pb.go` with appropriate struct tags for gob/json encoding
3. Use `.pb.go` types for BOTH SRPC and gRPC
4. Eliminate converters entirely

**Benefits:**
- ✅ No duplicate type definitions
- ✅ No converters needed
- ✅ Single source of truth
- ✅ Simpler codebase

**Challenges:**
- ❌ Breaking change for existing SRPC clients
- ❌ Proto doesn't support embedded structs
- ❌ Proto types are more verbose than hand-written Go structs
- ❌ Requires migration plan

---

## Current Status

**Decision:** Keep manual converters in `fleetmanager/grpcd/converters.go`

**Rationale:**
- Small codebase (~143 lines)
- Complex type conversions
- mog provides minimal benefit
- Manual code is clearer

**Future:** Revisit if converter code grows significantly or if we deprecate SRPC entirely.

---

## References

- [mog GitHub](https://github.com/hashicorp/mog)
- [HashiCorp Consul mog usage](https://github.com/hashicorp/consul/tree/main/proto)
- [Current converters](../fleetmanager/grpcd/converters.go)

