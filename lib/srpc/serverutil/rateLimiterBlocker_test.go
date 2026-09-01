package serverutil

import (
	"testing"

	"google.golang.org/grpc/codes"

	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/ratelimit"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

var _ srpc.MethodBlocker = (*RateLimiterBlocker)(nil)

func newTestBlocker(t *testing.T,
	limits ratelimit.Limits) *RateLimiterBlocker {
	t.Helper()
	limiter, err := ratelimit.New(limits, ratelimit.Options{})
	if err != nil {
		t.Fatalf("ratelimit.New: %s", err)
	}
	return NewRateLimiterBlocker(limiter)
}

func perUserLimits() ratelimit.Limits {
	return ratelimit.Limits{
		PerUserPerMethod: ratelimit.PerUserPerMethodLimits{
			Default: ratelimit.MethodLimit{
				RequestsPerSecond: 0.001, Burst: 1},
		},
	}
}

func TestRateLimiterBlocker_AdmitAndDeny(t *testing.T) {
	blocker := newTestBlocker(t, perUserLimits())
	authInfo := &srpc.AuthInformation{Username: "alice"}
	rn, err := blocker.BlockMethod("Foo", authInfo)
	if err != nil {
		t.Fatalf("first BlockMethod: %s", err)
	}
	if rn != nil {
		t.Fatal("expected nil release notifier on admit")
	}
	rn, err = blocker.BlockMethod("Foo", authInfo)
	if err == nil {
		t.Fatal("expected error on second BlockMethod")
	}
	if rn != nil {
		t.Fatal("expected nil release notifier on denial")
	}
	// The typed error must survive the adapter: gRPC and REST map denials to
	// codes.ResourceExhausted through it.
	reErr, ok := err.(*errors.ResourceExhaustedError)
	if !ok {
		t.Fatalf("expected *ResourceExhaustedError, got %T: %v", err, err)
	}
	if reErr.Resource != "Foo" {
		t.Errorf("got Resource=%q; want %q", reErr.Resource, "Foo")
	}
	want := ratelimit.LimitTypePerUserPerMethod.String()
	if reErr.Reason != want {
		t.Errorf("got Reason=%q; want %q", reErr.Reason, want)
	}
	if got := reErr.GrpcCode(); got != codes.ResourceExhausted {
		t.Errorf("got GrpcCode=%v; want %v", got, codes.ResourceExhausted)
	}
}

func TestRateLimiterBlocker_HaveMethodAccessBypassesPerUser(t *testing.T) {
	blocker := newTestBlocker(t, perUserLimits())
	powerInfo := &srpc.AuthInformation{
		Username: "alice", HaveMethodAccess: true}
	for i := 0; i < 5; i++ {
		if _, err := blocker.BlockMethod("Foo", powerInfo); err != nil {
			t.Fatalf("power user iteration %d: %s", i, err)
		}
	}
}

func TestRateLimiterBlocker_GlobalAppliesToPowerUser(t *testing.T) {
	limiter, err := ratelimit.New(ratelimit.Limits{
		Global: ratelimit.MethodLimit{RequestsPerSecond: 0.001, Burst: 1},
	}, ratelimit.Options{})
	if err != nil {
		t.Fatalf("ratelimit.New: %s", err)
	}
	blocker := NewRateLimiterBlocker(limiter)
	powerInfo := &srpc.AuthInformation{
		Username: "admin", HaveMethodAccess: true}
	if _, err := blocker.BlockMethod("Foo", powerInfo); err != nil {
		t.Fatalf("first power-user call: %s", err)
	}
	if _, err := blocker.BlockMethod("Foo", powerInfo); err == nil {
		t.Fatal("global limit must apply to power user")
	}
	if got := limiter.DeniedCount("Foo", ratelimit.LimitTypeGlobal,
		ratelimit.ProtocolSRPC); got != 1 {
		t.Fatalf("DeniedCount=%d; want 1", got)
	}
}
