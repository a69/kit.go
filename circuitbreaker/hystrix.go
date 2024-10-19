package circuitbreaker

import (
	"context"

	"github.com/afex/hystrix-go/hystrix"

	"github.com/a69/kit.go/endpoint"
)

// Hystrix returns an endpoint.Middleware that implements the circuit
// breaker pattern using the afex/hystrix-go package.
//
// When using this circuit breaker, please configure your commands separately.
//
// See https://godoc.org/github.com/afex/hystrix-go/hystrix for more
// information.
func Hystrix[REQ any, RES any](commandName string) endpoint.Middleware[REQ, RES] {
	return func(next endpoint.Endpoint[REQ, RES]) endpoint.Endpoint[REQ, RES] {
		return func(ctx context.Context, request REQ) (response RES, err error) {
			err = hystrix.Do(commandName, func() (err error) {
				response, err = next(ctx, request)
				return err
			}, nil)
			return
		}
	}
}
