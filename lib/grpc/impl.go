package grpc

import (
	"context"
	"crypto/tls"
	"io"
	"maps"
	"net"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/lib/tags"
)

type authInfoKeyType struct{}

var authInfoKey = authInfoKeyType{}

var publicMethods = make(map[string]struct{})
var unauthenticatedMethods = make(map[string]struct{})

// chunkReader implements io.Reader over a sequence of chunks.
type chunkReader struct {
	buffer []byte // Unread part of the current chunk.
	next   func() ([]byte, error)
}

// wrappedStream overrides Context to include auth info.
type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context {
	return w.ctx
}

func authInfoFromContext(ctx context.Context) *AuthInfo {
	if v := ctx.Value(authInfoKey); v != nil {
		return v.(*AuthInfo)
	}
	return nil
}

func contextWithAuthInfo(ctx context.Context,
	authInfo *AuthInfo) context.Context {
	return context.WithValue(ctx, authInfoKey, authInfo)
}

func (a *AuthInfo) getAuthInformation() *srpc.AuthInformation {
	if a == nil {
		return nil
	}
	return a.authInformation
}

func registerServiceOptions(serviceName string, options ServiceOptions) {
	allMethods := make(map[string]struct{})
	for _, method := range options.PublicMethods {
		fullMethod := "/" + serviceName + "/" + method
		publicMethods[fullMethod] = struct{}{}
		allMethods[fullMethod] = struct{}{}
	}
	for _, method := range options.UnauthenticatedMethods {
		fullMethod := "/" + serviceName + "/" + method
		unauthenticatedMethods[fullMethod] = struct{}{}
		allMethods[fullMethod] = struct{}{}
	}
	registerMethodMetrics(serviceName, allMethods)
}

func streamAuthInterceptor(srv interface{}, ss grpc.ServerStream,
	info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	ctx, err := authoriseRequest(ss.Context(), info.FullMethod)
	if err != nil {
		return err
	}
	wrapped := &wrappedStream{ServerStream: ss, ctx: ctx}
	recordCallStart()
	startTime := time.Now()
	defer func() {
		if r := recover(); r != nil {
			recordPanic()
			panic(r)
		}
	}()
	err = handler(srv, wrapped)
	recordCallEnd(info.FullMethod, startTime, err)
	return err
}

func unaryAuthInterceptor(ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	ctx, err := authoriseRequest(ctx, info.FullMethod)
	if err != nil {
		return nil, err
	}
	recordCallStart()
	startTime := time.Now()
	defer func() {
		if r := recover(); r != nil {
			recordPanic()
			panic(r)
		}
	}()
	resp, err := handler(ctx, req)
	recordCallEnd(info.FullMethod, startTime, err)
	return resp, err
}

func authoriseRequest(ctx context.Context,
	fullMethod string) (context.Context, error) {
	_, isPublic := publicMethods[fullMethod]
	_, isUnauthenticated := unauthenticatedMethods[fullMethod]
	if isUnauthenticated {
		return contextWithAuthInfo(ctx, &AuthInfo{}), nil
	}
	tlsState, err := tlsStateFromContext(ctx)
	if err != nil {
		return nil, err
	}
	authInformation, authorised, err := srpc.CheckTlsAuthorisation(
		grpcToSrpcMethod(fullMethod), tlsState,
		!doNotUseMethodPowersFromMetadata(ctx), isPublic)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	if !authorised {
		recordDeniedCall(fullMethod)
		return nil, status.Error(codes.PermissionDenied, "call on "+fullMethod)
	}
	return contextWithAuthInfo(ctx,
		&AuthInfo{authInformation: authInformation}), nil
}

func tlsStateFromContext(ctx context.Context) (tls.ConnectionState, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return tls.ConnectionState{},
			status.Error(codes.Unauthenticated, "no peer info in context")
	}
	if p.AuthInfo == nil {
		return tls.ConnectionState{},
			status.Error(codes.Unauthenticated, "no TLS auth info")
	}
	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return tls.ConnectionState{},
			status.Error(codes.Unauthenticated, "unexpected auth info type")
	}
	return tlsInfo.State, nil
}

// doNotUseMethodPowersFromMetadata returns true if the incoming request
// carries metadata opting out of method powers.
func doNotUseMethodPowersFromMetadata(ctx context.Context) bool {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return false
	}
	for _, v := range md.Get(DoNotUseMethodPowersMetadataKey) {
		if v == "true" {
			return true
		}
	}
	return false
}

// grpcToSrpcMethod converts "/package.Service/Method" to "Service.Method".
func grpcToSrpcMethod(fullMethod string) string {
	parts := strings.Split(strings.TrimPrefix(fullMethod, "/"), "/")
	if len(parts) != 2 {
		return fullMethod
	}
	serviceParts := strings.Split(parts[0], ".")
	serviceName := serviceParts[len(serviceParts)-1]
	return serviceName + "." + parts[1]
}

// Read implements io.Reader. It returns the number of bytes copied into p, or
// the error returned by next, which is io.EOF at the end of the data.
func (r *chunkReader) Read(p []byte) (int, error) {
	if len(p) < 1 {
		return 0, nil
	}
	for len(r.buffer) < 1 {
		chunk, err := r.next()
		if err != nil {
			return 0, err
		}
		r.buffer = chunk
	}
	n := copy(p, r.buffer)
	r.buffer = r.buffer[n:]
	return n, nil
}

// newStreamingConn returns a StreamingConn whose caller is the one recorded in
// ctx, or an unauthenticated one if ctx carries no AuthInfo.
func newStreamingConn(ctx context.Context, decode func(v interface{}) error,
	encode func(v interface{}) error, reader io.Reader) *StreamingConn {
	return &StreamingConn{
		authInformation: authInfoFromContext(ctx).getAuthInformation(),
		decode:          decode,
		encode:          encode,
		reader:          reader,
	}
}

// peek returns nil and an UnimplementedError, since the stream cannot be
// peeked at.
func (c *StreamingConn) peek(n int) ([]byte, error) {
	return nil, errors.NewUnimplementedError("Peek")
}

// read returns the number of bytes read into p from the reader, or an
// UnimplementedError if there is none.
func (c *StreamingConn) read(p []byte) (int, error) {
	if c.reader == nil {
		return 0, errors.NewUnimplementedError("Read")
	}
	return c.reader.Read(p)
}

// username returns the caller's username, or the empty string if the caller is
// unauthenticated.
func (c *StreamingConn) username() string {
	if c.authInformation == nil {
		return ""
	}
	return c.authInformation.Username
}

// ipToString returns the text form of ip, or "" if ip is empty.
func ipToString(ip net.IP) string {
	if len(ip) < 1 {
		return ""
	}
	return ip.String()
}

// mapToTags returns a copy of m, or nil if m is nil.
func mapToTags(m map[string]string) tags.Tags {
	return maps.Clone(tags.Tags(m))
}

// stringToIP returns the address parsed from s, or nil if s is empty. The
// error is an *errors.InvalidArgumentError if s is not an IP address.
func stringToIP(s string) (net.IP, error) {
	if s == "" {
		return nil, nil
	}
	ip := net.ParseIP(s)
	if ip == nil {
		return nil, errors.NewInvalidArgumentError("ip_address",
			"malformed IP address: "+s)
	}
	return ip, nil
}

// stringToMAC returns the address parsed from s, or nil if s is empty. The
// error is an *errors.InvalidArgumentError if s is not a MAC address.
func stringToMAC(s string) (net.HardwareAddr, error) {
	if s == "" {
		return nil, nil
	}
	mac, err := net.ParseMAC(s)
	if err != nil {
		return nil, errors.NewInvalidArgumentError("mac_address", err.Error())
	}
	return mac, nil
}

// tagsToMap returns a copy of t, or nil if t is nil.
func tagsToMap(t tags.Tags) map[string]string {
	return maps.Clone(map[string]string(t))
}
