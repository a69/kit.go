package booking

import (
	"context"
	"time"

	"github.com/a69/kit.go/endpoint"

	"github.com/a69/kit.go/examples/shipping/cargo"
	"github.com/a69/kit.go/examples/shipping/location"
)

type bookCargoRequest struct {
	Origin          location.UNLocode
	Destination     location.UNLocode
	ArrivalDeadline time.Time
}

type bookCargoResponse struct {
	ID  cargo.TrackingID `json:"tracking_id,omitempty"`
	Err error            `json:"error,omitempty"`
}

func (r bookCargoResponse) error() error { return r.Err }

func makeBookCargoEndpoint(s Service) endpoint.Endpoint[bookCargoRequest, bookCargoResponse] {
	return func(ctx context.Context, request bookCargoRequest) (bookCargoResponse, error) {
		id, err := s.BookNewCargo(request.Origin, request.Destination, request.ArrivalDeadline)
		return bookCargoResponse{ID: id, Err: err}, nil
	}
}

type loadCargoRequest struct {
	ID cargo.TrackingID
}

type loadCargoResponse struct {
	Cargo *Cargo `json:"cargo,omitempty"`
	Err   error  `json:"error,omitempty"`
}

func (r loadCargoResponse) error() error { return r.Err }

func makeLoadCargoEndpoint(s Service) endpoint.Endpoint[loadCargoRequest, loadCargoResponse] {
	return func(ctx context.Context, request loadCargoRequest) (loadCargoResponse, error) {
		c, err := s.LoadCargo(request.ID)
		return loadCargoResponse{Cargo: &c, Err: err}, nil
	}
}

type requestRoutesRequest struct {
	ID cargo.TrackingID
}

type requestRoutesResponse struct {
	Routes []cargo.Itinerary `json:"routes,omitempty"`
	Err    error             `json:"error,omitempty"`
}

func (r requestRoutesResponse) error() error { return r.Err }

func makeRequestRoutesEndpoint(s Service) endpoint.Endpoint[requestRoutesRequest, requestRoutesResponse] {
	return func(ctx context.Context, request requestRoutesRequest) (requestRoutesResponse, error) {

		itin := s.RequestPossibleRoutesForCargo(request.ID)
		return requestRoutesResponse{Routes: itin, Err: nil}, nil
	}
}

type assignToRouteRequest struct {
	ID        cargo.TrackingID
	Itinerary cargo.Itinerary
}

type assignToRouteResponse struct {
	Err error `json:"error,omitempty"`
}

func (r assignToRouteResponse) error() error { return r.Err }

func makeAssignToRouteEndpoint(s Service) endpoint.Endpoint[assignToRouteRequest, assignToRouteResponse] {
	return func(ctx context.Context, request assignToRouteRequest) (assignToRouteResponse, error) {
		err := s.AssignCargoToRoute(request.ID, request.Itinerary)
		return assignToRouteResponse{Err: err}, nil
	}
}

type changeDestinationRequest struct {
	ID          cargo.TrackingID
	Destination location.UNLocode
}

type changeDestinationResponse struct {
	Err error `json:"error,omitempty"`
}

func (r changeDestinationResponse) error() error { return r.Err }

func makeChangeDestinationEndpoint(s Service) endpoint.Endpoint[changeDestinationRequest, changeDestinationResponse] {
	return func(ctx context.Context, request changeDestinationRequest) (changeDestinationResponse, error) {
		err := s.ChangeDestination(request.ID, request.Destination)
		return changeDestinationResponse{Err: err}, nil
	}
}

type listCargosRequest struct{}

type listCargosResponse struct {
	Cargos []Cargo `json:"cargos,omitempty"`
	Err    error   `json:"error,omitempty"`
}

func (r listCargosResponse) error() error { return r.Err }

func makeListCargosEndpoint(s Service) endpoint.Endpoint[listCargosRequest, listCargosResponse] {
	return func(ctx context.Context, request listCargosRequest) (listCargosResponse, error) {
		return listCargosResponse{Cargos: s.Cargos(), Err: nil}, nil
	}
}

type listLocationsRequest struct {
}

type listLocationsResponse struct {
	Locations []Location `json:"locations,omitempty"`
	Err       error      `json:"error,omitempty"`
}

func makeListLocationsEndpoint(s Service) endpoint.Endpoint[listLocationsRequest, listLocationsResponse] {
	return func(ctx context.Context, request listLocationsRequest) (listLocationsResponse, error) {
		return listLocationsResponse{Locations: s.Locations(), Err: nil}, nil
	}
}
