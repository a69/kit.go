package endpoint_test

import (
	"context"
	"fmt"

	"github.com/a69/kit.go/endpoint"
)

func ExampleChain() {
	e := endpoint.Chain[struct{}, struct{}](
		annotate[struct{}, struct{}]("first"),
		annotate[struct{}, struct{}]("second"),
		annotate[struct{}, struct{}]("third"),
	)(myEndpoint)

	if _, err := e(ctx, req); err != nil {
		panic(err)
	}

	// Output:
	// first pre
	// second pre
	// third pre
	// my endpoint!
	// third post
	// second post
	// first post
}

var (
	ctx = context.Background()
	req = struct{}{}
)

func annotate[REQ any, RES any](s string) endpoint.Middleware[REQ, RES] {
	return func(next endpoint.Endpoint[REQ, RES]) endpoint.Endpoint[REQ, RES] {
		return func(ctx context.Context, request REQ) (RES, error) {
			fmt.Println(s, "pre")
			defer fmt.Println(s, "post")
			return next(ctx, request)
		}
	}
}

func myEndpoint(context.Context, struct{}) (struct{}, error) {
	fmt.Println("my endpoint!")
	return struct{}{}, nil
}
