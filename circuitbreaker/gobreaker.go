package circuitbreaker

import (
	"context"

	"github.com/sony/gobreaker"

	"github.com/a69/kit.go/endpoint"
)

// Gobreaker returns an endpoint.Middleware that implements the circuit
// breaker pattern using the sony/gobreaker package. Only errors returned by
// the wrapped endpoint count against the circuit breaker's error count.
//
// See http://godoc.org/github.com/sony/gobreaker for more information.
func Gobreaker[REQ any, RES any](cb *gobreaker.CircuitBreaker) endpoint.Middleware[REQ, RES] {
	return func(next endpoint.Endpoint[REQ, RES]) endpoint.Endpoint[REQ, RES] {
		return func(ctx context.Context, request REQ) (res RES, err error) {
			resp, err := cb.Execute(func() (interface{}, error) { return next(ctx, request) })
			if err != nil {
				return
			}
			return resp.(RES), err
		}
	}
}
