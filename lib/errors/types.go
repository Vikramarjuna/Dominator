package errors

import (
	"fmt"

	"google.golang.org/grpc/codes"
)

// CodedError is an interface for errors that provide a gRPC status code.
type CodedError interface {
	GrpcCode() codes.Code
}

type NotFoundError struct {
	Resource string
	ID       string
}

func (e *NotFoundError) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("%s %s not found", e.Resource, e.ID)
	}
	return fmt.Sprintf("%s not found", e.Resource)
}

func (e *NotFoundError) GrpcCode() codes.Code { return codes.NotFound }

func NewNotFoundError(resource, id string) *NotFoundError {
	return &NotFoundError{Resource: resource, ID: id}
}

type PermissionDeniedError struct {
	Resource string
	Action   string
	Reason   string
}

func (e *PermissionDeniedError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("permission denied: %s on %s: %s", e.Action, e.Resource, e.Reason)
	}
	return fmt.Sprintf("permission denied: %s on %s", e.Action, e.Resource)
}

func (e *PermissionDeniedError) GrpcCode() codes.Code { return codes.PermissionDenied }

func NewPermissionDeniedError(resource, action, reason string) *PermissionDeniedError {
	return &PermissionDeniedError{Resource: resource, Action: action, Reason: reason}
}

type UnauthenticatedError struct {
	Message string
}

func (e *UnauthenticatedError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("unauthenticated: %s", e.Message)
	}
	return "unauthenticated"
}

func (e *UnauthenticatedError) GrpcCode() codes.Code { return codes.Unauthenticated }

func NewUnauthenticatedError(message string) *UnauthenticatedError {
	return &UnauthenticatedError{Message: message}
}

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

func (e *AlreadyExistsError) GrpcCode() codes.Code { return codes.AlreadyExists }

func NewAlreadyExistsError(resource, id string) *AlreadyExistsError {
	return &AlreadyExistsError{Resource: resource, ID: id}
}

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

func (e *InvalidArgumentError) GrpcCode() codes.Code { return codes.InvalidArgument }

func NewInvalidArgumentError(argument, reason string) *InvalidArgumentError {
	return &InvalidArgumentError{Argument: argument, Reason: reason}
}

type FailedPreconditionError struct {
	Resource string
	State    string
	Reason   string
}

func (e *FailedPreconditionError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("%s %s: %s", e.Resource, e.State, e.Reason)
	}
	return fmt.Sprintf("%s %s", e.Resource, e.State)
}

func (e *FailedPreconditionError) GrpcCode() codes.Code { return codes.FailedPrecondition }

func NewFailedPreconditionError(resource, state, reason string) *FailedPreconditionError {
	return &FailedPreconditionError{Resource: resource, State: state, Reason: reason}
}

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

func (e *UnavailableError) GrpcCode() codes.Code { return codes.Unavailable }

func NewUnavailableError(service, reason string) *UnavailableError {
	return &UnavailableError{Service: service, Reason: reason}
}

type DeadlineExceededError struct {
	Operation string
	Timeout   string
}

func (e *DeadlineExceededError) Error() string {
	if e.Timeout != "" {
		return fmt.Sprintf("%s exceeded deadline of %s", e.Operation, e.Timeout)
	}
	return fmt.Sprintf("%s exceeded deadline", e.Operation)
}

func (e *DeadlineExceededError) GrpcCode() codes.Code { return codes.DeadlineExceeded }

func NewDeadlineExceededError(operation, timeout string) *DeadlineExceededError {
	return &DeadlineExceededError{Operation: operation, Timeout: timeout}
}

type ResourceExhaustedError struct {
	Resource string
	Reason   string
}

func (e *ResourceExhaustedError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("%s exhausted: %s", e.Resource, e.Reason)
	}
	return fmt.Sprintf("%s exhausted", e.Resource)
}

func (e *ResourceExhaustedError) GrpcCode() codes.Code { return codes.ResourceExhausted }

func NewResourceExhaustedError(resource, reason string) *ResourceExhaustedError {
	return &ResourceExhaustedError{Resource: resource, Reason: reason}
}

type InternalError struct {
	Message string
}

func (e *InternalError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("internal error: %s", e.Message)
	}
	return "internal error"
}

func (e *InternalError) GrpcCode() codes.Code { return codes.Internal }

func NewInternalError(message string) *InternalError {
	return &InternalError{Message: message}
}

type UnimplementedError struct {
	Operation string
}

func (e *UnimplementedError) Error() string {
	if e.Operation != "" {
		return fmt.Sprintf("%s not implemented", e.Operation)
	}
	return "not implemented"
}

func (e *UnimplementedError) GrpcCode() codes.Code { return codes.Unimplemented }

func NewUnimplementedError(operation string) *UnimplementedError {
	return &UnimplementedError{Operation: operation}
}
