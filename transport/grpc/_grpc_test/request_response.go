package test

import (
	"context"

	"github.com/a69/kit.go/transport/grpc/_grpc_test/pb"
)

func encodeRequest(ctx context.Context, req TestRequest) (interface{}, error) {
	r := req
	return &pb.TestRequest{A: r.A, B: r.B}, nil
}

func decodeRequest(ctx context.Context, req interface{}) (TestRequest, error) {
	r := req.(*pb.TestRequest)
	return TestRequest{A: r.A, B: r.B}, nil
}

func encodeResponse(ctx context.Context, resp TestResponse) (interface{}, error) {
	r := resp
	return &pb.TestResponse{V: r.V}, nil
}

func decodeResponse(ctx context.Context, resp interface{}) (TestResponse, error) {
	r := resp.(*pb.TestResponse)
	return TestResponse{V: r.V, Ctx: ctx}, nil
}
