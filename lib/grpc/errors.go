package grpc

import (
	"errors"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CodedError is an interface for errors that can provide their gRPC status code.
// This allows error types in lib/errors to remain gRPC-agnostic while still
// being able to specify their appropriate gRPC status codes.
type CodedError interface {
	GRPCCode() codes.Code
}

// ErrorToStatus converts internal errors to appropriate gRPC status codes.
// This function uses a hybrid approach:
// 1. First, it checks if the error implements CodedError interface (typed errors)
// 2. If not, it falls back to pattern matching on error messages
//
// This allows gradual migration from untyped to typed errors.
func ErrorToStatus(err error) error {
	if err == nil {
		return nil
	}

	// Check if error implements CodedError interface (typed errors)
	var codedErr CodedError
	if errors.As(err, &codedErr) {
		return status.Error(codedErr.GRPCCode(), err.Error())
	}

	// Fall back to pattern matching for untyped errors
	msg := err.Error()
	patternMappings := []struct {
		patterns []string
		code     codes.Code
	}{
		{[]string{"not found", "does not exist", "no such", "unknown"}, codes.NotFound},
		{[]string{"permission denied", "access denied", "forbidden"}, codes.PermissionDenied},
		{[]string{"unauthenticated", "not authenticated", "authentication required"}, codes.Unauthenticated},
		{[]string{"already exists", "duplicate", "conflict"}, codes.AlreadyExists},
		{[]string{"invalid", "malformed", "bad request", "illegal"}, codes.InvalidArgument},
		{[]string{"unavailable", "service down", "connection refused"}, codes.Unavailable},
		{[]string{"timeout", "deadline exceeded", "timed out"}, codes.DeadlineExceeded},
	}

	for _, mapping := range patternMappings {
		if containsAny(msg, mapping.patterns) {
			return status.Error(mapping.code, msg)
		}
	}

	// Default to Internal for unknown errors
	return status.Error(codes.Internal, msg)
}

// containsAny checks if the string contains any of the substrings (case-insensitive).
func containsAny(s string, substrings []string) bool {
	lower := strings.ToLower(s)
	for _, substr := range substrings {
		if strings.Contains(lower, substr) {
			return true
		}
	}
	return false
}
