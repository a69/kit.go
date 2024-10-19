package awslambda

import (
	"context"

	"github.com/a69/kit.go/endpoint"
	"github.com/a69/kit.go/transport"
	"github.com/go-kit/log"
)

// Handler wraps an endpoint.
type Handler[REQ any, RES any] struct {
	e            endpoint.Endpoint[REQ, RES]
	dec          DecodeRequestFunc[REQ]
	enc          EncodeResponseFunc[RES]
	before       []HandlerRequestFunc
	after        []HandlerResponseFunc[RES]
	errorEncoder ErrorEncoder
	finalizer    []HandlerFinalizerFunc
	errorHandler transport.ErrorHandler
}

// NewHandler constructs a new handler, which implements
// the AWS lambda.Handler interface.
func NewHandler[REQ any, RES any](
	e endpoint.Endpoint[REQ, RES],
	dec DecodeRequestFunc[REQ],
	enc EncodeResponseFunc[RES],
	options ...HandlerOption[REQ, RES],
) *Handler[REQ, RES] {
	h := &Handler[REQ, RES]{
		e:            e,
		dec:          dec,
		enc:          enc,
		errorEncoder: DefaultErrorEncoder,
		errorHandler: transport.NewLogErrorHandler(log.NewNopLogger()),
	}
	for _, option := range options {
		option(h)
	}
	return h
}

// HandlerOption sets an optional parameter for handlers.
type HandlerOption[REQ any, RES any] func(*Handler[REQ, RES])

// HandlerBefore functions are executed on the payload byte,
// before the request is decoded.
func HandlerBefore[REQ any, RES any](before ...HandlerRequestFunc) HandlerOption[REQ, RES] {
	return func(h *Handler[REQ, RES]) { h.before = append(h.before, before...) }
}

// HandlerAfter functions are only executed after invoking the endpoint
// but prior to returning a response.
func HandlerAfter[REQ any, RES any](after ...HandlerResponseFunc[RES]) HandlerOption[REQ, RES] {
	return func(h *Handler[REQ, RES]) { h.after = append(h.after, after...) }
}

// HandlerErrorLogger is used to log non-terminal errors.
// By default, no errors are logged.
// Deprecated: Use HandlerErrorHandler instead.
func HandlerErrorLogger[REQ any, RES any](logger log.Logger) HandlerOption[REQ, RES] {
	return func(h *Handler[REQ, RES]) { h.errorHandler = transport.NewLogErrorHandler(logger) }
}

// HandlerErrorHandler is used to handle non-terminal errors.
// By default, non-terminal errors are ignored.
func HandlerErrorHandler[REQ any, RES any](errorHandler transport.ErrorHandler) HandlerOption[REQ, RES] {
	return func(h *Handler[REQ, RES]) { h.errorHandler = errorHandler }
}

// HandlerErrorEncoder is used to encode errors.
func HandlerErrorEncoder[REQ any, RES any](ee ErrorEncoder) HandlerOption[REQ, RES] {
	return func(h *Handler[REQ, RES]) { h.errorEncoder = ee }
}

// HandlerFinalizer sets finalizer which are called at the end of
// request. By default no finalizer is registered.
func HandlerFinalizer[REQ any, RES any](f ...HandlerFinalizerFunc) HandlerOption[REQ, RES] {
	return func(h *Handler[REQ, RES]) { h.finalizer = append(h.finalizer, f...) }
}

// DefaultErrorEncoder defines the default behavior of encoding an error response,
// where it returns nil, and the error itself.
func DefaultErrorEncoder(ctx context.Context, err error) ([]byte, error) {
	return nil, err
}

// Invoke represents implementation of the AWS lambda.Handler interface.
func (h *Handler[REQ, RES]) Invoke(
	ctx context.Context,
	payload []byte,
) (resp []byte, err error) {
	if len(h.finalizer) > 0 {
		defer func() {
			for _, f := range h.finalizer {
				f(ctx, resp, err)
			}
		}()
	}

	for _, f := range h.before {
		ctx = f(ctx, payload)
	}

	request, err := h.dec(ctx, payload)
	if err != nil {
		h.errorHandler.Handle(ctx, err)
		return h.errorEncoder(ctx, err)
	}

	response, err := h.e(ctx, request)
	if err != nil {
		h.errorHandler.Handle(ctx, err)
		return h.errorEncoder(ctx, err)
	}

	for _, f := range h.after {
		ctx = f(ctx, response)
	}

	if resp, err = h.enc(ctx, response); err != nil {
		h.errorHandler.Handle(ctx, err)
		return h.errorEncoder(ctx, err)
	}

	return resp, err
}
