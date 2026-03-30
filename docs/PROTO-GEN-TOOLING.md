# Proto-Gen Tooling Guide

## Quick Start

### Generate Proto for Hypervisor

```bash
make -f Makefile.proto proto-gen-hypervisor
```

### Generate All Services

```bash
make -f Makefile.proto proto-gen
```

### CI Check (Verify Up-to-Date)

```bash
make -f Makefile.proto proto-check
```

Or use the script directly:
```bash
./scripts/check-proto-gen.sh
```

---

## Makefile Targets

| Target | Description |
|--------|-------------|
| `proto-gen` | Generate proto for all services |
| `proto-gen-hypervisor` | Generate hypervisor proto only |
| `proto-gen-fleetmanager` | Generate fleetmanager proto (with imports) |
| `proto-check` | Verify generated files are up-to-date (CI) |
| `proto-clean` | Remove all generated files |
| `proto-install-deps` | Install protoc plugins |
| `proto-help` | Show help |

---

## Cross-Package Imports

When one service uses types from another (e.g., FleetManager uses hypervisor.VmInfo):

### 1. Generate dependency first

```bash
make -f Makefile.proto proto-gen-hypervisor
```

### 2. Generate service with imports

```bash
proto-gen \
  --rpcd fleetmanager/rpcd \
  --proto proto/fleetmanager \
  --output proto/fleetmanager/grpc/fleetmanager.proto \
  --imports "hypervisor=proto/hypervisor/grpc/hypervisor.proto"
```

Or add to Makefile.proto:
```makefile
proto-gen-fleetmanager:
	$(PROTO_GEN) \
		--rpcd fleetmanager/rpcd \
		--proto proto/fleetmanager \
		--output proto/fleetmanager/grpc/fleetmanager.proto \
		--imports "hypervisor=proto/hypervisor/grpc/hypervisor.proto"
```

### Generated import statement

```protobuf
// proto/fleetmanager/grpc/fleetmanager.proto
syntax = "proto3";
package fleetmanager;

import "proto/hypervisor/grpc/hypervisor.proto";

message GetMachineInfoResponse {
  hypervisor.VmInfo VmInfo = 1;
}
```

---

## CI Integration

### GitHub Actions Example

```yaml
name: Check Proto

on: [pull_request]

jobs:
  proto-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Install protoc
        run: |
          sudo apt-get update
          sudo apt-get install -y protobuf-compiler
      
      - name: Install protoc plugins
        run: make -f Makefile.proto proto-install-deps
      
      - name: Check proto files
        run: ./scripts/check-proto-gen.sh
```

### Jenkins Example

```groovy
stage('Proto Check') {
    steps {
        sh 'make -f Makefile.proto proto-install-deps'
        sh './scripts/check-proto-gen.sh'
    }
}
```

---

## Command-Line Flags

```bash
proto-gen [flags]

Required:
  --rpcd string       Directory with SRPC methods (e.g., hypervisor/rpcd)
  --proto string      Directory with proto types (e.g., proto/hypervisor)
  --output string     Output .proto file (e.g., proto/hypervisor/grpc/hypervisor.proto)

Optional:
  --converters string Output converters file (e.g., proto/hypervisor/converters_gen.go)
  --imports string    Proto imports: "pkg=path,pkg2=path2"
  --protoc            Run protoc to generate Go code (default: true)
  -v                  Verbose output
```

---

## Adding a New Service

1. **Tag SRPC methods** with `@grpc` annotations

2. **Add Makefile target** in `Makefile.proto`:
```makefile
proto-gen-myservice:
	$(PROTO_GEN) \
		--rpcd myservice/rpcd \
		--proto proto/myservice \
		--output proto/myservice/grpc/myservice.proto \
		--converters proto/myservice/converters_gen.go
```

3. **Add to proto-gen target**:
```makefile
proto-gen: proto-gen-hypervisor proto-gen-myservice
```

4. **Test**:
```bash
make -f Makefile.proto proto-gen-myservice
```

5. **Commit** generated files:
```bash
git add proto/myservice/grpc/*.proto proto/myservice/converters_gen.go
git commit -m "Add gRPC support for myservice"
```

---

## Troubleshooting

### protoc not found

Install protobuf compiler:
- **Ubuntu:** `sudo apt-get install protobuf-compiler`
- **macOS:** `brew install protobuf`
- **Manual:** https://grpc.io/docs/protoc-installation/

### protoc plugins not found

```bash
make -f Makefile.proto proto-install-deps
```

### Generated files out of date in CI

Locally run:
```bash
make -f Makefile.proto proto-gen
git add proto/*/grpc/*.proto proto/*/converters_gen.go
git commit -m "Update generated proto files"
```

### Import path errors

Ensure dependency proto is generated first:
```bash
make -f Makefile.proto proto-gen-hypervisor  # Generate dependency
make -f Makefile.proto proto-gen-fleetmanager # Then service using it
```

