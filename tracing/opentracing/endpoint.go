package opentracing

import (
	"context"
	"strconv"

	"github.com/opentracing/opentracing-go"
	otext "github.com/opentracing/opentracing-go/ext"
	otlog "github.com/opentracing/opentracing-go/log"

	"github.com/a69/kit.go/endpoint"
	"github.com/a69/kit.go/sd/lb"
)

// TraceEndpoint returns a Middleware that wraps the `next` Endpoint in an
// OpenTracing Span called `operationName`.
//
// If `ctx` already has a Span, child span is created from it.
// If `ctx` doesn't yet have a Span, the new one is created.
func TraceEndpoint[REQ any, RES any](tracer opentracing.Tracer, operationName string, opts ...EndpointOption) endpoint.Middleware[REQ, RES] {
	cfg := &EndpointOptions{
		Tags: make(opentracing.Tags),
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return func(next endpoint.Endpoint[REQ, RES]) endpoint.Endpoint[REQ, RES] {
		return func(ctx context.Context, request REQ) (response RES, err error) {
			if cfg.GetOperationName != nil {
				if newOperationName := cfg.GetOperationName(ctx, operationName); newOperationName != "" {
					operationName = newOperationName
				}
			}

			var span opentracing.Span
			if parentSpan := opentracing.SpanFromContext(ctx); parentSpan != nil {
				span = tracer.StartSpan(
					operationName,
					opentracing.ChildOf(parentSpan.Context()),
				)
			} else {
				span = tracer.StartSpan(operationName)
			}
			defer span.Finish()

			applyTags(span, cfg.Tags)
			if cfg.GetTags != nil {
				extraTags := cfg.GetTags(ctx)
				applyTags(span, extraTags)
			}

			ctx = opentracing.ContextWithSpan(ctx, span)

			defer func() {
				if err != nil {
					if lbErr, ok := err.(lb.RetryError); ok {
						// handle errors originating from lb.Retry
						fields := make([]otlog.Field, 0, len(lbErr.RawErrors))
						for idx, rawErr := range lbErr.RawErrors {
							fields = append(fields, otlog.String(
								"gokit.retry.error."+strconv.Itoa(idx+1), rawErr.Error(),
							))
						}

						otext.LogError(span, lbErr, fields...)

						return
					}

					// generic error
					otext.LogError(span, err)

					return
				}
			}()

			return next(ctx, request)
		}
	}
}

// TraceServer returns a Middleware that wraps the `next` Endpoint in an
// OpenTracing Span called `operationName` with server span.kind tag..
func TraceServer[REQ any, RES any](tracer opentracing.Tracer, operationName string, opts ...EndpointOption) endpoint.Middleware[REQ, RES] {
	opts = append(opts, WithTags(map[string]interface{}{
		otext.SpanKindRPCServer.Key: otext.SpanKindRPCServer.Value,
	}))

	return TraceEndpoint[REQ, RES](tracer, operationName, opts...)
}

// TraceClient returns a Middleware that wraps the `next` Endpoint in an
// OpenTracing Span called `operationName` with client span.kind tag.
func TraceClient[REQ any, RES any](tracer opentracing.Tracer, operationName string, opts ...EndpointOption) endpoint.Middleware[REQ, RES] {
	opts = append(opts, WithTags(map[string]interface{}{
		otext.SpanKindRPCClient.Key: otext.SpanKindRPCClient.Value,
	}))

	return TraceEndpoint[REQ, RES](tracer, operationName, opts...)
}

func applyTags(span opentracing.Span, tags opentracing.Tags) {
	for key, value := range tags {
		span.SetTag(key, value)
	}
}
