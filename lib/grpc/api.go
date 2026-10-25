/*
Package grpc provides gRPC server-side helpers for Dominator services:
authentication and authorisation interceptors that reuse SRPC's
authorisation logic, and a helper to convert Go errors into gRPC status
errors.

A gRPC server registers UnaryAuthInterceptor and StreamAuthInterceptor when
constructing the server, and registers per-service public/unauthenticated
method options via RegisterServiceOptions. gRPC method names of the form
"/package.Service/Method" are stripped to the SRPC form "Service.Method"
before being passed to the shared authorisation check, so a single set of
permitted-method entries in the caller's X509 client certificate governs
access over both protocols. Method names in the gRPC service definition
must therefore match the SRPC method names exactly.
*/
package grpc

import (
	"context"
	"io"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	liberrors "github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

// CodedError is implemented by errors that provide a gRPC status code.
type CodedError interface {
	GrpcCode() codes.Code
}

// ErrorToStatus converts err into a gRPC status error. Errors implementing
// CodedError use their declared code; otherwise the message is matched against
// a table of prefixes and substrings, falling back to codes.Internal.
func ErrorToStatus(err error) error {
	return errorToStatus(err)
}

// DoNotUseMethodPowersMetadataKey is the metadata key (value "true") to opt
// out of method powers, like SRPC's doNotUseMethodPowers query parameter.
const DoNotUseMethodPowersMetadataKey = "donotusemethodpowers"

// AuthInfo holds the caller's authentication information.
type AuthInfo struct {
	authInformation *srpc.AuthInformation
}

// StreamingConn adapts a gRPC stream to the methods of srpc.Conn that a
// streaming method implementation uses (Decode, Encode, Flush,
// GetAuthInformation, Read and Username), so the same implementation can serve
// both SRPC and gRPC calls.
type StreamingConn struct {
	authInformation *srpc.AuthInformation
	decode          func(v interface{}) error
	encode          func(v interface{}) error
	reader          io.Reader
}

// ServiceOptions configures authorisation for a gRPC service.
type ServiceOptions struct {
	PublicMethods          []string // Method names.
	UnauthenticatedMethods []string // Method names.
}

// AuthInfoFromContext returns the AuthInfo attached to ctx, or nil.
func AuthInfoFromContext(ctx context.Context) *AuthInfo {
	return authInfoFromContext(ctx)
}

// ContextWithAuthInfo returns a copy of ctx carrying authInfo.
func ContextWithAuthInfo(ctx context.Context,
	authInfo *AuthInfo) context.Context {
	return contextWithAuthInfo(ctx, authInfo)
}

// GetAuthInformation returns the authentication information or nil.
func (a *AuthInfo) GetAuthInformation() *srpc.AuthInformation {
	return a.getAuthInformation()
}

// RegisterServiceOptions registers public and unauthenticated methods for a
// service. Method names must match the SRPC method names exactly.
func RegisterServiceOptions(serviceName string, options ServiceOptions) {
	registerServiceOptions(serviceName, options)
}

// StreamAuthInterceptor authenticates and authorises streaming RPCs.
func StreamAuthInterceptor(srv interface{}, ss grpc.ServerStream,
	info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	return streamAuthInterceptor(srv, ss, info, handler)
}

// UnaryAuthInterceptor authenticates and authorises unary RPCs.
func UnaryAuthInterceptor(ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return unaryAuthInterceptor(ctx, req, info, handler)
}

// NewChunkReader returns an io.Reader over the chunks returned by next. It
// returns io.EOF once next does. Empty chunks are skipped and a chunk larger
// than the caller's buffer is delivered over several Read calls.
func NewChunkReader(next func() ([]byte, error)) io.Reader {
	return &chunkReader{next: next}
}

// NewStreamingConn returns a StreamingConn for the call whose context is ctx.
// The caller's authentication information is taken from ctx, as set by the
// auth interceptors. decode fills in the request, and encode sends a reply;
// both convert between the SRPC and gRPC message types. reader supplies the
// raw bytes returned by Read and may be nil if the method does not read them.
func NewStreamingConn(ctx context.Context, decode func(v interface{}) error,
	encode func(v interface{}) error, reader io.Reader) *StreamingConn {
	return newStreamingConn(ctx, decode, encode, reader)
}

// Decode calls the decode function given to NewStreamingConn.
func (c *StreamingConn) Decode(v interface{}) error {
	return c.decode(v)
}

// Encode calls the encode function given to NewStreamingConn.
func (c *StreamingConn) Encode(v interface{}) error {
	return c.encode(v)
}

// Flush does nothing, since gRPC sends each message as soon as it is encoded.
func (c *StreamingConn) Flush() error {
	return nil
}

// GetAuthInformation returns the authentication information or nil.
func (c *StreamingConn) GetAuthInformation() *srpc.AuthInformation {
	return c.authInformation
}

// Read reads from the reader given to NewStreamingConn. It returns an
// UnimplementedError if there was none.
func (c *StreamingConn) Read(p []byte) (int, error) {
	if c.reader == nil {
		return 0, liberrors.NewUnimplementedError("Read")
	}
	return c.reader.Read(p)
}

// Username returns the caller's username, or the empty string if the caller
// is unauthenticated.
func (c *StreamingConn) Username() string {
	if c.authInformation == nil {
		return ""
	}
	return c.authInformation.Username
}
