package serverutil

import (
	"sync"

	"github.com/Cloud-Foundations/Dominator/lib/ratelimit"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

type PerUserMethodLimiter struct {
	mutex               sync.Mutex
	perUserMethodCounts map[userMethodType]uint
	perUserMethodLimits map[string]uint
}

type RateLimiterBlocker struct {
	limiter *ratelimit.Limiter
}

type userMethodType struct {
	method   string
	username string
}

func NewPerUserMethodLimiter(
	perUserMethodLimits map[string]uint) *PerUserMethodLimiter {
	return newPerUserMethodLimiter(perUserMethodLimits)
}

func NewRateLimiterBlocker(limiter *ratelimit.Limiter) *RateLimiterBlocker {
	return newRateLimiterBlocker(limiter)
}

func (limiter *PerUserMethodLimiter) BlockMethod(methodName string,
	authInfo *srpc.AuthInformation) (func(), error) {
	return limiter.blockMethod(methodName, authInfo)
}

func (blocker *RateLimiterBlocker) BlockMethod(methodName string,
	authInfo *srpc.AuthInformation) (func(), error) {
	return blocker.blockMethod(methodName, authInfo)
}
