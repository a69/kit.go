package endpoint

import (
	"context"
)

// Endpoint is the fundamental building block of servers and clients.
// It represents a single RPC method.
type Endpoint[REQ any, RES any] func(ctx context.Context, request REQ) (response RES, err error)

// Nop is an endpoint that does nothing and returns a nil error.
// Useful for tests.
func Nop[REQ any, RES any](context.Context, REQ) (_ RES, _ error) { return }

// Middleware is a chainable behavior modifier for endpoints.
type Middleware[REQ any, RES any] func(Endpoint[REQ, RES]) Endpoint[REQ, RES]

// Chain is a helper function for composing middlewares. Requests will
// traverse them in the order they're declared. That is, the first middleware
// is treated as the outermost middleware.
func Chain[REQ any, RES any](outer Middleware[REQ, RES], others ...Middleware[REQ, RES]) Middleware[REQ, RES] {
	return func(next Endpoint[REQ, RES]) Endpoint[REQ, RES] {
		for i := len(others) - 1; i >= 0; i-- { // reverse
			next = others[i](next)
		}
		return outer(next)
	}
}

// Failer may be implemented by Go kit response types that contain business
// logic error details. If Failed returns a non-nil error, the Go kit transport
// layer may interpret this as a business logic error, and may encode it
// differently than a regular, successful response.
//
// It's not necessary for your response types to implement Failer, but it may
// help for more sophisticated use cases. The addsvc example shows how Failer
// should be used by a complete application.
type Failer interface {
	Failed() error
}
