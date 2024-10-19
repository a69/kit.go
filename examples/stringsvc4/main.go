package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"net/http"
	"strings"

	"github.com/a69/kit.go/endpoint"
	httptransport "github.com/a69/kit.go/transport/http"
	natstransport "github.com/a69/kit.go/transport/nats"

	"github.com/nats-io/nats.go"
)

// StringService provides operations on strings.
type StringService interface {
	Uppercase(context.Context, string) (string, error)
	Count(context.Context, string) int
}

// stringService is a concrete implementation of StringService
type stringService struct{}

func (stringService) Uppercase(_ context.Context, s string) (string, error) {
	if s == "" {
		return "", ErrEmpty
	}
	return strings.ToUpper(s), nil
}

func (stringService) Count(_ context.Context, s string) int {
	return len(s)
}

// ErrEmpty is returned when an input string is empty.
var ErrEmpty = errors.New("empty string")

// For each method, we define request and response structs
type uppercaseRequest struct {
	S string `json:"s"`
}

type uppercaseResponse struct {
	V   string `json:"v"`
	Err string `json:"err,omitempty"` // errors don't define JSON marshaling
}

type countRequest struct {
	S string `json:"s"`
}

type countResponse struct {
	V int `json:"v"`
}

// Endpoints are a primary abstraction in go-kit. An endpoint represents a single RPC (method in our service interface)
func makeUppercaseHTTPEndpoint(nc *nats.Conn) endpoint.Endpoint[uppercaseRequest, uppercaseResponse] {
	return natstransport.NewPublisher[uppercaseRequest, uppercaseResponse](
		nc,
		"stringsvc.uppercase",
		natstransport.EncodeJSONRequest[uppercaseRequest],
		decode[uppercaseResponse],
	).Endpoint()
}

func makeCountHTTPEndpoint(nc *nats.Conn) endpoint.Endpoint[countRequest, countResponse] {
	return natstransport.NewPublisher[countRequest, countResponse](
		nc,
		"stringsvc.count",
		natstransport.EncodeJSONRequest[countRequest],
		decode[countResponse],
	).Endpoint()
}

func makeUppercaseEndpoint(svc StringService) endpoint.Endpoint[uppercaseRequest, uppercaseResponse] {
	return func(ctx context.Context, request uppercaseRequest) (uppercaseResponse, error) {
		v, err := svc.Uppercase(ctx, request.S)
		if err != nil {
			return uppercaseResponse{v, err.Error()}, nil
		}
		return uppercaseResponse{v, ""}, nil
	}
}

func makeCountEndpoint(svc StringService) endpoint.Endpoint[countRequest, countResponse] {
	return func(ctx context.Context, request countRequest) (countResponse, error) {
		v := svc.Count(ctx, request.S)
		return countResponse{v}, nil
	}
}

// Transports expose the service to the network. In this fourth example we utilize JSON over NATS and HTTP.
func main() {
	svc := stringService{}

	natsURL := flag.String("nats-url", nats.DefaultURL, "URL for connection to NATS")
	flag.Parse()

	nc, err := nats.Connect(*natsURL)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	uppercaseHTTPHandler := httptransport.NewServer(
		makeUppercaseHTTPEndpoint(nc),
		decodeHTTPRequest[uppercaseRequest],
		httptransport.EncodeJSONResponse[uppercaseResponse],
	)

	countHTTPHandler := httptransport.NewServer(
		makeCountHTTPEndpoint(nc),
		decodeHTTPRequest[countRequest],
		httptransport.EncodeJSONResponse[countResponse],
	)

	uppercaseHandler := natstransport.NewSubscriber(
		makeUppercaseEndpoint(svc),
		decode[uppercaseRequest],
		natstransport.EncodeJSONResponse[uppercaseResponse],
	)

	countHandler := natstransport.NewSubscriber(
		makeCountEndpoint(svc),
		decode[countRequest],
		natstransport.EncodeJSONResponse[countResponse],
	)

	uSub, err := nc.QueueSubscribe("stringsvc.uppercase", "stringsvc", uppercaseHandler.ServeMsg(nc))
	if err != nil {
		log.Fatal(err)
	}
	defer uSub.Unsubscribe()

	cSub, err := nc.QueueSubscribe("stringsvc.count", "stringsvc", countHandler.ServeMsg(nc))
	if err != nil {
		log.Fatal(err)
	}
	defer cSub.Unsubscribe()

	http.Handle("/uppercase", uppercaseHTTPHandler)
	http.Handle("/count", countHTTPHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))

}

func decodeHTTPRequest[REQ any](_ context.Context, r *http.Request) (req REQ, err error) {
	err = json.NewDecoder(r.Body).Decode(&req)
	return
}

func decode[T any](_ context.Context, msg *nats.Msg) (t T, err error) {
	err = json.Unmarshal(msg.Data, &t)
	return
}
