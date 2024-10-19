package profilesvc

import (
	"context"
	"net/url"
	"strings"

	"github.com/a69/kit.go/endpoint"
	httptransport "github.com/a69/kit.go/transport/http"
)

// Endpoints collects all of the endpoints that compose a profile service. It's
// meant to be used as a helper struct, to collect all of the endpoints into a
// single parameter.
//
// In a server, it's useful for functions that need to operate on a per-endpoint
// basis. For example, you might pass an Endpoints to a function that produces
// an http.Handler, with each method (endpoint) wired up to a specific path. (It
// is probably a mistake in design to invoke the Service methods on the
// Endpoints struct in a server.)
//
// In a client, it's useful to collect individually constructed endpoints into a
// single type that implements the Service interface. For example, you might
// construct individual endpoints using transport/http.NewClient, combine them
// into an Endpoints, and return it to the caller as a Service.
type Endpoints struct {
	PostProfileEndpoint   endpoint.Endpoint[PostProfileRequest, PostProfileResponse]
	GetProfileEndpoint    endpoint.Endpoint[GetProfileRequest, GetProfileResponse]
	PutProfileEndpoint    endpoint.Endpoint[PutProfileRequest, PutProfileResponse]
	PatchProfileEndpoint  endpoint.Endpoint[PatchProfileRequest, PatchProfileResponse]
	DeleteProfileEndpoint endpoint.Endpoint[DeleteProfileRequest, DeleteProfileResponse]
	GetAddressesEndpoint  endpoint.Endpoint[GetAddressesRequest, GetAddressesResponse]
	GetAddressEndpoint    endpoint.Endpoint[GetAddressRequest, GetAddressResponse]
	PostAddressEndpoint   endpoint.Endpoint[PostAddressRequest, PostAddressResponse]
	DeleteAddressEndpoint endpoint.Endpoint[DeleteAddressRequest, DeleteAddressResponse]
}

// MakeServerEndpoints returns an Endpoints struct where each endpoint invokes
// the corresponding method on the provided service. Useful in a profilesvc
// server.
func MakeServerEndpoints(s Service) Endpoints {
	return Endpoints{
		PostProfileEndpoint:   MakePostProfileEndpoint(s),
		GetProfileEndpoint:    MakeGetProfileEndpoint(s),
		PutProfileEndpoint:    MakePutProfileEndpoint(s),
		PatchProfileEndpoint:  MakePatchProfileEndpoint(s),
		DeleteProfileEndpoint: MakeDeleteProfileEndpoint(s),
		GetAddressesEndpoint:  MakeGetAddressesEndpoint(s),
		GetAddressEndpoint:    MakeGetAddressEndpoint(s),
		PostAddressEndpoint:   MakePostAddressEndpoint(s),
		DeleteAddressEndpoint: MakeDeleteAddressEndpoint(s),
	}
}

// MakeClientEndpoints returns an Endpoints struct where each endpoint invokes
// the corresponding method on the remote instance, via a transport/http.Client.
// Useful in a profilesvc client.
func MakeClientEndpoints(instance string) (Endpoints, error) {
	if !strings.HasPrefix(instance, "http") {
		instance = "http://" + instance
	}
	tgt, err := url.Parse(instance)
	if err != nil {
		return Endpoints{}, err
	}
	tgt.Path = ""

	// Note that the request encoders need to modify the request URL, changing
	// the path. That's fine: we simply need to provide specific encoders for
	// each endpoint.

	return Endpoints{
		PostProfileEndpoint:   httptransport.NewClient[PostProfileRequest, PostProfileResponse]("POST", tgt, encodePostProfileRequest, decodePostProfileResponse, nil).Endpoint(),
		GetProfileEndpoint:    httptransport.NewClient[GetProfileRequest, GetProfileResponse]("GET", tgt, encodeGetProfileRequest, decodeGetProfileResponse, nil).Endpoint(),
		PutProfileEndpoint:    httptransport.NewClient[PutProfileRequest, PutProfileResponse]("PUT", tgt, encodePutProfileRequest, decodePutProfileResponse, nil).Endpoint(),
		PatchProfileEndpoint:  httptransport.NewClient[PatchProfileRequest, PatchProfileResponse]("PATCH", tgt, encodePatchProfileRequest, decodePatchProfileResponse, nil).Endpoint(),
		DeleteProfileEndpoint: httptransport.NewClient[DeleteProfileRequest, DeleteProfileResponse]("DELETE", tgt, encodeDeleteProfileRequest, decodeDeleteProfileResponse, nil).Endpoint(),
		GetAddressesEndpoint:  httptransport.NewClient[GetAddressesRequest, GetAddressesResponse]("GET", tgt, encodeGetAddressesRequest, decodeGetAddressesResponse, nil).Endpoint(),
		GetAddressEndpoint:    httptransport.NewClient[GetAddressRequest, GetAddressResponse]("GET", tgt, encodeGetAddressRequest, decodeGetAddressResponse, nil).Endpoint(),
		PostAddressEndpoint:   httptransport.NewClient[PostAddressRequest, PostAddressResponse]("POST", tgt, encodePostAddressRequest, decodePostAddressResponse, nil).Endpoint(),
		DeleteAddressEndpoint: httptransport.NewClient[DeleteAddressRequest, DeleteAddressResponse]("DELETE", tgt, encodeDeleteAddressRequest, decodeDeleteAddressResponse, nil).Endpoint(),
	}, nil
}

// PostProfile implements Service. Primarily useful in a client.
func (e Endpoints) PostProfile(ctx context.Context, p Profile) error {
	request := PostProfileRequest{Profile: p}
	response, err := e.PostProfileEndpoint(ctx, request)
	if err != nil {
		return err
	}
	resp := response
	return resp.Err
}

// GetProfile implements Service. Primarily useful in a client.
func (e Endpoints) GetProfile(ctx context.Context, id string) (Profile, error) {
	request := GetProfileRequest{ID: id}
	response, err := e.GetProfileEndpoint(ctx, request)
	if err != nil {
		return Profile{}, err
	}
	resp := response
	return resp.Profile, resp.Err
}

// PutProfile implements Service. Primarily useful in a client.
func (e Endpoints) PutProfile(ctx context.Context, id string, p Profile) error {
	request := PutProfileRequest{ID: id, Profile: p}
	response, err := e.PutProfileEndpoint(ctx, request)
	if err != nil {
		return err
	}
	resp := response
	return resp.Err
}

// PatchProfile implements Service. Primarily useful in a client.
func (e Endpoints) PatchProfile(ctx context.Context, id string, p Profile) error {
	request := PatchProfileRequest{ID: id, Profile: p}
	response, err := e.PatchProfileEndpoint(ctx, request)
	if err != nil {
		return err
	}
	resp := response
	return resp.Err
}

// DeleteProfile implements Service. Primarily useful in a client.
func (e Endpoints) DeleteProfile(ctx context.Context, id string) error {
	request := DeleteProfileRequest{ID: id}
	response, err := e.DeleteProfileEndpoint(ctx, request)
	if err != nil {
		return err
	}
	resp := response
	return resp.Err
}

// GetAddresses implements Service. Primarily useful in a client.
func (e Endpoints) GetAddresses(ctx context.Context, profileID string) ([]Address, error) {
	request := GetAddressesRequest{ProfileID: profileID}
	response, err := e.GetAddressesEndpoint(ctx, request)
	if err != nil {
		return nil, err
	}
	resp := response
	return resp.Addresses, resp.Err
}

// GetAddress implements Service. Primarily useful in a client.
func (e Endpoints) GetAddress(ctx context.Context, profileID string, addressID string) (Address, error) {
	request := GetAddressRequest{ProfileID: profileID, AddressID: addressID}
	response, err := e.GetAddressEndpoint(ctx, request)
	if err != nil {
		return Address{}, err
	}
	resp := response
	return resp.Address, resp.Err
}

// PostAddress implements Service. Primarily useful in a client.
func (e Endpoints) PostAddress(ctx context.Context, profileID string, a Address) error {
	request := PostAddressRequest{ProfileID: profileID, Address: a}
	response, err := e.PostAddressEndpoint(ctx, request)
	if err != nil {
		return err
	}
	resp := response
	return resp.Err
}

// DeleteAddress implements Service. Primarily useful in a client.
func (e Endpoints) DeleteAddress(ctx context.Context, profileID string, addressID string) error {
	request := DeleteAddressRequest{ProfileID: profileID, AddressID: addressID}
	response, err := e.DeleteAddressEndpoint(ctx, request)
	if err != nil {
		return err
	}
	resp := response
	return resp.Err
}

// MakePostProfileEndpoint returns an endpoint via the passed service.
// Primarily useful in a server.
func MakePostProfileEndpoint(s Service) endpoint.Endpoint[PostProfileRequest, PostProfileResponse] {
	return func(ctx context.Context, req PostProfileRequest) (response PostProfileResponse, err error) {
		e := s.PostProfile(ctx, req.Profile)
		return PostProfileResponse{Err: e}, nil
	}
}

// MakeGetProfileEndpoint returns an endpoint via the passed service.
// Primarily useful in a server.
func MakeGetProfileEndpoint(s Service) endpoint.Endpoint[GetProfileRequest, GetProfileResponse] {
	return func(ctx context.Context, req GetProfileRequest) (response GetProfileResponse, err error) {
		p, e := s.GetProfile(ctx, req.ID)
		return GetProfileResponse{Profile: p, Err: e}, nil
	}
}

// MakePutProfileEndpoint returns an endpoint via the passed service.
// Primarily useful in a server.
func MakePutProfileEndpoint(s Service) endpoint.Endpoint[PutProfileRequest, PutProfileResponse] {
	return func(ctx context.Context, req PutProfileRequest) (response PutProfileResponse, err error) {
		e := s.PutProfile(ctx, req.ID, req.Profile)
		return PutProfileResponse{Err: e}, nil
	}
}

// MakePatchProfileEndpoint returns an endpoint via the passed service.
// Primarily useful in a server.
func MakePatchProfileEndpoint(s Service) endpoint.Endpoint[PatchProfileRequest, PatchProfileResponse] {
	return func(ctx context.Context, req PatchProfileRequest) (response PatchProfileResponse, err error) {
		e := s.PatchProfile(ctx, req.ID, req.Profile)
		return PatchProfileResponse{Err: e}, nil
	}
}

// MakeDeleteProfileEndpoint returns an endpoint via the passed service.
// Primarily useful in a server.
func MakeDeleteProfileEndpoint(s Service) endpoint.Endpoint[DeleteProfileRequest, DeleteProfileResponse] {
	return func(ctx context.Context, req DeleteProfileRequest) (response DeleteProfileResponse, err error) {
		e := s.DeleteProfile(ctx, req.ID)
		return DeleteProfileResponse{Err: e}, nil
	}
}

// MakeGetAddressesEndpoint returns an endpoint via the passed service.
// Primarily useful in a server.
func MakeGetAddressesEndpoint(s Service) endpoint.Endpoint[GetAddressesRequest, GetAddressesResponse] {
	return func(ctx context.Context, req GetAddressesRequest) (response GetAddressesResponse, err error) {
		a, e := s.GetAddresses(ctx, req.ProfileID)
		return GetAddressesResponse{Addresses: a, Err: e}, nil
	}
}

// MakeGetAddressEndpoint returns an endpoint via the passed service.
// Primarily useful in a server.
func MakeGetAddressEndpoint(s Service) endpoint.Endpoint[GetAddressRequest, GetAddressResponse] {
	return func(ctx context.Context, req GetAddressRequest) (response GetAddressResponse, err error) {
		a, e := s.GetAddress(ctx, req.ProfileID, req.AddressID)
		return GetAddressResponse{Address: a, Err: e}, nil
	}
}

// MakePostAddressEndpoint returns an endpoint via the passed service.
// Primarily useful in a server.
func MakePostAddressEndpoint(s Service) endpoint.Endpoint[PostAddressRequest, PostAddressResponse] {
	return func(ctx context.Context, req PostAddressRequest) (response PostAddressResponse, err error) {
		e := s.PostAddress(ctx, req.ProfileID, req.Address)
		return PostAddressResponse{Err: e}, nil
	}
}

// MakeDeleteAddressEndpoint returns an endpoint via the passed service.
// Primarily useful in a server.
func MakeDeleteAddressEndpoint(s Service) endpoint.Endpoint[DeleteAddressRequest, DeleteAddressResponse] {
	return func(ctx context.Context, req DeleteAddressRequest) (response DeleteAddressResponse, err error) {
		e := s.DeleteAddress(ctx, req.ProfileID, req.AddressID)
		return DeleteAddressResponse{Err: e}, nil
	}
}

// We have two options to return errors from the business logic.
//
// We could return the error via the endpoint itself. That makes certain things
// a little bit easier, like providing non-200 HTTP responses to the client. But
// Go kit assumes that endpoint errors are (or may be treated as)
// transport-domain errors. For example, an endpoint error will count against a
// circuit breaker error count.
//
// Therefore, it's often better to return service (business logic) errors in the
// response object. This means we have to do a bit more work in the HTTP
// response encoder to detect e.g. a not-found error and provide a proper HTTP
// status code. That work is done with the errorer interface, in transport.go.
// Response types that may contain business-logic errors implement that
// interface.

type PostProfileRequest struct {
	Profile Profile
}

type PostProfileResponse struct {
	Err error `json:"err,omitempty"`
}

func (r PostProfileResponse) error() error { return r.Err }

type GetProfileRequest struct {
	ID string
}

type GetProfileResponse struct {
	Profile Profile `json:"profile,omitempty"`
	Err     error   `json:"err,omitempty"`
}

func (r GetProfileResponse) error() error { return r.Err }

type PutProfileRequest struct {
	ID      string
	Profile Profile
}

type PutProfileResponse struct {
	Err error `json:"err,omitempty"`
}

func (r PutProfileResponse) error() error { return nil }

type PatchProfileRequest struct {
	ID      string
	Profile Profile
}

type PatchProfileResponse struct {
	Err error `json:"err,omitempty"`
}

func (r PatchProfileResponse) error() error { return r.Err }

type DeleteProfileRequest struct {
	ID string
}

type DeleteProfileResponse struct {
	Err error `json:"err,omitempty"`
}

func (r DeleteProfileResponse) error() error { return r.Err }

type GetAddressesRequest struct {
	ProfileID string
}

type GetAddressesResponse struct {
	Addresses []Address `json:"addresses,omitempty"`
	Err       error     `json:"err,omitempty"`
}

func (r GetAddressesResponse) error() error { return r.Err }

type GetAddressRequest struct {
	ProfileID string
	AddressID string
}

type GetAddressResponse struct {
	Address Address `json:"address,omitempty"`
	Err     error   `json:"err,omitempty"`
}

func (r GetAddressResponse) error() error { return r.Err }

type PostAddressRequest struct {
	ProfileID string
	Address   Address
}

type PostAddressResponse struct {
	Err error `json:"err,omitempty"`
}

func (r PostAddressResponse) error() error { return r.Err }

type DeleteAddressRequest struct {
	ProfileID string
	AddressID string
}

type DeleteAddressResponse struct {
	Err error `json:"err,omitempty"`
}

func (r DeleteAddressResponse) error() error { return r.Err }
