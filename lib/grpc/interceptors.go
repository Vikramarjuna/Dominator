package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

// connKeyType is an unexported type for the context key to prevent collisions.
type connKeyType struct{}

// connKey is the context key for storing/retrieving Conn.
var connKey = connKeyType{}

// UnaryAuthInterceptor extracts authentication from TLS client certificates
// and stores a Conn in the context for handlers to use.
// This mirrors SRPC's httpHandler auth extraction.
func UnaryAuthInterceptor(ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

	conn, err := extractConn(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	ctx = context.WithValue(ctx, connKey, conn)
	return handler(ctx, req)
}

// StreamAuthInterceptor extracts authentication from TLS client certificates
// for streaming RPCs. It wraps the ServerStream to inject the Conn into context.
func StreamAuthInterceptor(srv interface{}, ss grpc.ServerStream,
	info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {

	conn, err := extractConn(ss.Context())
	if err != nil {
		return status.Error(codes.Unauthenticated, err.Error())
	}
	wrapped := &wrappedStream{
		ServerStream: ss,
		ctx:          context.WithValue(ss.Context(), connKey, conn),
	}
	return handler(srv, wrapped)
}

// GetConn retrieves the Conn from context.
// Returns nil if no Conn is present (e.g., if interceptor wasn't used).
func GetConn(ctx context.Context) *Conn {
	if v := ctx.Value(connKey); v != nil {
		return v.(*Conn)
	}
	return nil
}

// extractConn extracts authentication information from the gRPC peer's TLS state
// and creates a Conn. This reuses SRPC's auth extraction logic.
func extractConn(ctx context.Context) (*Conn, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, errors.New("no peer info in context")
	}
	if p.AuthInfo == nil {
		return nil, errors.New("no auth info in peer")
	}
	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return nil, errors.New("peer auth is not TLS")
	}

	// Reuse SRPC's auth extraction from TLS certificates
	authInfo, err := srpc.GetAuthFromTLS(tlsInfo.State)
	if err != nil {
		return nil, err
	}

	// Also get permitted methods if available
	permittedMethods, _ := srpc.GetPermittedMethodsFromTLS(tlsInfo.State)

	return &Conn{
		authInfo:         authInfo,
		permittedMethods: permittedMethods,
	}, nil
}

// wrappedStream wraps grpc.ServerStream to override Context().
type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context returns the wrapped context containing the Conn.
func (w *wrappedStream) Context() context.Context {
	return w.ctx
}

