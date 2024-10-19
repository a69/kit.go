package addtransport

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc"

	stdopentracing "github.com/opentracing/opentracing-go"
	stdzipkin "github.com/openzipkin/zipkin-go"
	"github.com/sony/gobreaker"
	"golang.org/x/time/rate"

	"github.com/a69/kit.go/circuitbreaker"
	"github.com/a69/kit.go/endpoint"
	"github.com/a69/kit.go/log"
	"github.com/a69/kit.go/ratelimit"
	"github.com/a69/kit.go/tracing/opentracing"
	"github.com/a69/kit.go/tracing/zipkin"
	grpctransport "github.com/a69/kit.go/transport/grpc"

	"github.com/a69/kit.go/examples/addsvc/pb"
	"github.com/a69/kit.go/examples/addsvc/pkg/addendpoint"
	"github.com/a69/kit.go/examples/addsvc/pkg/addservice"
)

type grpcServer struct {
	sum    grpctransport.Handler
	concat grpctransport.Handler
}

// NewGRPCServer makes a set of endpoints available as a gRPC AddServer.
func NewGRPCServer(endpoints addendpoint.Set, otTracer stdopentracing.Tracer, zipkinTracer *stdzipkin.Tracer, logger log.Logger) pb.AddServer {

	return &grpcServer{
		sum: grpctransport.NewServer(
			endpoints.SumEndpoint,
			decodeGRPCSumRequest,
			encodeGRPCSumResponse,
			append(makeServerOptions[addendpoint.SumRequest, addendpoint.SumResponse](zipkinTracer), grpctransport.ServerBefore[addendpoint.SumRequest, addendpoint.SumResponse](opentracing.GRPCToContext(otTracer, "Sum", logger)))...,
		),
		concat: grpctransport.NewServer(
			endpoints.ConcatEndpoint,
			decodeGRPCConcatRequest,
			encodeGRPCConcatResponse,
			append(makeServerOptions[addendpoint.ConcatRequest, addendpoint.ConcatResponse](zipkinTracer), grpctransport.ServerBefore[addendpoint.ConcatRequest, addendpoint.ConcatResponse](opentracing.GRPCToContext(otTracer, "Concat", logger)))...,
		),
	}
}

func (s *grpcServer) Sum(ctx context.Context, req *pb.SumRequest) (*pb.SumReply, error) {
	_, rep, err := s.sum.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return rep.(*pb.SumReply), nil
}

func (s *grpcServer) Concat(ctx context.Context, req *pb.ConcatRequest) (*pb.ConcatReply, error) {
	_, rep, err := s.concat.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return rep.(*pb.ConcatReply), nil
}

// NewGRPCClient returns an AddService backed by a gRPC server at the other end
// of the conn. The caller is responsible for constructing the conn, and
// eventually closing the underlying transport. We bake-in certain middlewares,
// implementing the client library pattern.
func NewGRPCClient(conn *grpc.ClientConn, otTracer stdopentracing.Tracer, zipkinTracer *stdzipkin.Tracer, logger log.Logger) addservice.Service {

	// Each individual endpoint is an grpc/transport.Client (which implements
	// endpoint.Endpoint) that gets wrapped with various middlewares. If you
	// made your own client library, you'd do this work there, so your server
	// could rely on a consistent set of client behavior.
	var sumEndpoint endpoint.Endpoint[addendpoint.SumRequest, addendpoint.SumResponse]
	{
		sumEndpoint = grpctransport.NewClient(
			conn,
			"pb.Add",
			"Sum",
			encodeGRPCSumRequest,
			decodeGRPCSumResponse,
			pb.SumReply{},
			append(makeClientOptions[addendpoint.SumRequest, addendpoint.SumResponse](zipkinTracer), grpctransport.ClientBefore[addendpoint.SumRequest, addendpoint.SumResponse](opentracing.ContextToGRPC(otTracer, logger)))...,
		).Endpoint()
		sumEndpoint = opentracing.TraceClient[addendpoint.SumRequest, addendpoint.SumResponse](otTracer, "Sum")(sumEndpoint)
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
		concatEndpoint = grpctransport.NewClient(
			conn,
			"pb.Add",
			"Concat",
			encodeGRPCConcatRequest,
			decodeGRPCConcatResponse,
			pb.ConcatReply{},
			append(makeClientOptions[addendpoint.ConcatRequest, addendpoint.ConcatResponse](zipkinTracer), grpctransport.ClientBefore[addendpoint.ConcatRequest, addendpoint.ConcatResponse](opentracing.ContextToGRPC(otTracer, logger)))...,
		).Endpoint()
		concatEndpoint = opentracing.TraceClient[addendpoint.ConcatRequest, addendpoint.ConcatResponse](otTracer, "Concat")(concatEndpoint)
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

// decodeGRPCSumRequest is a transport/grpc.DecodeRequestFunc that converts a
// gRPC sum request to a user-domain sum request. Primarily useful in a server.
func decodeGRPCSumRequest(_ context.Context, grpcReq interface{}) (addendpoint.SumRequest, error) {
	req := grpcReq.(*pb.SumRequest)
	return addendpoint.SumRequest{A: int(req.A), B: int(req.B)}, nil
}

// decodeGRPCConcatRequest is a transport/grpc.DecodeRequestFunc that converts a
// gRPC concat request to a user-domain concat request. Primarily useful in a
// server.
func decodeGRPCConcatRequest(_ context.Context, grpcReq interface{}) (addendpoint.ConcatRequest, error) {
	req := grpcReq.(*pb.ConcatRequest)
	return addendpoint.ConcatRequest{A: req.A, B: req.B}, nil
}

// decodeGRPCSumResponse is a transport/grpc.DecodeResponseFunc that converts a
// gRPC sum reply to a user-domain sum response. Primarily useful in a client.
func decodeGRPCSumResponse(_ context.Context, grpcReply interface{}) (addendpoint.SumResponse, error) {
	reply := grpcReply.(*pb.SumReply)
	return addendpoint.SumResponse{V: int(reply.V), Err: str2err(reply.Err)}, nil
}

// decodeGRPCConcatResponse is a transport/grpc.DecodeResponseFunc that converts
// a gRPC concat reply to a user-domain concat response. Primarily useful in a
// client.
func decodeGRPCConcatResponse(_ context.Context, grpcReply interface{}) (addendpoint.ConcatResponse, error) {
	reply := grpcReply.(*pb.ConcatReply)
	return addendpoint.ConcatResponse{V: reply.V, Err: str2err(reply.Err)}, nil
}

// encodeGRPCSumResponse is a transport/grpc.EncodeResponseFunc that converts a
// user-domain sum response to a gRPC sum reply. Primarily useful in a server.
func encodeGRPCSumResponse(_ context.Context, response addendpoint.SumResponse) (interface{}, error) {
	return &pb.SumReply{V: int64(response.V), Err: err2str(response.Err)}, nil
}

// encodeGRPCConcatResponse is a transport/grpc.EncodeResponseFunc that converts
// a user-domain concat response to a gRPC concat reply. Primarily useful in a
// server.
func encodeGRPCConcatResponse(_ context.Context, response addendpoint.ConcatResponse) (interface{}, error) {
	return &pb.ConcatReply{V: response.V, Err: err2str(response.Err)}, nil
}

// encodeGRPCSumRequest is a transport/grpc.EncodeRequestFunc that converts a
// user-domain sum request to a gRPC sum request. Primarily useful in a client.
func encodeGRPCSumRequest(_ context.Context, request addendpoint.SumRequest) (interface{}, error) {
	return &pb.SumRequest{A: int64(request.A), B: int64(request.B)}, nil
}

// encodeGRPCConcatRequest is a transport/grpc.EncodeRequestFunc that converts a
// user-domain concat request to a gRPC concat request. Primarily useful in a
// client.
func encodeGRPCConcatRequest(_ context.Context, request addendpoint.ConcatRequest) (interface{}, error) {
	return &pb.ConcatRequest{A: request.A, B: request.B}, nil
}

// These annoying helper functions are required to translate Go error types to
// and from strings, which is the type we use in our IDLs to represent errors.
// There is special casing to treat empty strings as nil errors.

func str2err(s string) error {
	if s == "" {
		return nil
	}
	return errors.New(s)
}

func err2str(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func makeClientOptions[REQ any, RES any](zipkinTracer *stdzipkin.Tracer) []grpctransport.ClientOption[REQ, RES] {
	var options []grpctransport.ClientOption[REQ, RES]

	if zipkinTracer != nil {
		// Zipkin GRPC Client Trace can either be instantiated per gRPC method with a
		// provided operation name or a global tracing client can be instantiated
		// without an operation name and fed to each Go kit client as ClientOption.
		// In the latter case, the operation name will be the endpoint's grpc method
		// path.
		//
		// In this example, we demonstrace a global tracing client.
		options = append(options, zipkin.GRPCClientTrace[REQ, RES](zipkinTracer))

	}
	return options
}

func makeServerOptions[REQ any, RES any](zipkinTracer *stdzipkin.Tracer) []grpctransport.ServerOption[REQ, RES] {
	var options []grpctransport.ServerOption[REQ, RES]

	if zipkinTracer != nil {
		// Zipkin GRPC Client Trace can either be instantiated per gRPC method with a
		// provided operation name or a global tracing client can be instantiated
		// without an operation name and fed to each Go kit client as ClientOption.
		// In the latter case, the operation name will be the endpoint's grpc method
		// path.
		//
		// In this example, we demonstrace a global tracing client.
		options = append(options, zipkin.GRPCServerTrace[REQ, RES](zipkinTracer))

	}
	return options
}

func makeLimiter[REQ any, RES any]() endpoint.Middleware[REQ, RES] {
	return ratelimit.NewErroringLimiter[REQ, RES](rate.NewLimiter(rate.Every(time.Second), 100))
}
