package test

import (
	"context"

	"google.golang.org/grpc"

	"github.com/a69/kit.go/endpoint"
	grpctransport "github.com/a69/kit.go/transport/grpc"
	"github.com/a69/kit.go/transport/grpc/_grpc_test/pb"
)

type clientBinding struct {
	test endpoint.Endpoint[TestRequest, TestResponse]
}

func (c *clientBinding) Test(ctx context.Context, a string, b int64) (context.Context, string, error) {
	response, err := c.test(ctx, TestRequest{A: a, B: b})
	if err != nil {
		return nil, "", err
	}
	r := response
	return r.Ctx, r.V, nil
}

func NewClient(cc *grpc.ClientConn) Service {
	return &clientBinding{
		test: grpctransport.NewClient(
			cc,
			"pb.Test",
			"Test",
			encodeRequest,
			decodeResponse,
			&pb.TestResponse{},
			grpctransport.ClientBefore[TestRequest, TestResponse](
				injectCorrelationID,
			),
			grpctransport.ClientBefore[TestRequest, TestResponse](
				displayClientRequestHeaders,
			),
			grpctransport.ClientAfter[TestRequest, TestResponse](
				displayClientResponseHeaders,
				displayClientResponseTrailers,
			),
			grpctransport.ClientAfter[TestRequest, TestResponse](
				extractConsumedCorrelationID,
			),
		).Endpoint(),
	}
}
