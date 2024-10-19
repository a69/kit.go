package tracking

import (
	"context"

	"github.com/a69/kit.go/endpoint"
)

type trackCargoRequest struct {
	ID string
}

type trackCargoResponse struct {
	Cargo *Cargo `json:"cargo,omitempty"`
	Err   error  `json:"error,omitempty"`
}

func (r trackCargoResponse) error() error { return r.Err }

func makeTrackCargoEndpoint(ts Service) endpoint.Endpoint[trackCargoRequest, trackCargoResponse] {
	return func(ctx context.Context, request trackCargoRequest) (trackCargoResponse, error) {
		c, err := ts.Track(request.ID)
		return trackCargoResponse{Cargo: &c, Err: err}, nil
	}
}
