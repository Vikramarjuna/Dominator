#!/bin/bash
# CI script to verify generated proto files are up-to-date

set -e

echo "Checking if generated proto files are up-to-date..."

# Build proto-gen
cd cmd/proto-gen && go build && cd ../..

# Run proto-gen
make -f Makefile.proto proto-gen

# Check for differences
if git diff --exit-code proto/*/grpc/*.proto proto/*/converters_gen.go 2>/dev/null; then
    echo "✓ Generated files are up-to-date"
    exit 0
else
    echo "✗ Generated files are out of date!"
    echo "Run: make -f Makefile.proto proto-gen"
    exit 1
fi

