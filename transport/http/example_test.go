package http

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
)

func ExamplePopulateRequestContext() {
	handler := NewServer(
		func(ctx context.Context, _ struct{}) (response struct{}, err error) {
			fmt.Println("Method", ctx.Value(ContextKeyRequestMethod).(string))
			fmt.Println("RequestPath", ctx.Value(ContextKeyRequestPath).(string))
			fmt.Println("RequestURI", ctx.Value(ContextKeyRequestURI).(string))
			fmt.Println("X-Request-ID", ctx.Value(ContextKeyRequestXRequestID).(string))
			return struct{}{}, nil
		},
		func(context.Context, *http.Request) (struct{}, error) { return struct{}{}, nil },
		func(context.Context, http.ResponseWriter, struct{}) error { return nil },
		ServerBefore[struct{}, struct{}](PopulateRequestContext),
	)

	server := httptest.NewServer(handler)
	defer server.Close()

	req, _ := http.NewRequest("PATCH", fmt.Sprintf("%s/search?q=sympatico", server.URL), nil)
	req.Header.Set("X-Request-Id", "a1b2c3d4e5")
	http.DefaultClient.Do(req)

	// Output:
	// Method PATCH
	// RequestPath /search
	// RequestURI /search?q=sympatico
	// X-Request-ID a1b2c3d4e5
}
