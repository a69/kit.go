package addtransport

import (
	"context"
	"time"

	"github.com/sony/gobreaker"

	"github.com/a69/kit.go/circuitbreaker"
	"github.com/a69/kit.go/endpoint"
	"github.com/a69/kit.go/examples/addsvc/pkg/addendpoint"
	"github.com/a69/kit.go/examples/addsvc/pkg/addservice"
	addthrift "github.com/a69/kit.go/examples/addsvc/thrift/gen-go/addsvc"
)

type thriftServer struct {
	ctx       context.Context
	endpoints addendpoint.Set
}

// NewThriftServer makes a set of endpoints available as a Thrift service.
func NewThriftServer(endpoints addendpoint.Set) addthrift.AddService {
	return &thriftServer{
		endpoints: endpoints,
	}
}

func (s *thriftServer) Sum(ctx context.Context, a int64, b int64) (*addthrift.SumReply, error) {
	request := addendpoint.SumRequest{A: int(a), B: int(b)}
	response, err := s.endpoints.SumEndpoint(ctx, request)
	if err != nil {
		return nil, err
	}
	return &addthrift.SumReply{Value: int64(response.V), Err: err2str(response.Err)}, nil
}

func (s *thriftServer) Concat(ctx context.Context, a string, b string) (*addthrift.ConcatReply, error) {
	request := addendpoint.ConcatRequest{A: a, B: b}
	response, err := s.endpoints.ConcatEndpoint(ctx, request)
	if err != nil {
		return nil, err
	}
	return &addthrift.ConcatReply{Value: response.V, Err: err2str(response.Err)}, nil
}

// NewThriftClient returns an AddService backed by a Thrift server described by
// the provided client. The caller is responsible for constructing the client,
// and eventually closing the underlying transport. We bake-in certain middlewares,
// implementing the client library pattern.
func NewThriftClient(client *addthrift.AddServiceClient) addservice.Service {

	// Each individual endpoint is an http/transport.Client (which implements
	// endpoint.Endpoint) that gets wrapped with various middlewares. If you
	// could rely on a consistent set of client behavior.
	var sumEndpoint endpoint.Endpoint[addendpoint.SumRequest, addendpoint.SumResponse]
	{
		sumEndpoint = MakeThriftSumEndpoint(client)
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
		concatEndpoint = MakeThriftConcatEndpoint(client)
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
	}
}

// MakeThriftSumEndpoint returns an endpoint that invokes the passed Thrift client.
// Useful only in clients, and only until a proper transport/thrift.Client exists.
func MakeThriftSumEndpoint(client *addthrift.AddServiceClient) endpoint.Endpoint[addendpoint.SumRequest, addendpoint.SumResponse] {
	return func(ctx context.Context, request addendpoint.SumRequest) (addendpoint.SumResponse, error) {
		reply, err := client.Sum(ctx, int64(request.A), int64(request.B))
		if err == addservice.ErrIntOverflow {
			return addendpoint.SumResponse{}, err // special case; see comment on ErrIntOverflow
		}
		return addendpoint.SumResponse{V: int(reply.Value), Err: err}, nil
	}
}

// MakeThriftConcatEndpoint returns an endpoint that invokes the passed Thrift
// client. Useful only in clients, and only until a proper
// transport/thrift.Client exists.
func MakeThriftConcatEndpoint(client *addthrift.AddServiceClient) endpoint.Endpoint[addendpoint.ConcatRequest, addendpoint.ConcatResponse] {
	return func(ctx context.Context, request addendpoint.ConcatRequest) (addendpoint.ConcatResponse, error) {
		reply, err := client.Concat(ctx, request.A, request.B)
		return addendpoint.ConcatResponse{V: reply.Value, Err: err}, nil
	}
}
