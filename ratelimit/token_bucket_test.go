package ratelimit_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"golang.org/x/time/rate"

	"github.com/a69/kit.go/endpoint"
	"github.com/a69/kit.go/ratelimit"
)

var nopEndpoint endpoint.Endpoint[struct{}, struct{}] = func(context.Context, struct{}) (struct{}, error) { return struct{}{}, nil }

func TestXRateErroring(t *testing.T) {
	limit := rate.NewLimiter(rate.Every(time.Minute), 1)
	testSuccessThenFailure(
		t,
		ratelimit.NewErroringLimiter[struct{}, struct{}](limit)(nopEndpoint),
		ratelimit.ErrLimited.Error())

}

func TestXRateDelaying(t *testing.T) {
	limit := rate.NewLimiter(rate.Every(time.Minute), 1)
	testSuccessThenFailure(
		t,
		ratelimit.NewDelayingLimiter[struct{}, struct{}](limit)(nopEndpoint),
		"exceed context deadline")
}

func testSuccessThenFailure(t *testing.T, e endpoint.Endpoint[struct{}, struct{}], failContains string) {
	ctx, cxl := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cxl()

	// First request should succeed.
	if _, err := e(ctx, struct{}{}); err != nil {
		t.Errorf("unexpected: %v\n", err)
	}

	// Next request should fail.
	if _, err := e(ctx, struct{}{}); !strings.Contains(err.Error(), failContains) {
		t.Errorf("expected `%s`: %v\n", failContains, err)
	}
}
