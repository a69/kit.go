package handling

import (
	"context"
	"time"

	"github.com/a69/kit.go/endpoint"

	"github.com/a69/kit.go/examples/shipping/cargo"
	"github.com/a69/kit.go/examples/shipping/location"
	"github.com/a69/kit.go/examples/shipping/voyage"
)

type registerIncidentRequest struct {
	ID             cargo.TrackingID
	Location       location.UNLocode
	Voyage         voyage.Number
	EventType      cargo.HandlingEventType
	CompletionTime time.Time
}

type registerIncidentResponse struct {
	Err error `json:"error,omitempty"`
}

func (r registerIncidentResponse) error() error { return r.Err }

func makeRegisterIncidentEndpoint(hs Service) endpoint.Endpoint[registerIncidentRequest, registerIncidentResponse] {
	return func(ctx context.Context, request registerIncidentRequest) (registerIncidentResponse, error) {

		err := hs.RegisterHandlingEvent(request.CompletionTime, request.ID, request.Voyage, request.Location, request.EventType)
		return registerIncidentResponse{Err: err}, nil
	}
}
