package serverutil

import (
	"github.com/Cloud-Foundations/Dominator/lib/ratelimit"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func newRateLimiterBlocker(limiter *ratelimit.Limiter) *RateLimiterBlocker {
	return &RateLimiterBlocker{limiter: limiter}
}

// There is nothing to release: the token buckets are non-blocking.
func (blocker *RateLimiterBlocker) blockMethod(methodName string,
	authInfo *srpc.AuthInformation) (func(), error) {
	return nil, blocker.limiter.Allow(methodName, authInfo.Username,
		ratelimit.ProtocolSRPC, authInfo.HaveMethodAccess)
}
