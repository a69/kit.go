package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/a69/kit.go/endpoint"
	"github.com/a69/kit.go/transport"
	"github.com/go-kit/log"
)

// Handler which should be called from the gRPC binding of the service
// implementation. The incoming request parameter, and returned response
// parameter, are both gRPC types, not user-domain.
type Handler interface {
	ServeGRPC(ctx context.Context, request interface{}) (context.Context, interface{}, error)
}

// Server wraps an endpoint and implements grpc.Handler.
type Server[REQ any, RES any] struct {
	e            endpoint.Endpoint[REQ, RES]
	dec          DecodeRequestFunc[REQ]
	enc          EncodeResponseFunc[RES]
	before       []ServerRequestFunc
	after        []ServerResponseFunc
	finalizer    []ServerFinalizerFunc
	errorHandler transport.ErrorHandler
}

// NewServer constructs a new server, which implements wraps the provided
// endpoint and implements the Handler interface. Consumers should write
// bindings that adapt the concrete gRPC methods from their compiled protobuf
// definitions to individual handlers. Request and response objects are from the
// caller business domain, not gRPC request and reply types.
func NewServer[REQ any, RES any](
	e endpoint.Endpoint[REQ, RES],
	dec DecodeRequestFunc[REQ],
	enc EncodeResponseFunc[RES],
	options ...ServerOption[REQ, RES],
) *Server[REQ, RES] {
	s := &Server[REQ, RES]{
		e:            e,
		dec:          dec,
		enc:          enc,
		errorHandler: transport.NewLogErrorHandler(log.NewNopLogger()),
	}
	for _, option := range options {
		option(s)
	}
	return s
}

// ServerOption sets an optional parameter for servers.
type ServerOption[REQ any, RES any] func(*Server[REQ, RES])

// ServerBefore functions are executed on the gRPC request object before the
// request is decoded.
func ServerBefore[REQ any, RES any](before ...ServerRequestFunc) ServerOption[REQ, RES] {
	return func(s *Server[REQ, RES]) { s.before = append(s.before, before...) }
}

// ServerAfter functions are executed on the gRPC response writer after the
// endpoint is invoked, but before anything is written to the client.
func ServerAfter[REQ any, RES any](after ...ServerResponseFunc) ServerOption[REQ, RES] {
	return func(s *Server[REQ, RES]) { s.after = append(s.after, after...) }
}

// ServerErrorLogger is used to log non-terminal errors. By default, no errors
// are logged.
// Deprecated: Use ServerErrorHandler instead.
func ServerErrorLogger[REQ any, RES any](logger log.Logger) ServerOption[REQ, RES] {
	return func(s *Server[REQ, RES]) { s.errorHandler = transport.NewLogErrorHandler(logger) }
}

// ServerErrorHandler is used to handle non-terminal errors. By default, non-terminal errors
// are ignored.
func ServerErrorHandler[REQ any, RES any](errorHandler transport.ErrorHandler) ServerOption[REQ, RES] {
	return func(s *Server[REQ, RES]) { s.errorHandler = errorHandler }
}

// ServerFinalizer is executed at the end of every gRPC request.
// By default, no finalizer is registered.
func ServerFinalizer[REQ any, RES any](f ...ServerFinalizerFunc) ServerOption[REQ, RES] {
	return func(s *Server[REQ, RES]) { s.finalizer = append(s.finalizer, f...) }
}

// ServeGRPC implements the Handler interface.
func (s Server[REQ, RES]) ServeGRPC(ctx context.Context, req interface{}) (retctx context.Context, resp interface{}, err error) {
	// Retrieve gRPC metadata.
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.MD{}
	}

	if len(s.finalizer) > 0 {
		defer func() {
			for _, f := range s.finalizer {
				f(ctx, err)
			}
		}()
	}

	for _, f := range s.before {
		ctx = f(ctx, md)
	}

	var (
		request  REQ
		response RES
		grpcResp interface{}
	)

	request, err = s.dec(ctx, req)
	if err != nil {
		s.errorHandler.Handle(ctx, err)
		return ctx, nil, err
	}

	response, err = s.e(ctx, request)
	if err != nil {
		s.errorHandler.Handle(ctx, err)
		return ctx, nil, err
	}

	var mdHeader, mdTrailer metadata.MD
	for _, f := range s.after {
		ctx = f(ctx, &mdHeader, &mdTrailer)
	}

	grpcResp, err = s.enc(ctx, response)
	if err != nil {
		s.errorHandler.Handle(ctx, err)
		return ctx, nil, err
	}

	if len(mdHeader) > 0 {
		if err = grpc.SendHeader(ctx, mdHeader); err != nil {
			s.errorHandler.Handle(ctx, err)
			return ctx, nil, err
		}
	}

	if len(mdTrailer) > 0 {
		if err = grpc.SetTrailer(ctx, mdTrailer); err != nil {
			s.errorHandler.Handle(ctx, err)
			return ctx, nil, err
		}
	}

	return ctx, grpcResp, nil
}

// ServerFinalizerFunc can be used to perform work at the end of an gRPC
// request, after the response has been written to the client.
type ServerFinalizerFunc func(ctx context.Context, err error)

// Interceptor is a grpc UnaryInterceptor that injects the method name into
// context so it can be consumed by Go kit gRPC middlewares. The Interceptor
// typically is added at creation time of the grpc-go server.
// Like this: `grpc.NewServer(grpc.UnaryInterceptor(kitgrpc.Interceptor))`
func Interceptor(
	ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
) (resp interface{}, err error) {
	ctx = context.WithValue(ctx, ContextKeyRequestMethod, info.FullMethod)
	return handler(ctx, req)
}
