package main

import (
	"flag"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	rpcdDir      = flag.String("rpcd", "", "Directory containing SRPC method implementations (e.g., hypervisor/rpcd)")
	protoDir     = flag.String("proto", "", "Directory containing proto type definitions (e.g., proto/hypervisor)")
	outputProto  = flag.String("output", "", "Output .proto file (e.g., proto/hypervisor/grpc/hypervisor.proto)")
	converterOut = flag.String("converters", "", "Output file for generated converters (optional)")
	imports      = flag.String("imports", "", "Proto imports in format 'pkg=path,pkg2=path2' (e.g., 'hypervisor=proto/hypervisor/grpc/hypervisor.proto')")
	runProtoc    = flag.Bool("protoc", true, "Run protoc to generate Go code (default: true)")
	verbose      = flag.Bool("v", false, "Verbose output")
)

func main() {
	flag.Parse()

	if *rpcdDir == "" {
		log.Fatal("--rpcd is required")
	}
	if *protoDir == "" {
		log.Fatal("--proto is required")
	}
	if *outputProto == "" {
		log.Fatal("--output is required")
	}

	// Parse imports
	protoImports := parseImports(*imports)

	if *verbose {
		log.Printf("Scanning rpcd directory: %s", *rpcdDir)
		log.Printf("Proto types directory: %s", *protoDir)
		log.Printf("Output proto file: %s", *outputProto)
		if len(protoImports) > 0 {
			log.Printf("Proto imports: %v", protoImports)
		}
	}

	// Step 1: Parse rpcd directory for methods with @grpc and @http annotations
	methods, err := parseMethods(*rpcdDir)
	if err != nil {
		log.Fatalf("Failed to parse methods: %v", err)
	}

	if *verbose {
		log.Printf("Found %d methods with @grpc tags", len(methods))
		for _, m := range methods {
			log.Printf("  - %s -> %s (%s, %s)", m.Name, m.GrpcName, m.RequestType, m.ResponseType)
		}
	}

	// Step 2: Collect all types referenced by methods
	typeNames := collectReferencedTypes(methods)
	if *verbose {
		log.Printf("Methods reference %d types", len(typeNames))
	}

	// Step 3: Scan proto directory for those specific types
	types, err := scanSpecificTypes(*protoDir, typeNames)
	if err != nil {
		log.Fatalf("Failed to scan types: %v", err)
	}

	if *verbose {
		log.Printf("Found %d types", len(types))
		for _, t := range types {
			log.Printf("  - %s (%d fields)", t.Name, len(t.Fields))
		}
	}

	// Step 4: Generate .proto file
	if err := generateProto(*outputProto, types, methods, protoImports); err != nil {
		log.Fatalf("Failed to generate proto: %v", err)
	}

	if *verbose {
		log.Printf("Generated proto file: %s", *outputProto)
	}

	// Step 5: Generate converters (if requested)
	if *converterOut != "" {
		if err := generateConverters(*converterOut, types); err != nil {
			log.Fatalf("Failed to generate converters: %v", err)
		}
		if *verbose {
			log.Printf("Generated converters: %s", *converterOut)
		}
	}

	// Step 6: Run protoc to generate Go code (if requested)
	if *runProtoc {
		if err := invokeProtoc(*outputProto); err != nil {
			log.Fatalf("Failed to run protoc: %v", err)
		}
		if *verbose {
			log.Printf("Generated Go code from proto")
		}
	}

	log.Printf("Success! Generated proto file: %s", *outputProto)
}

func invokeProtoc(protoFile string) error {
	// Get the directory containing the proto file
	protoDir := filepath.Dir(protoFile)

	// Run protoc from the proto directory
	cmd := exec.Command("protoc",
		"--go_out=.",
		"--go_opt=paths=source_relative",
		"--go-grpc_out=.",
		"--go-grpc_opt=paths=source_relative",
		filepath.Base(protoFile))

	cmd.Dir = protoDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("protoc failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

func collectReferencedTypes(methods []MethodInfo) map[string]bool {
	types := make(map[string]bool)
	for _, method := range methods {
		types[method.RequestType] = true
		types[method.ResponseType] = true
	}
	return types
}

// parseImports parses the imports flag format: "pkg=path,pkg2=path2"
// Returns map of package name -> proto file path
func parseImports(importsStr string) map[string]string {
	imports := make(map[string]string)

	if importsStr == "" {
		return imports
	}

	// Split by comma
	pairs := strings.Split(importsStr, ",")
	for _, pair := range pairs {
		// Split by equals
		parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(parts) == 2 {
			pkg := strings.TrimSpace(parts[0])
			path := strings.TrimSpace(parts[1])
			imports[pkg] = path
		}
	}

	return imports
}

func getPackageName(path string) string {
	// Extract package name from path
	parts := strings.Split(filepath.Clean(path), string(filepath.Separator))
	if len(parts) == 0 {
		return "unknown"
	}
	return parts[len(parts)-1]
}
