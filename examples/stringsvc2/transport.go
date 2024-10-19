package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/a69/kit.go/endpoint"
)

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
		v := svc.Count(request.S)
		return countResponse{v}, nil
	}
}

func decodeRequest[REQ any](_ context.Context, r *http.Request) (req REQ, err error) {
	err = json.NewDecoder(r.Body).Decode(&req)
	return
}

func encodeResponse[RES any](_ context.Context, w http.ResponseWriter, response RES) error {
	return json.NewEncoder(w).Encode(response)
}

type uppercaseRequest struct {
	S string `json:"s"`
}

type uppercaseResponse struct {
	V   string `json:"v"`
	Err string `json:"err,omitempty"`
}

type countRequest struct {
	S string `json:"s"`
}

type countResponse struct {
	V int `json:"v"`
}
