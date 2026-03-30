package errors

import (
	"fmt"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DominatorError represents an error with associated gRPC and HTTP status codes
type DominatorError struct {
	message    string
	grpcCode   codes.Code
	httpStatus int
	cause      error
}

func (e *DominatorError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.message, e.cause)
	}
	return e.message
}

func (e *DominatorError) Unwrap() error {
	return e.cause
}

func (e *DominatorError) GRPCCode() codes.Code {
	return e.grpcCode
}

func (e *DominatorError) HTTPStatus() int {
	return e.httpStatus
}

// Error constructors

func NotFound(format string, args ...interface{}) error {
	return &DominatorError{
		message:    fmt.Sprintf(format, args...),
		grpcCode:   codes.NotFound,
		httpStatus: http.StatusNotFound,
	}
}

func InvalidArgument(format string, args ...interface{}) error {
	return &DominatorError{
		message:    fmt.Sprintf(format, args...),
		grpcCode:   codes.InvalidArgument,
		httpStatus: http.StatusBadRequest,
	}
}

func PermissionDenied(format string, args ...interface{}) error {
	return &DominatorError{
		message:    fmt.Sprintf(format, args...),
		grpcCode:   codes.PermissionDenied,
		httpStatus: http.StatusForbidden,
	}
}

func AlreadyExists(format string, args ...interface{}) error {
	return &DominatorError{
		message:    fmt.Sprintf(format, args...),
		grpcCode:   codes.AlreadyExists,
		httpStatus: http.StatusConflict,
	}
}

func Internal(format string, args ...interface{}) error {
	return &DominatorError{
		message:    fmt.Sprintf(format, args...),
		grpcCode:   codes.Internal,
		httpStatus: http.StatusInternalServerError,
	}
}

func Unauthenticated(format string, args ...interface{}) error {
	return &DominatorError{
		message:    fmt.Sprintf(format, args...),
		grpcCode:   codes.Unauthenticated,
		httpStatus: http.StatusUnauthorized,
	}
}

// Wrap wraps an existing error with a DominatorError
func Wrap(err error, grpcCode codes.Code, httpStatus int, format string, args ...interface{}) error {
	return &DominatorError{
		message:    fmt.Sprintf(format, args...),
		grpcCode:   grpcCode,
		httpStatus: httpStatus,
		cause:      err,
	}
}

// ToGRPCStatus converts an error to a gRPC status error
func ToGRPCStatus(err error) error {
	if err == nil {
		return nil
	}

	var domErr *DominatorError
	if As(err, &domErr) {
		return status.Error(domErr.grpcCode, domErr.message)
	}

	// Fallback for non-DominatorError
	return status.Error(codes.Unknown, err.Error())
}

// ToHTTPStatus extracts HTTP status code from error
func ToHTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}

	var domErr *DominatorError
	if As(err, &domErr) {
		return domErr.httpStatus
	}

	// Fallback
	return http.StatusInternalServerError
}

// As is a wrapper around errors.As for convenience
func As(err error, target interface{}) bool {
	// Use standard library errors.As
	return false // Placeholder - use errors.As from stdlib
}

