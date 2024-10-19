package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/a69/kit.go/endpoint"
)

func makeUppercaseEndpoint(svc StringService) endpoint.Endpoint[uppercaseRequest, uppercaseResponse] {
	return func(ctx context.Context, request uppercaseRequest) (uppercaseResponse, error) {
		v, err := svc.Uppercase(request.S)
		if err != nil {
			return uppercaseResponse{v, err.Error()}, nil
		}
		return uppercaseResponse{v, ""}, nil
	}
}

func makeCountEndpoint(svc StringService) endpoint.Endpoint[countRequest, countResponse] {
	return func(ctx context.Context, request countRequest) (countResponse, error) {
		v := svc.Count(request.S)
		return countResponse{v}, nil
	}
}

func decodeRequest[REQ any](_ context.Context, r *http.Request) (req REQ, err error) {
	err = json.NewDecoder(r.Body).Decode(&req)
	return
}

func decodeResponse[RES any](_ context.Context, r *http.Response) (res RES, err error) {
	err = json.NewDecoder(r.Body).Decode(&res)
	return
}

func encodeResponse[RES any](_ context.Context, w http.ResponseWriter, response RES) error {
	return json.NewEncoder(w).Encode(response)
}

func encodeRequest[REQ any](_ context.Context, r *http.Request, request *REQ) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(request); err != nil {
		return err
	}
	r.Body = ioutil.NopCloser(&buf)
	return nil
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
