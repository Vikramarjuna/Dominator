package grpc

import (
	"net/http"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

type RestAuthConfig struct {
	ServiceName  string
	PathToMethod func(path, httpMethod string) string
}

// NewRestAuthMiddleware creates HTTP middleware for REST API authentication.
func NewRestAuthMiddleware(config RestAuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			method := config.PathToMethod(r.URL.Path, r.Method)
			fullMethod := "/" + config.ServiceName + "/" + method
			startTime := time.Now()

			if method == "Unknown" {
				http.Error(w, "endpoint not found: "+r.URL.Path, http.StatusNotFound)
				return
			}

			if _, ok := unauthenticatedMethods[fullMethod]; ok {
				ctx := ContextWithAuthInfo(r.Context(), &AuthInfo{})
				recordRestCallStart()
				next.ServeHTTP(w, r.WithContext(ctx))
				recordRestCallEnd(fullMethod, startTime, nil)
				return
			}

			if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
				recordRestDeniedCall(fullMethod)
				http.Error(w, "client certificate required", http.StatusUnauthorized)
				return
			}
			_, isPublic := publicMethods[fullMethod]
			authInformation, authorised, err := srpc.CheckTlsAuthorisation(
				grpcToSrpcMethod(fullMethod), *r.TLS, true, isPublic)
			if err != nil {
				recordRestDeniedCall(fullMethod)
				http.Error(w, "authentication failed: "+err.Error(), http.StatusUnauthorized)
				return
			}
			if !authorised {
				recordRestDeniedCall(fullMethod)
				http.Error(w, "permission denied: "+fullMethod, http.StatusForbidden)
				return
			}
			ctx := ContextWithAuthInfo(r.Context(),
				&AuthInfo{authInformation: authInformation})
			recordRestCallStart()
			next.ServeHTTP(w, r.WithContext(ctx))
			recordRestCallEnd(fullMethod, startTime, nil)
		})
	}
}
