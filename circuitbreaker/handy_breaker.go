package circuitbreaker

import (
	"context"
	"time"

	"github.com/streadway/handy/breaker"

	"github.com/a69/kit.go/endpoint"
)

// HandyBreaker returns an endpoint.Middleware that implements the circuit
// breaker pattern using the streadway/handy/breaker package. Only errors
// returned by the wrapped endpoint count against the circuit breaker's error
// count.
//
// See http://godoc.org/github.com/streadway/handy/breaker for more
// information.
func HandyBreaker[REQ any, RES any](cb breaker.Breaker) endpoint.Middleware[REQ, RES] {
	return func(next endpoint.Endpoint[REQ, RES]) endpoint.Endpoint[REQ, RES] {
		return func(ctx context.Context, request REQ) (response RES, err error) {
			if !cb.Allow() {
				err = breaker.ErrCircuitOpen
				return
			}

			defer func(begin time.Time) {
				if err == nil {
					cb.Success(time.Since(begin))
				} else {
					cb.Failure(time.Since(begin))
				}
			}(time.Now())

			response, err = next(ctx, request)
			return
		}
	}
}
