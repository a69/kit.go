package test

import (
	"context"
	"fmt"

	"github.com/a69/kit.go/endpoint"
	grpctransport "github.com/a69/kit.go/transport/grpc"
	"github.com/a69/kit.go/transport/grpc/_grpc_test/pb"
)

type service struct{}

func (service) Test(ctx context.Context, a string, b int64) (context.Context, string, error) {
	return nil, fmt.Sprintf("%s = %d", a, b), nil
}

func NewService() Service {
	return service{}
}

func makeTestEndpoint(svc Service) endpoint.Endpoint[TestRequest, TestResponse] {
	return func(ctx context.Context, request TestRequest) (TestResponse, error) {
		req := request
		newCtx, v, err := svc.Test(ctx, req.A, req.B)
		return TestResponse{
			V:   v,
			Ctx: newCtx,
		}, err
	}
}

type serverBinding struct {
	pb.UnimplementedTestServer

	test *grpctransport.Server[TestRequest, TestResponse]
}

func (b *serverBinding) Test(ctx context.Context, req *pb.TestRequest) (*pb.TestResponse, error) {
	_, response, err := b.test.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return response.(*pb.TestResponse), nil
}

func NewBinding(svc Service) *serverBinding {
	return &serverBinding{
		test: grpctransport.NewServer(
			makeTestEndpoint(svc),
			decodeRequest,
			encodeResponse,
			grpctransport.ServerBefore[TestRequest, TestResponse](
				extractCorrelationID,
			),
			grpctransport.ServerBefore[TestRequest, TestResponse](
				displayServerRequestHeaders,
			),
			grpctransport.ServerAfter[TestRequest, TestResponse](
				injectResponseHeader,
				injectResponseTrailer,
				injectConsumedCorrelationID,
			),
			grpctransport.ServerAfter[TestRequest, TestResponse](
				displayServerResponseHeaders,
				displayServerResponseTrailers,
			),
		),
	}
}
