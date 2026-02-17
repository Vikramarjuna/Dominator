#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

# Check required tools
command -v protoc &> /dev/null || { echo "Error: protoc not found"; exit 1; }
command -v protoc-gen-go &> /dev/null || { echo "Error: protoc-gen-go not found. Run: make install-grpc-tools"; exit 1; }
command -v protoc-gen-go-grpc &> /dev/null || { echo "Error: protoc-gen-go-grpc not found. Run: make install-grpc-tools"; exit 1; }
command -v protoc-gen-grpc-gateway &> /dev/null || { echo "Error: protoc-gen-grpc-gateway not found. Run: make install-grpc-tools"; exit 1; }

# Find googleapis protos from grpc-gateway module
GOPATH="${GOPATH:-$(go env GOPATH)}"
GOOGLEAPIS_DIR=$(find "$GOPATH/pkg/mod/github.com/grpc-ecosystem" -type d -name "googleapis" -path "*/grpc-gateway@*/third_party/*" 2>/dev/null | sort -V | tail -1)
[ -z "$GOOGLEAPIS_DIR" ] && { echo "Error: googleapis protos not found. Run 'go mod download' first."; exit 1; }

# Find all proto files
PROTO_FILES=$(find proto -name "*.proto" -type f | sort)
[ -z "$PROTO_FILES" ] && { echo "Error: No .proto files found"; exit 1; }

echo "Generating gRPC code..."
protoc \
    --proto_path=. \
    --proto_path="$GOOGLEAPIS_DIR" \
    --go_out=. \
    --go_opt=paths=source_relative \
    --go-grpc_out=. \
    --go-grpc_opt=require_unimplemented_servers=false,paths=source_relative \
    --grpc-gateway_out=. \
    --grpc-gateway_opt=paths=source_relative \
    $PROTO_FILES

# Generate HTTP path to gRPC method mappings for REST auth
echo "Generating REST path mappings..."
"$SCRIPT_DIR/generate-rest-routes.sh"

echo "Done."