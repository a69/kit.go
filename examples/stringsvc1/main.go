package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/a69/kit.go/endpoint"
	httptransport "github.com/a69/kit.go/transport/http"
)

// StringService provides operations on strings.
type StringService interface {
	Uppercase(string) (string, error)
	Count(string) int
}

// stringService is a concrete implementation of StringService
type stringService struct{}

func (stringService) Uppercase(s string) (string, error) {
	if s == "" {
		return "", ErrEmpty
	}
	return strings.ToUpper(s), nil
}

func (stringService) Count(s string) int {
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
func makeUppercaseEndpoint(svc StringService) endpoint.Endpoint[uppercaseRequest, uppercaseResponse] {
	return func(_ context.Context, request uppercaseRequest) (uppercaseResponse, error) {
		v, err := svc.Uppercase(request.S)
		if err != nil {
			return uppercaseResponse{v, err.Error()}, nil
		}
		return uppercaseResponse{v, ""}, nil
	}
}

func makeCountEndpoint(svc StringService) endpoint.Endpoint[countRequest, countResponse] {
	return func(_ context.Context, request countRequest) (countResponse, error) {
		req := request
		v := svc.Count(req.S)
		return countResponse{v}, nil
	}
}

// Transports expose the service to the network. In this first example we utilize JSON over HTTP.
func main() {
	svc := stringService{}

	uppercaseHandler := httptransport.NewServer[uppercaseRequest, uppercaseResponse](
		makeUppercaseEndpoint(svc),
		decodeRequest[uppercaseRequest],
		encodeResponse[uppercaseResponse],
	)

	countHandler := httptransport.NewServer[countRequest, countResponse](
		makeCountEndpoint(svc),
		decodeRequest[countRequest],
		encodeResponse[countResponse],
	)

	http.Handle("/uppercase", uppercaseHandler)
	http.Handle("/count", countHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func decodeRequest[REQ any](_ context.Context, r *http.Request) (req REQ, err error) {
	json.NewDecoder(r.Body).Decode(&req)
	return
}

func encodeResponse[RES any](_ context.Context, w http.ResponseWriter, response RES) error {
	return json.NewEncoder(w).Encode(response)
}
