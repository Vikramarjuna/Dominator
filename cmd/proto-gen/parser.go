package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

// MethodInfo represents an RPC method
type MethodInfo struct {
	Name         string
	GrpcName     string // May differ from Name
	RequestType  string
	ResponseType string
	HttpMethod   string // GET, POST, etc.
	HttpPath     string // /v1/vms/{id}
	HttpBody     string // "*" or field name
	IsStreaming  bool
}

func parseMethods(rpcdDir string) ([]MethodInfo, error) {
	// Check if rpcd directory exists
	if _, err := os.Stat(rpcdDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("rpcd directory not found: %s\nThe rpcd directory should contain SRPC method implementations with @grpc tags", rpcdDir)
	}

	// Parse all Go files in rpcd
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, rpcdDir, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse rpcd: %w", err)
	}

	var methods []MethodInfo

	// Scan each package
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			// Look for function declarations with @grpc comment
			for _, decl := range file.Decls {
				funcDecl, ok := decl.(*ast.FuncDecl)
				if !ok {
					continue
				}

				// Check if this function has @grpc tag
				grpcTag, httpTag := extractTags(funcDecl.Doc)
				if grpcTag == "" {
					continue
				}

				// Extract method info
				methodInfo, err := extractMethodInfo(funcDecl, grpcTag, httpTag)
				if err != nil {
					return nil, fmt.Errorf("failed to extract method %s: %w", funcDecl.Name.Name, err)
				}

				methods = append(methods, methodInfo)
			}
		}
	}

	return methods, nil
}

func extractTags(doc *ast.CommentGroup) (grpcTag, httpTag string) {
	if doc == nil {
		return "", ""
	}

	for _, comment := range doc.List {
		text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))

		// Check for @grpc tag
		if strings.HasPrefix(text, "@grpc") {
			grpcTag = strings.TrimSpace(strings.TrimPrefix(text, "@grpc"))
			if grpcTag == "" {
				grpcTag = "default" // Use function name
			}
		}

		// Check for @http tag
		if strings.HasPrefix(text, "@http") {
			httpTag = strings.TrimSpace(strings.TrimPrefix(text, "@http"))
		}
	}

	return grpcTag, httpTag
}

func extractMethodInfo(funcDecl *ast.FuncDecl, grpcTag, httpTag string) (MethodInfo, error) {
	methodInfo := MethodInfo{
		Name:     funcDecl.Name.Name,
		GrpcName: grpcTag,
	}

	if methodInfo.GrpcName == "default" {
		methodInfo.GrpcName = methodInfo.Name
	}

	// Extract request and response types from function signature
	// Expected signature: func (t *srpcType) MethodName(conn *srpc.Conn, req *RequestType, resp *ResponseType) error
	if funcDecl.Type.Params == nil || len(funcDecl.Type.Params.List) < 3 {
		return methodInfo, fmt.Errorf("invalid method signature - expected (conn, req, resp)")
	}

	// Second parameter is request type
	reqParam := funcDecl.Type.Params.List[1]
	methodInfo.RequestType = extractTypeName(reqParam.Type)

	// Third parameter is response type
	respParam := funcDecl.Type.Params.List[2]
	methodInfo.ResponseType = extractTypeName(respParam.Type)

	// Parse HTTP annotation
	if httpTag != "" {
		if err := parseHttpTag(httpTag, &methodInfo); err != nil {
			return methodInfo, fmt.Errorf("invalid @http tag: %w", err)
		}
	}

	return methodInfo, nil
}

func extractTypeName(expr ast.Expr) string {
	// Handle pointer types
	if starExpr, ok := expr.(*ast.StarExpr); ok {
		expr = starExpr.X
	}

	// Handle selector expressions (pkg.Type)
	if selExpr, ok := expr.(*ast.SelectorExpr); ok {
		return selExpr.Sel.Name
	}

	// Handle identifiers
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}

	return "unknown"
}

func parseHttpTag(httpTag string, methodInfo *MethodInfo) error {
	// Parse HTTP tag: "GET /v1/vms/{id}"
	parts := strings.Fields(httpTag)
	if len(parts) < 2 {
		return fmt.Errorf("invalid format - expected 'METHOD /path'")
	}

	methodInfo.HttpMethod = parts[0]
	methodInfo.HttpPath = parts[1]

	// Check for body specification
	if len(parts) > 2 && parts[2] == "body:" {
		if len(parts) > 3 {
			methodInfo.HttpBody = parts[3]
		}
	} else {
		// Default body handling
		if methodInfo.HttpMethod == "POST" || methodInfo.HttpMethod == "PUT" || methodInfo.HttpMethod == "PATCH" {
			methodInfo.HttpBody = "*"
		}
	}

	return nil
}
