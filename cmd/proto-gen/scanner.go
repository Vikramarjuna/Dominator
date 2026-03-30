package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"strings"
)

// TypeInfo represents a Go type that should be converted to proto
type TypeInfo struct {
	Name   string
	Fields []FieldInfo
}

// FieldInfo represents a struct field
type FieldInfo struct {
	Name        string
	GoType      string
	ProtoType   string
	ProtoNumber int
	IsRepeated  bool
	IsOptional  bool
}

// scanSpecificTypes scans a package directory for specific type names and recursively finds all referenced types
// Returns both local types and external type references (from other packages)
func scanSpecificTypes(pkgPath string, typeNames map[string]bool) ([]TypeInfo, error) {
	pkgDir, err := getPackageDir(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find package: %w", err)
	}

	// Parse all Go files in the package
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, pkgDir, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse package: %w", err)
	}

	// Build a map of all type definitions
	allTypes := make(map[string]*ast.StructType)
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok || genDecl.Tok != token.TYPE {
					continue
				}

				for _, spec := range genDecl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}

					structType, ok := typeSpec.Type.(*ast.StructType)
					if !ok {
						continue
					}

					allTypes[typeSpec.Name.Name] = structType
				}
			}
		}
	}

	// Recursively collect all types starting from the initial set
	visited := make(map[string]bool)
	var result []TypeInfo
	var toProcess []string

	// Start with the initial type names
	for typeName := range typeNames {
		toProcess = append(toProcess, typeName)
	}

	// Process types recursively
	for len(toProcess) > 0 {
		// Pop a type name
		typeName := toProcess[0]
		toProcess = toProcess[1:]

		// Skip if already visited
		if visited[typeName] {
			continue
		}
		visited[typeName] = true

		// Find the type definition
		structType, exists := allTypes[typeName]
		if !exists {
			// Type not found in this package (might be from another package or builtin)
			continue
		}

		// Extract type info
		typeInfo, err := extractTypeInfo(typeName, structType)
		if err != nil {
			return nil, fmt.Errorf("failed to extract type %s: %w", typeName, err)
		}

		result = append(result, typeInfo)

		// Add all referenced types to the processing queue
		for _, field := range typeInfo.Fields {
			referencedTypes := extractReferencedTypes(field.GoType)
			for _, refType := range referencedTypes {
				if !visited[refType] {
					toProcess = append(toProcess, refType)
				}
			}
		}
	}

	return result, nil
}

// extractReferencedTypes extracts type names referenced by a Go type string
func extractReferencedTypes(goType string) []string {
	var types []string

	// Remove pointer prefix
	goType = strings.TrimPrefix(goType, "*")

	// Handle slices: []Type -> Type
	if strings.HasPrefix(goType, "[]") {
		elemType := strings.TrimPrefix(goType, "[]")
		return extractReferencedTypes(elemType)
	}

	// Handle maps: map[K]V -> [K, V]
	if strings.HasPrefix(goType, "map[") {
		// Extract key and value types
		inner := strings.TrimPrefix(goType, "map[")
		inner = strings.TrimSuffix(inner, "]")
		parts := strings.SplitN(inner, "]", 2)
		if len(parts) == 2 {
			types = append(types, extractReferencedTypes(parts[0])...)
			types = append(types, extractReferencedTypes(parts[1])...)
		}
		return types
	}

	// Skip builtin types
	builtins := map[string]bool{
		"string": true, "bool": true,
		"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
		"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
		"float32": true, "float64": true,
		"byte": true, "rune": true,
	}

	// Skip qualified types from other packages (e.g., net.IP, time.Time)
	if strings.Contains(goType, ".") {
		return types
	}

	// If it's not a builtin and not qualified, it's a custom type in this package
	if !builtins[goType] && goType != "" {
		types = append(types, goType)
	}

	return types
}

// scanPackage scans a package for all types with @grpc tags (old approach, kept for reference)
func scanPackage(pkgPath string) ([]TypeInfo, error) {
	// Get the package directory from GOPATH/GOMOD
	pkgDir, err := getPackageDir(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find package: %w", err)
	}

	// Parse all Go files in the package
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, pkgDir, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse package: %w", err)
	}

	var types []TypeInfo

	// Scan each package
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			// Look for type declarations with @grpc comment
			for _, decl := range file.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok || genDecl.Tok != token.TYPE {
					continue
				}

				// Check if this type has @grpc tag in comments
				if !hasGrpcTag(genDecl.Doc) {
					continue
				}

				// Process each type spec
				for _, spec := range genDecl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}

					structType, ok := typeSpec.Type.(*ast.StructType)
					if !ok {
						continue
					}

					// Extract type info
					typeInfo, err := extractTypeInfo(typeSpec.Name.Name, structType)
					if err != nil {
						return nil, fmt.Errorf("failed to extract type %s: %w", typeSpec.Name.Name, err)
					}

					types = append(types, typeInfo)
				}
			}
		}
	}

	return types, nil
}

func hasGrpcTag(doc *ast.CommentGroup) bool {
	if doc == nil {
		return false
	}
	for _, comment := range doc.List {
		if strings.Contains(comment.Text, "@grpc") {
			return true
		}
	}
	return false
}

func extractTypeInfo(name string, structType *ast.StructType) (TypeInfo, error) {
	typeInfo := TypeInfo{
		Name:   name,
		Fields: []FieldInfo{},
	}

	fieldNumbers := make(map[int]string) // Track explicit field numbers to detect duplicates
	nextAutoNumber := 1                  // Auto-assign starting from 1

	for _, field := range structType.Fields.List {
		// Skip fields without names (embedded fields)
		if len(field.Names) == 0 {
			continue
		}

		fieldName := field.Names[0].Name

		// Skip Error field (SRPC-only, not in proto)
		if fieldName == "Error" {
			continue
		}

		// Extract proto tag
		protoNum, skip, err := extractProtoTag(field.Tag, fieldName)
		if err != nil {
			return typeInfo, fmt.Errorf("field %s: %w", fieldName, err)
		}
		if skip {
			continue // Field has proto:"-" tag
		}

		// Auto-assign field number if not explicitly set
		if protoNum == 0 {
			// Find next available number
			for {
				if _, exists := fieldNumbers[nextAutoNumber]; !exists {
					protoNum = nextAutoNumber
					nextAutoNumber++
					break
				}
				nextAutoNumber++
			}
		} else {
			// Check for duplicate explicit field numbers
			if existingField, exists := fieldNumbers[protoNum]; exists {
				return typeInfo, fmt.Errorf("duplicate field number %d: %s and %s", protoNum, existingField, fieldName)
			}
		}

		fieldNumbers[protoNum] = fieldName

		// Determine Go type and proto type
		goType := exprToString(field.Type)
		protoType, isRepeated := mapGoTypeToProto(goType)

		fieldInfo := FieldInfo{
			Name:        fieldName,
			GoType:      goType,
			ProtoType:   protoType,
			ProtoNumber: protoNum,
			IsRepeated:  isRepeated,
		}

		typeInfo.Fields = append(typeInfo.Fields, fieldInfo)
	}

	return typeInfo, nil
}

func extractProtoTag(tag *ast.BasicLit, fieldName string) (protoNum int, skip bool, err error) {
	// If no tag at all, auto-assign field number (return 0)
	if tag == nil {
		return 0, false, nil
	}

	// Parse struct tag
	tagValue := strings.Trim(tag.Value, "`")
	tags := reflect.StructTag(tagValue)
	protoTag := tags.Get("proto")

	// If no proto tag, auto-assign field number (return 0)
	if protoTag == "" {
		return 0, false, nil
	}

	// Check for proto:"-" (skip this field)
	if protoTag == "-" {
		return 0, true, nil
	}

	// Parse explicit field number
	var num int
	_, parseErr := fmt.Sscanf(protoTag, "%d", &num)
	if parseErr != nil {
		return 0, false, fmt.Errorf("invalid proto tag %q: %w", protoTag, parseErr)
	}

	if num <= 0 {
		return 0, false, fmt.Errorf("proto field number must be positive, got %d", num)
	}

	return num, false, nil
}

func exprToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		// Simple type: string, int, bool, etc.
		return e.Name

	case *ast.SelectorExpr:
		// Qualified type: net.IP, time.Time, etc.
		pkg := exprToString(e.X)
		return pkg + "." + e.Sel.Name

	case *ast.StarExpr:
		// Pointer type: *string
		return "*" + exprToString(e.X)

	case *ast.ArrayType:
		// Array/slice type: []string
		if e.Len == nil {
			// Slice
			return "[]" + exprToString(e.Elt)
		}
		// Array (treat as slice for proto)
		return "[]" + exprToString(e.Elt)

	case *ast.MapType:
		// Map type: map[string]int
		key := exprToString(e.Key)
		val := exprToString(e.Value)
		return fmt.Sprintf("map[%s]%s", key, val)

	default:
		return fmt.Sprintf("unknown<%T>", expr)
	}
}

func mapGoTypeToProto(goType string) (protoType string, isRepeated bool) {
	// Handle slices (repeated fields)
	if strings.HasPrefix(goType, "[]") {
		elemType := strings.TrimPrefix(goType, "[]")
		protoElemType, _ := mapGoTypeToProto(elemType)
		return protoElemType, true
	}

	// Handle maps
	if strings.HasPrefix(goType, "map[") {
		// Extract key and value types
		// map[string]int -> map<string, int64>
		inner := strings.TrimPrefix(goType, "map[")
		inner = strings.TrimSuffix(inner, "]")
		parts := strings.SplitN(inner, "]", 2)
		if len(parts) == 2 {
			keyType, _ := mapGoTypeToProto(parts[0])
			valType, _ := mapGoTypeToProto(parts[1])
			return fmt.Sprintf("map<%s, %s>", keyType, valType), false
		}
	}

	// Handle pointers (strip them)
	if strings.HasPrefix(goType, "*") {
		return mapGoTypeToProto(strings.TrimPrefix(goType, "*"))
	}

	// Map basic Go types to proto types
	switch goType {
	case "string":
		return "string", false
	case "bool":
		return "bool", false
	case "int", "int32":
		return "int32", false
	case "int64":
		return "int64", false
	case "uint", "uint32":
		return "uint32", false
	case "uint64":
		return "uint64", false
	case "uint8", "byte":
		return "uint32", false // Proto doesn't have uint8, use uint32
	case "uint16":
		return "uint32", false
	case "int8":
		return "int32", false
	case "int16":
		return "int32", false
	case "float32":
		return "float", false
	case "float64":
		return "double", false

	// Handle common Dominator types
	case "net.IP":
		return "bytes", false
	case "net.HardwareAddr":
		return "bytes", false
	case "time.Time":
		return "int64", false // Unix timestamp
	case "time.Duration":
		return "int64", false // Nanoseconds

	// Dominator enum types (these are uint types, map to uint32)
	case "ConsoleType", "State", "FirmwareType", "MachineType", "VolumeFormat",
		"WatchdogAction", "WatchdogModel", "VolumeInterface", "VolumeType", "Interface", "Type":
		return "uint32", false

	// Special external types
	case "tags.Tags":
		return "map<string, string>", false
	case "filter.Filter":
		// Filter is complex - for now, skip it or use bytes
		return "bytes", false

	// Custom types - assume they're messages
	default:
		// If it contains a dot, it's a qualified type (e.g., filter.Filter)
		if strings.Contains(goType, ".") {
			parts := strings.Split(goType, ".")
			return parts[len(parts)-1], false
		}
		// Otherwise, assume it's a message type in the same package
		return goType, false
	}
}

func getPackageDir(pkgPath string) (string, error) {
	// Convert to absolute path
	return filepath.Abs(pkgPath)
}
