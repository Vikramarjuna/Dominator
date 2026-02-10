#!/bin/bash
# Generate gRPC code from proto files
# This script generates:
# 1. Protobuf Go code (.pb.go files)
# 2. gRPC service code (_grpc.pb.go files)
# 3. gRPC-gateway reverse proxy code (.pb.gw.go files)
# Note: Type converters are manually written in converters.go files

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

cd "${REPO_ROOT}"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}==> Generating gRPC code...${NC}"

# Check for required tools
# Default to $HOME/go/bin if tools are not in PATH
PROTOC="${PROTOC:-protoc}"
PROTOC_GEN_GO="${PROTOC_GEN_GO:-}"
PROTOC_GEN_GO_GRPC="${PROTOC_GEN_GO_GRPC:-}"
PROTOC_GEN_GRPC_GATEWAY="${PROTOC_GEN_GRPC_GATEWAY:-}"

# Try to find protoc-gen-go
if [ -z "${PROTOC_GEN_GO}" ]; then
    if command -v protoc-gen-go &> /dev/null; then
        PROTOC_GEN_GO="protoc-gen-go"
    elif [ -x "${HOME}/go/bin/protoc-gen-go" ]; then
        PROTOC_GEN_GO="${HOME}/go/bin/protoc-gen-go"
    fi
fi

# Try to find protoc-gen-go-grpc
if [ -z "${PROTOC_GEN_GO_GRPC}" ]; then
    if command -v protoc-gen-go-grpc &> /dev/null; then
        PROTOC_GEN_GO_GRPC="protoc-gen-go-grpc"
    elif [ -x "${HOME}/go/bin/protoc-gen-go-grpc" ]; then
        PROTOC_GEN_GO_GRPC="${HOME}/go/bin/protoc-gen-go-grpc"
    fi
fi

# Try to find protoc-gen-grpc-gateway
if [ -z "${PROTOC_GEN_GRPC_GATEWAY}" ]; then
    if command -v protoc-gen-grpc-gateway &> /dev/null; then
        PROTOC_GEN_GRPC_GATEWAY="protoc-gen-grpc-gateway"
    elif [ -x "${HOME}/go/bin/protoc-gen-grpc-gateway" ]; then
        PROTOC_GEN_GRPC_GATEWAY="${HOME}/go/bin/protoc-gen-grpc-gateway"
    fi
fi

# Verify protoc is available
if ! command -v "${PROTOC}" &> /dev/null; then
    echo "Error: protoc not found. Please install protoc or set PROTOC environment variable."
    echo "  Example: export PROTOC=/usr/local/compilers/protoc-26.1/bin/protoc"
    exit 1
fi

# Verify protoc-gen-go is available
if ! command -v "${PROTOC_GEN_GO}" &> /dev/null; then
    echo "Error: protoc-gen-go not found."
    echo ""
    echo "Please install all required tools by running:"
    echo "  make install-grpc-tools"
    echo ""
    echo "Or install protoc-gen-go manually:"
    echo "  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"
    echo ""
    echo "Or set environment variable:"
    echo "  export PROTOC_GEN_GO=/usr/local/gotools/go1.24/bin/protoc-gen-go"
    exit 1
fi

# Verify protoc-gen-go-grpc is available
if ! command -v "${PROTOC_GEN_GO_GRPC}" &> /dev/null; then
    echo "Error: protoc-gen-go-grpc not found."
    echo ""
    echo "Please install all required tools by running:"
    echo "  make install-grpc-tools"
    echo ""
    echo "Or install protoc-gen-go-grpc manually:"
    echo "  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"
    echo ""
    echo "Or set environment variable:"
    echo "  export PROTOC_GEN_GO_GRPC=/usr/local/gotools/go1.24/bin/protoc-gen-go-grpc"
    exit 1
fi

# Verify protoc-gen-grpc-gateway is available
if ! command -v "${PROTOC_GEN_GRPC_GATEWAY}" &> /dev/null; then
    echo "Error: protoc-gen-grpc-gateway not found."
    echo ""
    echo "Please install all required tools by running:"
    echo "  make install-grpc-tools"
    echo ""
    echo "Or install protoc-gen-grpc-gateway manually:"
    echo "  go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest"
    echo ""
    echo "Or set environment variable:"
    echo "  export PROTOC_GEN_GRPC_GATEWAY=/usr/local/gotools/go1.24/bin/protoc-gen-grpc-gateway"
    exit 1
fi

echo -e "${GREEN}✓ All required tools found${NC}"

# Find all .proto files
echo -e "${BLUE}==> Discovering proto files...${NC}"
PROTO_FILES=$(find proto -name "*.proto" -type f | sort)

if [ -z "${PROTO_FILES}" ]; then
    echo "Error: No .proto files found in proto/ directory"
    exit 1
fi

echo "Found proto files:"
echo "${PROTO_FILES}" | sed 's/^/  /'

# Step 1: Generate protobuf code for all proto files
echo -e "${BLUE}==> Generating protobuf code...${NC}"
if ! "${PROTOC}" -I proto -I . \
    --plugin=protoc-gen-go="${PROTOC_GEN_GO}" \
    --plugin=protoc-gen-go-grpc="${PROTOC_GEN_GO_GRPC}" \
    --plugin=protoc-gen-grpc-gateway="${PROTOC_GEN_GRPC_GATEWAY}" \
    --go_out=proto \
    --go_opt=paths=source_relative \
    --go-grpc_out=proto \
    --go-grpc_opt=require_unimplemented_servers=false,paths=source_relative \
    --grpc-gateway_out=proto \
    --grpc-gateway_opt=paths=source_relative \
    ${PROTO_FILES}; then
    echo "Error: protoc failed"
    exit 1
fi

echo -e "${GREEN}✓ Generated .pb.go, _grpc.pb.go, and .pb.gw.go files${NC}"

echo -e "${GREEN}==> All gRPC code generated successfully!${NC}"
echo ""
echo "Note: Type converters are manually maintained in converters.go files."
echo "      Proto files are kept clean and language-agnostic."

