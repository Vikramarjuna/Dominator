/*
Package grpc provides common gRPC infrastructure for Dominator services.

It mirrors the lib/srpc package, providing authentication and connection
handling for gRPC servers using the same TLS certificates and auth model
as SRPC. This enables Phase 3 embedded gRPC support where both SRPC and
gRPC protocols can coexist in the same daemon.

Key components:
  - Conn: Mirrors srpc.Conn, providing GetAuthInformation() for handlers
  - UnaryAuthInterceptor: Extracts auth from TLS certs into context
  - StreamAuthInterceptor: Same for streaming RPCs
  - GetConn: Retrieves the Conn from context in handlers

Example usage in a gRPC handler:

	func (s *server) ChangeMachineTags(ctx context.Context,
	    req *pb.ChangeMachineTagsRequest) (*pb.ChangeMachineTagsResponse, error) {

	    conn := libgrpc.GetConn(ctx)
	    authInfo := conn.GetAuthInformation()

	    err := s.manager.ChangeMachineTags(req.Hostname, authInfo, tags)
	    // ...
	}
*/
package grpc

import (
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

// Conn represents a gRPC connection with authentication information.
// It mirrors srpc.Conn to provide the same API for handlers, allowing
// code reuse between SRPC and gRPC implementations.
type Conn struct {
	authInfo         *srpc.AuthInformation
	permittedMethods map[string]struct{}
}

// GetAuthInformation returns authentication information for the client.
// This mirrors srpc.Conn.GetAuthInformation() to allow the same handler
// patterns for both SRPC and gRPC.
func (c *Conn) GetAuthInformation() *srpc.AuthInformation {
	if c == nil {
		return nil
	}
	return c.authInfo
}

// GetPermittedMethods returns the methods permitted by the client certificate.
// Returns nil if all methods are permitted, empty map if none are permitted.
func (c *Conn) GetPermittedMethods() map[string]struct{} {
	if c == nil {
		return nil
	}
	return c.permittedMethods
}

