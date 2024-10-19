package addtransport

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/a69/kit.go/circuitbreaker"
	"github.com/a69/kit.go/endpoint"
	"github.com/a69/kit.go/examples/addsvc/pkg/addendpoint"
	"github.com/a69/kit.go/examples/addsvc/pkg/addservice"
	"github.com/a69/kit.go/log"
	"github.com/a69/kit.go/tracing/opentracing"
	"github.com/a69/kit.go/transport/http/jsonrpc"
	stdopentracing "github.com/opentracing/opentracing-go"
	"github.com/sony/gobreaker"
)

// NewJSONRPCHandler returns a JSON RPC Server/Handler that can be passed to http.Handle()
func NewJSONRPCHandler(endpoints addendpoint.Set, logger log.Logger) *jsonrpc.Server {
	handler := jsonrpc.NewServer(
		makeEndpointCodecMap(endpoints),
		jsonrpc.ServerErrorLogger(logger),
	)
	return handler
}

// NewJSONRPCClient returns an addservice backed by a JSON RPC over HTTP server
// living at the remote instance. We expect instance to come from a service
// discovery system, so likely of the form "host:port". We bake-in certain
// middlewares, implementing the client library pattern.
func NewJSONRPCClient(instance string, tracer stdopentracing.Tracer, logger log.Logger) (addservice.Service, error) {
	// Quickly sanitize the instance string.
	if !strings.HasPrefix(instance, "http") {
		instance = "http://" + instance
	}
	u, err := url.Parse(instance)
	if err != nil {
		return nil, err
	}

	var sumEndpoint endpoint.Endpoint[addendpoint.SumRequest, addendpoint.SumResponse]
	{
		sumEndpoint = jsonrpc.NewClient(
			u,
			"sum",
			jsonrpc.ClientRequestEncoder[addendpoint.SumRequest, addendpoint.SumResponse](encodeSumRequest),
			jsonrpc.ClientResponseDecoder[addendpoint.SumRequest, addendpoint.SumResponse](decodeSumResponse),
		).Endpoint()
		sumEndpoint = opentracing.TraceClient[addendpoint.SumRequest, addendpoint.SumResponse](tracer, "Sum")(sumEndpoint)
		sumEndpoint = makeLimiter[addendpoint.SumRequest, addendpoint.SumResponse]()(sumEndpoint)
		sumEndpoint = circuitbreaker.Gobreaker[addendpoint.SumRequest, addendpoint.SumResponse](gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "Sum",
			Timeout: 30 * time.Second,
		}))(sumEndpoint)
	}

	var concatEndpoint endpoint.Endpoint[addendpoint.ConcatRequest, addendpoint.ConcatResponse]
	{
		concatEndpoint = jsonrpc.NewClient(
			u,
			"concat",
			jsonrpc.ClientRequestEncoder[addendpoint.ConcatRequest, addendpoint.ConcatResponse](encodeConcatRequest),
			jsonrpc.ClientResponseDecoder[addendpoint.ConcatRequest, addendpoint.ConcatResponse](decodeConcatResponse),
		).Endpoint()
		concatEndpoint = opentracing.TraceClient[addendpoint.ConcatRequest, addendpoint.ConcatResponse](tracer, "Concat")(concatEndpoint)
		concatEndpoint = makeLimiter[addendpoint.ConcatRequest, addendpoint.ConcatResponse]()(concatEndpoint)
		concatEndpoint = circuitbreaker.Gobreaker[addendpoint.ConcatRequest, addendpoint.ConcatResponse](gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "Concat",
			Timeout: 30 * time.Second,
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

// makeEndpointCodecMap returns a codec map configured for the addsvc.
func makeEndpointCodecMap(endpoints addendpoint.Set) jsonrpc.EndpointCodecMap {
	return jsonrpc.EndpointCodecMap{
		"sum": &jsonrpc.EndpointCodec[addendpoint.SumRequest, addendpoint.SumResponse]{
			Endpoint: endpoints.SumEndpoint,
			Decode:   decodeSumRequest,
			Encode:   encodeSumResponse,
		},
		"concat": &jsonrpc.EndpointCodec[addendpoint.ConcatRequest, addendpoint.ConcatResponse]{
			Endpoint: endpoints.ConcatEndpoint,
			Decode:   decodeConcatRequest,
			Encode:   encodeConcatResponse,
		},
	}
}

func decodeSumRequest(_ context.Context, msg json.RawMessage) (addendpoint.SumRequest, error) {
	var req addendpoint.SumRequest
	err := json.Unmarshal(msg, &req)
	if err != nil {
		return req, &jsonrpc.Error{
			Code:    -32000,
			Message: fmt.Sprintf("couldn't unmarshal body to sum request: %s", err),
		}
	}
	return req, nil
}

func encodeSumResponse(_ context.Context, res addendpoint.SumResponse) (json.RawMessage, error) {

	b, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("couldn't marshal response: %s", err)
	}
	return b, nil
}

func decodeSumResponse(_ context.Context, res jsonrpc.Response) (addendpoint.SumResponse, error) {
	if res.Error != nil {
		return addendpoint.SumResponse{}, *res.Error
	}
	var sumres addendpoint.SumResponse
	err := json.Unmarshal(res.Result, &sumres)
	if err != nil {
		return addendpoint.SumResponse{}, fmt.Errorf("couldn't unmarshal body to SumResponse: %s", err)
	}
	return sumres, nil
}

func encodeSumRequest(_ context.Context, req *addendpoint.SumRequest) (json.RawMessage, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("couldn't marshal request: %s", err)
	}
	return b, nil
}

func decodeConcatRequest(_ context.Context, msg json.RawMessage) (addendpoint.ConcatRequest, error) {
	var req addendpoint.ConcatRequest
	err := json.Unmarshal(msg, &req)
	if err != nil {
		return req, &jsonrpc.Error{
			Code:    -32000,
			Message: fmt.Sprintf("couldn't unmarshal body to concat request: %s", err),
		}
	}
	return req, nil
}

func encodeConcatResponse(_ context.Context, res addendpoint.ConcatResponse) (json.RawMessage, error) {
	b, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("couldn't marshal response: %s", err)
	}
	return b, nil
}

func decodeConcatResponse(_ context.Context, res jsonrpc.Response) (addendpoint.ConcatResponse, error) {
	if res.Error != nil {
		return addendpoint.ConcatResponse{}, *res.Error
	}
	var concatres addendpoint.ConcatResponse
	err := json.Unmarshal(res.Result, &concatres)
	if err != nil {
		return addendpoint.ConcatResponse{}, fmt.Errorf("couldn't unmarshal body to ConcatResponse: %s", err)
	}
	return concatres, nil
}

func encodeConcatRequest(_ context.Context, req *addendpoint.ConcatRequest) (json.RawMessage, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("couldn't marshal request: %s", err)
	}
	return b, nil
}
