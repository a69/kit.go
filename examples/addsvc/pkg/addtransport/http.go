package addtransport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	stdopentracing "github.com/opentracing/opentracing-go"
	stdzipkin "github.com/openzipkin/zipkin-go"
	"github.com/sony/gobreaker"

	"github.com/a69/kit.go/circuitbreaker"
	"github.com/a69/kit.go/endpoint"
	"github.com/a69/kit.go/log"
	"github.com/a69/kit.go/tracing/opentracing"
	"github.com/a69/kit.go/tracing/zipkin"
	httptransport "github.com/a69/kit.go/transport/http"

	"github.com/a69/kit.go/examples/addsvc/pkg/addendpoint"
	"github.com/a69/kit.go/examples/addsvc/pkg/addservice"
)

// NewHTTPHandler returns an HTTP handler that makes a set of endpoints
// available on predefined paths.
func NewHTTPHandler(endpoints addendpoint.Set, otTracer stdopentracing.Tracer, zipkinTracer *stdzipkin.Tracer, logger log.Logger) http.Handler {

	m := http.NewServeMux()
	m.Handle("/sum", httptransport.NewServer[addendpoint.SumRequest, addendpoint.SumResponse](
		endpoints.SumEndpoint,
		decodeHTTPSumRequest,
		encodeHTTPGenericResponse[addendpoint.SumResponse],
		append(makeHTTPServerOptions[addendpoint.SumRequest, addendpoint.SumResponse](zipkinTracer), httptransport.ServerBefore[addendpoint.SumRequest, addendpoint.SumResponse](opentracing.HTTPToContext(otTracer, "Sum", logger)))...,
	))
	m.Handle("/concat", httptransport.NewServer(
		endpoints.ConcatEndpoint,
		decodeHTTPConcatRequest,
		encodeHTTPGenericResponse[addendpoint.ConcatResponse],
		append(makeHTTPServerOptions[addendpoint.ConcatRequest, addendpoint.ConcatResponse](zipkinTracer), httptransport.ServerBefore[addendpoint.ConcatRequest, addendpoint.ConcatResponse](opentracing.HTTPToContext(otTracer, "Concat", logger)))...,
	))
	return m
}

// NewHTTPClient returns an AddService backed by an HTTP server living at the
// remote instance. We expect instance to come from a service discovery system,
// so likely of the form "host:port". We bake-in certain middlewares,
// implementing the client library pattern.
func NewHTTPClient(instance string, otTracer stdopentracing.Tracer, zipkinTracer *stdzipkin.Tracer, logger log.Logger) (addservice.Service, error) {
	// Quickly sanitize the instance string.
	if !strings.HasPrefix(instance, "http") {
		instance = "http://" + instance
	}
	u, err := url.Parse(instance)
	if err != nil {
		return nil, err
	}

	// Each individual endpoint is an http/transport.Client (which implements
	// endpoint.Endpoint) that gets wrapped with various middlewares. If you
	// made your own client library, you'd do this work there, so your server
	// could rely on a consistent set of client behavior.
	var sumEndpoint endpoint.Endpoint[addendpoint.SumRequest, addendpoint.SumResponse]
	{
		sumEndpoint = httptransport.NewClient[addendpoint.SumRequest, addendpoint.SumResponse](
			"POST",
			copyURL(u, "/sum"),
			encodeHTTPGenericRequest[addendpoint.SumRequest],
			decodeHTTPSumResponse,
			append(makeHTTPClientOptions[addendpoint.SumRequest, addendpoint.SumResponse](zipkinTracer), httptransport.ClientBefore[addendpoint.SumRequest, addendpoint.SumResponse](opentracing.ContextToHTTP(otTracer, logger)))...,
		).Endpoint()
		sumEndpoint = opentracing.TraceClient[addendpoint.SumRequest, addendpoint.SumResponse](otTracer, "Sum")(sumEndpoint)
		if zipkinTracer != nil {
			sumEndpoint = zipkin.TraceEndpoint[addendpoint.SumRequest, addendpoint.SumResponse](zipkinTracer, "Sum")(sumEndpoint)
		}
		sumEndpoint = makeLimiter[addendpoint.SumRequest, addendpoint.SumResponse]()(sumEndpoint)
		sumEndpoint = circuitbreaker.Gobreaker[addendpoint.SumRequest, addendpoint.SumResponse](gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "Sum",
			Timeout: 30 * time.Second,
		}))(sumEndpoint)
	}

	// The Concat endpoint is the same thing, with slightly different
	// middlewares to demonstrate how to specialize per-endpoint.
	var concatEndpoint endpoint.Endpoint[addendpoint.ConcatRequest, addendpoint.ConcatResponse]
	{
		concatEndpoint = httptransport.NewClient(
			"POST",
			copyURL(u, "/concat"),
			encodeHTTPGenericRequest[addendpoint.ConcatRequest],
			decodeHTTPConcatResponse,
			append(makeHTTPClientOptions[addendpoint.ConcatRequest, addendpoint.ConcatResponse](zipkinTracer), httptransport.ClientBefore[addendpoint.ConcatRequest, addendpoint.ConcatResponse](opentracing.ContextToHTTP(otTracer, logger)))...,
		).Endpoint()
		concatEndpoint = opentracing.TraceClient[addendpoint.ConcatRequest, addendpoint.ConcatResponse](otTracer, "Concat")(concatEndpoint)
		if zipkinTracer != nil {
			concatEndpoint = zipkin.TraceEndpoint[addendpoint.ConcatRequest, addendpoint.ConcatResponse](zipkinTracer, "Concat")(concatEndpoint)
		}
		concatEndpoint = makeLimiter[addendpoint.ConcatRequest, addendpoint.ConcatResponse]()(concatEndpoint)
		concatEndpoint = circuitbreaker.Gobreaker[addendpoint.ConcatRequest, addendpoint.ConcatResponse](gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "Concat",
			Timeout: 10 * time.Second,
		}))(concatEndpoint)
	}

	// Returning the endpoint.Set as a service.Service relies on the
	// endpoint.Set implementing the Service methods. That's just a simple bit
	// of glue code.
	return addendpoint.Set{
		SumEndpoint:    sumEndpoint,
		ConcatEndpoint: concatEndpoint,
	}, nil
}

func copyURL(base *url.URL, path string) *url.URL {
	next := *base
	next.Path = path
	return &next
}

func errorEncoder(_ context.Context, err error, w http.ResponseWriter) {
	w.WriteHeader(err2code(err))
	json.NewEncoder(w).Encode(errorWrapper{Error: err.Error()})
}

func err2code(err error) int {
	switch err {
	case addservice.ErrTwoZeroes, addservice.ErrMaxSizeExceeded, addservice.ErrIntOverflow:
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

func errorDecoder(r *http.Response) error {
	var w errorWrapper
	if err := json.NewDecoder(r.Body).Decode(&w); err != nil {
		return err
	}
	return errors.New(w.Error)
}

type errorWrapper struct {
	Error string `json:"error"`
}

// decodeHTTPSumRequest is a transport/http.DecodeRequestFunc that decodes a
// JSON-encoded sum request from the HTTP request body. Primarily useful in a
// server.
func decodeHTTPSumRequest(_ context.Context, r *http.Request) (addendpoint.SumRequest, error) {
	var req addendpoint.SumRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	return req, err
}

// decodeHTTPConcatRequest is a transport/http.DecodeRequestFunc that decodes a
// JSON-encoded concat request from the HTTP request body. Primarily useful in a
// server.
func decodeHTTPConcatRequest(_ context.Context, r *http.Request) (addendpoint.ConcatRequest, error) {
	var req addendpoint.ConcatRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	return req, err
}

// decodeHTTPSumResponse is a transport/http.DecodeResponseFunc that decodes a
// JSON-encoded sum response from the HTTP response body. If the response has a
// non-200 status code, we will interpret that as an error and attempt to decode
// the specific error message from the response body. Primarily useful in a
// client.
func decodeHTTPSumResponse(_ context.Context, r *http.Response) (addendpoint.SumResponse, error) {
	if r.StatusCode != http.StatusOK {
		return addendpoint.SumResponse{}, errors.New(r.Status)
	}
	var resp addendpoint.SumResponse
	err := json.NewDecoder(r.Body).Decode(&resp)
	return resp, err
}

// decodeHTTPConcatResponse is a transport/http.DecodeResponseFunc that decodes
// a JSON-encoded concat response from the HTTP response body. If the response
// has a non-200 status code, we will interpret that as an error and attempt to
// decode the specific error message from the response body. Primarily useful in
// a client.
func decodeHTTPConcatResponse(_ context.Context, r *http.Response) (addendpoint.ConcatResponse, error) {
	if r.StatusCode != http.StatusOK {
		return addendpoint.ConcatResponse{}, errors.New(r.Status)
	}
	var resp addendpoint.ConcatResponse
	err := json.NewDecoder(r.Body).Decode(&resp)
	return resp, err
}

// encodeHTTPGenericRequest is a transport/http.EncodeRequestFunc that
// JSON-encodes any request to the request body. Primarily useful in a client.
func encodeHTTPGenericRequest[REQ any](_ context.Context, r *http.Request, request *REQ) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(request); err != nil {
		return err
	}
	r.Body = ioutil.NopCloser(&buf)
	return nil
}

// encodeHTTPGenericResponse is a transport/http.EncodeResponseFunc that encodes
// the response as JSON to the response writer. Primarily useful in a server.
func encodeHTTPGenericResponse[RES any](ctx context.Context, w http.ResponseWriter, response RES) error {

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}
func makeHTTPServerOptions[REQ any, RES any](zipkinTracer *stdzipkin.Tracer) []httptransport.ServerOption[REQ, RES] {
	var options []httptransport.ServerOption[REQ, RES]

	if zipkinTracer != nil {
		// Zipkin GRPC Client Trace can either be instantiated per gRPC method with a
		// provided operation name or a global tracing client can be instantiated
		// without an operation name and fed to each Go kit client as ClientOption.
		// In the latter case, the operation name will be the endpoint's grpc method
		// path.
		//
		// In this example, we demonstrace a global tracing client.
		options = append(options, zipkin.HTTPServerTrace[REQ, RES](zipkinTracer))

	}
	return options
}
func makeHTTPClientOptions[REQ any, RES any](zipkinTracer *stdzipkin.Tracer) []httptransport.ClientOption[REQ, RES] {
	var options []httptransport.ClientOption[REQ, RES]

	if zipkinTracer != nil {
		// Zipkin GRPC Client Trace can either be instantiated per gRPC method with a
		// provided operation name or a global tracing client can be instantiated
		// without an operation name and fed to each Go kit client as ClientOption.
		// In the latter case, the operation name will be the endpoint's grpc method
		// path.
		//
		// In this example, we demonstrace a global tracing client.
		options = append(options, zipkin.HTTPClientTrace[REQ, RES](zipkinTracer))

	}
	return options
}
