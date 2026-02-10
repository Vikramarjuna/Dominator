package grpc

import (
	"errors"
	"testing"

	liberrors "github.com/Cloud-Foundations/Dominator/lib/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestErrorToStatus_TypedErrors(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode codes.Code
	}{
		{
			name:         "NotFoundError",
			err:          liberrors.NewNotFoundError("machine", "host1"),
			expectedCode: codes.NotFound,
		},
		{
			name:         "PermissionDeniedError",
			err:          liberrors.NewPermissionDeniedError("machine", "update", "not owner"),
			expectedCode: codes.PermissionDenied,
		},
		{
			name:         "UnauthenticatedError",
			err:          liberrors.NewUnauthenticatedError("no credentials"),
			expectedCode: codes.Unauthenticated,
		},
		{
			name:         "AlreadyExistsError",
			err:          liberrors.NewAlreadyExistsError("machine", "host1"),
			expectedCode: codes.AlreadyExists,
		},
		{
			name:         "InvalidArgumentError",
			err:          liberrors.NewInvalidArgumentError("hostname", "empty"),
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "UnavailableError",
			err:          liberrors.NewUnavailableError("topology", "loading"),
			expectedCode: codes.Unavailable,
		},
		{
			name:         "DeadlineExceededError",
			err:          liberrors.NewDeadlineExceededError("GetMachineInfo", "30s"),
			expectedCode: codes.DeadlineExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grpcErr := ErrorToStatus(tt.err)
			st, ok := status.FromError(grpcErr)
			if !ok {
				t.Fatalf("expected gRPC status error, got %T", grpcErr)
			}
			if st.Code() != tt.expectedCode {
				t.Errorf("expected code %v, got %v", tt.expectedCode, st.Code())
			}
		})
	}
}

func TestErrorToStatus_PatternMatching(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode codes.Code
	}{
		{
			name:         "not found pattern",
			err:          errors.New("machine not found"),
			expectedCode: codes.NotFound,
		},
		{
			name:         "does not exist pattern",
			err:          errors.New("hypervisor does not exist"),
			expectedCode: codes.NotFound,
		},
		{
			name:         "permission denied pattern",
			err:          errors.New("permission denied"),
			expectedCode: codes.PermissionDenied,
		},
		{
			name:         "unauthenticated pattern",
			err:          errors.New("not authenticated"),
			expectedCode: codes.Unauthenticated,
		},
		{
			name:         "already exists pattern",
			err:          errors.New("machine already exists"),
			expectedCode: codes.AlreadyExists,
		},
		{
			name:         "invalid pattern",
			err:          errors.New("invalid hostname"),
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "unavailable pattern",
			err:          errors.New("service unavailable"),
			expectedCode: codes.Unavailable,
		},
		{
			name:         "timeout pattern",
			err:          errors.New("operation timed out"),
			expectedCode: codes.DeadlineExceeded,
		},
		{
			name:         "unknown error defaults to Internal",
			err:          errors.New("something went wrong"),
			expectedCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grpcErr := ErrorToStatus(tt.err)
			st, ok := status.FromError(grpcErr)
			if !ok {
				t.Fatalf("expected gRPC status error, got %T", grpcErr)
			}
			if st.Code() != tt.expectedCode {
				t.Errorf("expected code %v, got %v", tt.expectedCode, st.Code())
			}
		})
	}
}

func TestErrorToStatus_NilError(t *testing.T) {
	grpcErr := ErrorToStatus(nil)
	if grpcErr != nil {
		t.Errorf("expected nil, got %v", grpcErr)
	}
}

func TestErrorToStatus_WrappedTypedError(t *testing.T) {
	// Test that wrapped typed errors are still detected
	baseErr := liberrors.NewNotFoundError("machine", "host1")
	wrappedErr := errors.Join(errors.New("context"), baseErr)

	grpcErr := ErrorToStatus(wrappedErr)
	st, ok := status.FromError(grpcErr)
	if !ok {
		t.Fatalf("expected gRPC status error, got %T", grpcErr)
	}
	if st.Code() != codes.NotFound {
		t.Errorf("expected NotFound code, got %v", st.Code())
	}
}

