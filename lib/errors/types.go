package errors

import (
	"fmt"

	"google.golang.org/grpc/codes"
)

// Typed errors for better error handling and gRPC status code mapping.
// These errors implement the error interface and can be used with errors.As().
// Each error type knows its own gRPC status code via the GRPCCode() method.
//
// Note: We use google.golang.org/grpc/codes because:
// 1. It's an industry-standard enum package (lightweight, stable API)
// 2. Only imports standard library (fmt, strconv)
// 3. Adds ~1-2 KB to binary (just enum constants)
// 4. Automatically maps to HTTP status codes via grpc-gateway
// 5. Better than reinventing our own error code system

// NotFoundError indicates a resource was not found.
type NotFoundError struct {
	Resource string // e.g., "machine", "hypervisor", "subnet"
	ID       string // Optional identifier (hostname, IP, etc.)
}

func (e *NotFoundError) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("%s %s not found", e.Resource, e.ID)
	}
	return fmt.Sprintf("%s not found", e.Resource)
}

// GRPCCode returns the gRPC status code for this error.
func (e *NotFoundError) GRPCCode() codes.Code {
	return codes.NotFound
}

// NewNotFoundError creates a new NotFoundError.
func NewNotFoundError(resource, id string) *NotFoundError {
	return &NotFoundError{Resource: resource, ID: id}
}

// PermissionDeniedError indicates the caller lacks permission for an operation.
type PermissionDeniedError struct {
	Resource string // e.g., "machine", "hypervisor"
	Action   string // e.g., "update", "delete", "power on"
	Reason   string // Optional additional context
}

func (e *PermissionDeniedError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("permission denied: %s on %s: %s", e.Action, e.Resource, e.Reason)
	}
	return fmt.Sprintf("permission denied: %s on %s", e.Action, e.Resource)
}

// GRPCCode returns the gRPC status code for this error.
func (e *PermissionDeniedError) GRPCCode() codes.Code {
	return codes.PermissionDenied
}

// NewPermissionDeniedError creates a new PermissionDeniedError.
func NewPermissionDeniedError(resource, action, reason string) *PermissionDeniedError {
	return &PermissionDeniedError{Resource: resource, Action: action, Reason: reason}
}

// UnauthenticatedError indicates the caller is not authenticated.
type UnauthenticatedError struct {
	Message string
}

func (e *UnauthenticatedError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("unauthenticated: %s", e.Message)
	}
	return "unauthenticated"
}

// GRPCCode returns the gRPC status code for this error.
func (e *UnauthenticatedError) GRPCCode() codes.Code {
	return codes.Unauthenticated
}

// NewUnauthenticatedError creates a new UnauthenticatedError.
func NewUnauthenticatedError(message string) *UnauthenticatedError {
	return &UnauthenticatedError{Message: message}
}

// AlreadyExistsError indicates a resource already exists.
type AlreadyExistsError struct {
	Resource string
	ID       string
}

func (e *AlreadyExistsError) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("%s %s already exists", e.Resource, e.ID)
	}
	return fmt.Sprintf("%s already exists", e.Resource)
}

// GRPCCode returns the gRPC status code for this error.
func (e *AlreadyExistsError) GRPCCode() codes.Code {
	return codes.AlreadyExists
}

// NewAlreadyExistsError creates a new AlreadyExistsError.
func NewAlreadyExistsError(resource, id string) *AlreadyExistsError {
	return &AlreadyExistsError{Resource: resource, ID: id}
}

// InvalidArgumentError indicates an invalid argument was provided.
type InvalidArgumentError struct {
	Argument string
	Reason   string
}

func (e *InvalidArgumentError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("invalid argument %s: %s", e.Argument, e.Reason)
	}
	return fmt.Sprintf("invalid argument: %s", e.Argument)
}

// GRPCCode returns the gRPC status code for this error.
func (e *InvalidArgumentError) GRPCCode() codes.Code {
	return codes.InvalidArgument
}

// NewInvalidArgumentError creates a new InvalidArgumentError.
func NewInvalidArgumentError(argument, reason string) *InvalidArgumentError {
	return &InvalidArgumentError{Argument: argument, Reason: reason}
}

// UnavailableError indicates a service or resource is temporarily unavailable.
type UnavailableError struct {
	Service string
	Reason  string
}

func (e *UnavailableError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("%s unavailable: %s", e.Service, e.Reason)
	}
	return fmt.Sprintf("%s unavailable", e.Service)
}

// GRPCCode returns the gRPC status code for this error.
func (e *UnavailableError) GRPCCode() codes.Code {
	return codes.Unavailable
}

// NewUnavailableError creates a new UnavailableError.
func NewUnavailableError(service, reason string) *UnavailableError {
	return &UnavailableError{Service: service, Reason: reason}
}

// DeadlineExceededError indicates an operation timed out.
type DeadlineExceededError struct {
	Operation string
	Timeout   string // e.g., "30s", "5m"
}

func (e *DeadlineExceededError) Error() string {
	if e.Timeout != "" {
		return fmt.Sprintf("%s exceeded deadline of %s", e.Operation, e.Timeout)
	}
	return fmt.Sprintf("%s exceeded deadline", e.Operation)
}

// GRPCCode returns the gRPC status code for this error.
func (e *DeadlineExceededError) GRPCCode() codes.Code {
	return codes.DeadlineExceeded
}

// NewDeadlineExceededError creates a new DeadlineExceededError.
func NewDeadlineExceededError(operation, timeout string) *DeadlineExceededError {
	return &DeadlineExceededError{Operation: operation, Timeout: timeout}
}
