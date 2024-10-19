package profilesvc

// The profilesvc is just over HTTP, so we just have a single transport.go.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"

	"github.com/a69/kit.go/log"
	"github.com/a69/kit.go/transport"
	httptransport "github.com/a69/kit.go/transport/http"
)

var (
	// ErrBadRouting is returned when an expected path variable is missing.
	// It always indicates programmer error.
	ErrBadRouting = errors.New("inconsistent mapping between route and handler (programmer error)")
)

// MakeHTTPHandler mounts all of the service endpoints into an http.Handler.
// Useful in a profilesvc server.
func MakeHTTPHandler(s Service, logger log.Logger) http.Handler {
	r := mux.NewRouter()
	e := MakeServerEndpoints(s)

	// POST    /profiles/                          adds another profile
	// GET     /profiles/:id                       retrieves the given profile by id
	// PUT     /profiles/:id                       post updated profile information about the profile
	// PATCH   /profiles/:id                       partial updated profile information
	// DELETE  /profiles/:id                       remove the given profile
	// GET     /profiles/:id/addresses/            retrieve addresses associated with the profile
	// GET     /profiles/:id/addresses/:addressID  retrieve a particular profile address
	// POST    /profiles/:id/addresses/            add a new address
	// DELETE  /profiles/:id/addresses/:addressID  remove an address

	r.Methods("POST").Path("/profiles/").Handler(httptransport.NewServer(
		e.PostProfileEndpoint,
		decodePostProfileRequest,
		encodeResponse[PostProfileResponse],
		makeServerOption[PostProfileRequest, PostProfileResponse](logger)...,
	))
	r.Methods("GET").Path("/profiles/{id}").Handler(httptransport.NewServer(
		e.GetProfileEndpoint,
		decodeGetProfileRequest,
		encodeResponse[GetProfileResponse],
		makeServerOption[GetProfileRequest, GetProfileResponse](logger)...,
	))
	r.Methods("PUT").Path("/profiles/{id}").Handler(httptransport.NewServer(
		e.PutProfileEndpoint,
		decodePutProfileRequest,
		encodeResponse[PutProfileResponse],
		makeServerOption[PutProfileRequest, PutProfileResponse](logger)...,
	))
	r.Methods("PATCH").Path("/profiles/{id}").Handler(httptransport.NewServer(
		e.PatchProfileEndpoint,
		decodePatchProfileRequest,
		encodeResponse[PatchProfileResponse],
		makeServerOption[PatchProfileRequest, PatchProfileResponse](logger)...,
	))
	r.Methods("DELETE").Path("/profiles/{id}").Handler(httptransport.NewServer(
		e.DeleteProfileEndpoint,
		decodeDeleteProfileRequest,
		encodeResponse[DeleteProfileResponse],
		makeServerOption[DeleteProfileRequest, DeleteProfileResponse](logger)...,
	))
	r.Methods("GET").Path("/profiles/{id}/addresses/").Handler(httptransport.NewServer(
		e.GetAddressesEndpoint,
		decodeGetAddressesRequest,
		encodeResponse[GetAddressesResponse],
		makeServerOption[GetAddressesRequest, GetAddressesResponse](logger)...,
	))
	r.Methods("GET").Path("/profiles/{id}/addresses/{addressID}").Handler(httptransport.NewServer(
		e.GetAddressEndpoint,
		decodeGetAddressRequest,
		encodeResponse[GetAddressResponse],
		makeServerOption[GetAddressRequest, GetAddressResponse](logger)...,
	))
	r.Methods("POST").Path("/profiles/{id}/addresses/").Handler(httptransport.NewServer(
		e.PostAddressEndpoint,
		decodePostAddressRequest,
		encodeResponse[PostAddressResponse],
		makeServerOption[PostAddressRequest, PostAddressResponse](logger)...,
	))
	r.Methods("DELETE").Path("/profiles/{id}/addresses/{addressID}").Handler(httptransport.NewServer(
		e.DeleteAddressEndpoint,
		decodeDeleteAddressRequest,
		encodeResponse[DeleteAddressResponse],
		makeServerOption[DeleteAddressRequest, DeleteAddressResponse](logger)...,
	))
	return r
}

func decodePostProfileRequest(_ context.Context, r *http.Request) (request PostProfileRequest, err error) {
	var req PostProfileRequest
	if e := json.NewDecoder(r.Body).Decode(&req.Profile); e != nil {
		return PostProfileRequest{}, e
	}
	return req, nil
}

func decodeGetProfileRequest(_ context.Context, r *http.Request) (request GetProfileRequest, err error) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		return GetProfileRequest{}, ErrBadRouting
	}
	return GetProfileRequest{ID: id}, nil
}

func decodePutProfileRequest(_ context.Context, r *http.Request) (request PutProfileRequest, err error) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		return PutProfileRequest{}, ErrBadRouting
	}
	var profile Profile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		return PutProfileRequest{}, err
	}
	return PutProfileRequest{
		ID:      id,
		Profile: profile,
	}, nil
}

func decodePatchProfileRequest(_ context.Context, r *http.Request) (request PatchProfileRequest, err error) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		return PatchProfileRequest{}, ErrBadRouting
	}
	var profile Profile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		return PatchProfileRequest{}, err
	}
	return PatchProfileRequest{
		ID:      id,
		Profile: profile,
	}, nil
}

func decodeDeleteProfileRequest(_ context.Context, r *http.Request) (request DeleteProfileRequest, err error) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		return DeleteProfileRequest{}, ErrBadRouting
	}
	return DeleteProfileRequest{ID: id}, nil
}

func decodeGetAddressesRequest(_ context.Context, r *http.Request) (request GetAddressesRequest, err error) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		return GetAddressesRequest{}, ErrBadRouting
	}
	return GetAddressesRequest{ProfileID: id}, nil
}

func decodeGetAddressRequest(_ context.Context, r *http.Request) (request GetAddressRequest, err error) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		return GetAddressRequest{}, ErrBadRouting
	}
	addressID, ok := vars["addressID"]
	if !ok {
		return GetAddressRequest{}, ErrBadRouting
	}
	return GetAddressRequest{
		ProfileID: id,
		AddressID: addressID,
	}, nil
}

func decodePostAddressRequest(_ context.Context, r *http.Request) (request PostAddressRequest, err error) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		return PostAddressRequest{}, ErrBadRouting
	}
	var address Address
	if err := json.NewDecoder(r.Body).Decode(&address); err != nil {
		return PostAddressRequest{}, err
	}
	return PostAddressRequest{
		ProfileID: id,
		Address:   address,
	}, nil
}

func decodeDeleteAddressRequest(_ context.Context, r *http.Request) (request DeleteAddressRequest, err error) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		return DeleteAddressRequest{}, ErrBadRouting
	}
	addressID, ok := vars["addressID"]
	if !ok {
		return DeleteAddressRequest{}, ErrBadRouting
	}
	return DeleteAddressRequest{
		ProfileID: id,
		AddressID: addressID,
	}, nil
}

func encodePostProfileRequest(ctx context.Context, req *http.Request, request *PostProfileRequest) error {
	// r.Methods("POST").Path("/profiles/")
	req.URL.Path = "/profiles/"
	return encodeRequest(ctx, req, request)
}

func encodeGetProfileRequest(ctx context.Context, req *http.Request, request *GetProfileRequest) error {
	// r.Methods("GET").Path("/profiles/{id}")
	profileID := url.QueryEscape(request.ID)
	req.URL.Path = "/profiles/" + profileID
	return encodeRequest(ctx, req, request)
}

func encodePutProfileRequest(ctx context.Context, req *http.Request, request *PutProfileRequest) error {
	// r.Methods("PUT").Path("/profiles/{id}")
	profileID := url.QueryEscape(request.ID)
	req.URL.Path = "/profiles/" + profileID
	return encodeRequest(ctx, req, request)
}

func encodePatchProfileRequest(ctx context.Context, req *http.Request, request *PatchProfileRequest) error {
	// r.Methods("PATCH").Path("/profiles/{id}")
	profileID := url.QueryEscape(request.ID)
	req.URL.Path = "/profiles/" + profileID
	return encodeRequest(ctx, req, request)
}

func encodeDeleteProfileRequest(ctx context.Context, req *http.Request, request *DeleteProfileRequest) error {
	// r.Methods("DELETE").Path("/profiles/{id}")
	profileID := url.QueryEscape(request.ID)
	req.URL.Path = "/profiles/" + profileID
	return encodeRequest(ctx, req, request)
}

func encodeGetAddressesRequest(ctx context.Context, req *http.Request, request *GetAddressesRequest) error {
	// r.Methods("GET").Path("/profiles/{id}/addresses/")
	profileID := url.QueryEscape(request.ProfileID)
	req.URL.Path = "/profiles/" + profileID + "/addresses/"
	return encodeRequest(ctx, req, request)
}

func encodeGetAddressRequest(ctx context.Context, req *http.Request, request *GetAddressRequest) error {
	// r.Methods("GET").Path("/profiles/{id}/addresses/{addressID}")
	profileID := url.QueryEscape(request.ProfileID)
	addressID := url.QueryEscape(request.AddressID)
	req.URL.Path = "/profiles/" + profileID + "/addresses/" + addressID
	return encodeRequest(ctx, req, request)
}

func encodePostAddressRequest(ctx context.Context, req *http.Request, request *PostAddressRequest) error {
	// r.Methods("POST").Path("/profiles/{id}/addresses/")
	profileID := url.QueryEscape(request.ProfileID)
	req.URL.Path = "/profiles/" + profileID + "/addresses/"
	return encodeRequest(ctx, req, request)
}

func encodeDeleteAddressRequest(ctx context.Context, req *http.Request, request *DeleteAddressRequest) error {
	// r.Methods("DELETE").Path("/profiles/{id}/addresses/{addressID}")
	profileID := url.QueryEscape(request.ProfileID)
	addressID := url.QueryEscape(request.AddressID)
	req.URL.Path = "/profiles/" + profileID + "/addresses/" + addressID
	return encodeRequest(ctx, req, request)
}

func decodePostProfileResponse(_ context.Context, resp *http.Response) (PostProfileResponse, error) {
	var response PostProfileResponse
	err := json.NewDecoder(resp.Body).Decode(&response)
	return response, err
}

func decodeGetProfileResponse(_ context.Context, resp *http.Response) (GetProfileResponse, error) {
	var response GetProfileResponse
	err := json.NewDecoder(resp.Body).Decode(&response)
	return response, err
}

func decodePutProfileResponse(_ context.Context, resp *http.Response) (PutProfileResponse, error) {
	var response PutProfileResponse
	err := json.NewDecoder(resp.Body).Decode(&response)
	return response, err
}

func decodePatchProfileResponse(_ context.Context, resp *http.Response) (PatchProfileResponse, error) {
	var response PatchProfileResponse
	err := json.NewDecoder(resp.Body).Decode(&response)
	return response, err
}

func decodeDeleteProfileResponse(_ context.Context, resp *http.Response) (DeleteProfileResponse, error) {
	var response DeleteProfileResponse
	err := json.NewDecoder(resp.Body).Decode(&response)
	return response, err
}

func decodeGetAddressesResponse(_ context.Context, resp *http.Response) (GetAddressesResponse, error) {
	var response GetAddressesResponse
	err := json.NewDecoder(resp.Body).Decode(&response)
	return response, err
}

func decodeGetAddressResponse(_ context.Context, resp *http.Response) (GetAddressResponse, error) {
	var response GetAddressResponse
	err := json.NewDecoder(resp.Body).Decode(&response)
	return response, err
}

func decodePostAddressResponse(_ context.Context, resp *http.Response) (PostAddressResponse, error) {
	var response PostAddressResponse
	err := json.NewDecoder(resp.Body).Decode(&response)
	return response, err
}

func decodeDeleteAddressResponse(_ context.Context, resp *http.Response) (DeleteAddressResponse, error) {
	var response DeleteAddressResponse
	err := json.NewDecoder(resp.Body).Decode(&response)
	return response, err
}

// errorer is implemented by all concrete response types that may contain
// errors. It allows us to change the HTTP response code without needing to
// trigger an endpoint (transport-level) error. For more information, read the
// big comment in endpoints.go.
type errorer interface {
	error() error
}

// encodeResponse is the common method to encode all response types to the
// client. I chose to do it this way because, since we're using JSON, there's no
// reason to provide anything more specific. It's certainly possible to
// specialize on a per-response (per-method) basis.
func encodeResponse[RES any](ctx context.Context, w http.ResponseWriter, response RES) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}

// encodeRequest likewise JSON-encodes the request to the HTTP request body.
// Don't use it directly as a transport/http.Client EncodeRequestFunc:
// profilesvc endpoints require mutating the HTTP method and request path.
func encodeRequest[REQ any](_ context.Context, req *http.Request, request REQ) error {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(request)
	if err != nil {
		return err
	}
	req.Body = ioutil.NopCloser(&buf)
	return nil
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	if err == nil {
		panic("encodeError with nil error")
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(codeFrom(err))
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": err.Error(),
	})
}

func codeFrom(err error) int {
	switch err {
	case ErrNotFound:
		return http.StatusNotFound
	case ErrAlreadyExists, ErrInconsistentIDs:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func makeServerOption[REQ any, RES any](logger log.Logger) []httptransport.ServerOption[REQ, RES] {
	return []httptransport.ServerOption[REQ, RES]{
		httptransport.ServerErrorHandler[REQ, RES](transport.NewLogErrorHandler(logger)),
	}
}
