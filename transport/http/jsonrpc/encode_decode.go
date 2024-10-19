package jsonrpc

import (
	"encoding/json"
	httptransport "github.com/a69/kit.go/transport/http"
	"net/http"

	"github.com/a69/kit.go/endpoint"

	"context"
)

// Server-Side Codec

// EndpointCodec defines a server Endpoint and its associated codecs
type EndpointCodec[REQ any, RES any] struct {
	Endpoint endpoint.Endpoint[REQ, RES]
	Decode   DecodeRequestFunc[REQ]
	Encode   EncodeResponseFunc[RES]
}

func (e EndpointCodec[REQ, RES]) Handle(ctx context.Context, after []httptransport.ServerResponseFunc, w http.ResponseWriter, params json.RawMessage) (res json.RawMessage, err error) { // Decode the JSON "params"
	reqParams, err := e.Decode(ctx, params)
	if err != nil {
		return
	}

	// Call the Endpoint with the params
	response, err := e.Endpoint(ctx, reqParams)
	if err != nil {
		return
	}

	for _, f := range after {
		ctx = f(ctx, w)
	}

	return e.Encode(ctx, response)
}

type EndpointHandler interface {
	Handle(ctx context.Context, after []httptransport.ServerResponseFunc, w http.ResponseWriter, params json.RawMessage) (json.RawMessage, error)
}

// EndpointCodecMap maps the Request.Method to the proper EndpointCodec
type EndpointCodecMap map[string]EndpointHandler

// DecodeRequestFunc extracts a user-domain request object from raw JSON
// It's designed to be used in JSON RPC servers, for server-side endpoints.
// One straightforward DecodeRequestFunc could be something that unmarshals
// JSON from the request body to the concrete request type.
type DecodeRequestFunc[REQ any] func(context.Context, json.RawMessage) (request REQ, err error)

// EncodeResponseFunc encodes the passed response object to a JSON RPC result.
// It's designed to be used in HTTP servers, for server-side endpoints.
// One straightforward EncodeResponseFunc could be something that JSON encodes
// the object directly.
type EncodeResponseFunc[RES any] func(context.Context, RES) (response json.RawMessage, err error)

// Client-Side Codec

// EncodeRequestFunc encodes the given request object to raw JSON.
// It's designed to be used in JSON RPC clients, for client-side
// endpoints. One straightforward EncodeResponseFunc could be something that
// JSON encodes the object directly.
type EncodeRequestFunc[REQ any] func(context.Context, *REQ) (request json.RawMessage, err error)

// DecodeResponseFunc extracts a user-domain response object from an JSON RPC
// response object. It's designed to be used in JSON RPC clients, for
// client-side endpoints. It is the responsibility of this function to decide
// whether any error present in the JSON RPC response should be surfaced to the
// client endpoint.
type DecodeResponseFunc[RES any] func(context.Context, Response) (response RES, err error)
