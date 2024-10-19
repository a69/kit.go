package handling

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	kitlog "github.com/a69/kit.go/log"
	"github.com/a69/kit.go/transport"
	kithttp "github.com/a69/kit.go/transport/http"

	"github.com/a69/kit.go/examples/shipping/cargo"
	"github.com/a69/kit.go/examples/shipping/location"
	"github.com/a69/kit.go/examples/shipping/voyage"
)

// MakeHandler returns a handler for the handling service.
func MakeHandler(hs Service, logger kitlog.Logger) http.Handler {
	r := mux.NewRouter()

	opts := []kithttp.ServerOption[registerIncidentRequest, registerIncidentResponse]{
		kithttp.ServerErrorHandler[registerIncidentRequest, registerIncidentResponse](transport.NewLogErrorHandler(logger)),
		kithttp.ServerErrorEncoder[registerIncidentRequest, registerIncidentResponse](encodeError),
	}

	registerIncidentHandler := kithttp.NewServer[registerIncidentRequest, registerIncidentResponse](
		makeRegisterIncidentEndpoint(hs),
		decodeRegisterIncidentRequest,
		encodeResponse,
		opts...,
	)

	r.Handle("/handling/v1/incidents", registerIncidentHandler).Methods("POST")

	return r
}

func decodeRegisterIncidentRequest(_ context.Context, r *http.Request) (registerIncidentRequest, error) {
	var body struct {
		CompletionTime time.Time `json:"completion_time"`
		TrackingID     string    `json:"tracking_id"`
		VoyageNumber   string    `json:"voyage"`
		Location       string    `json:"location"`
		EventType      string    `json:"event_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return registerIncidentRequest{}, err
	}

	return registerIncidentRequest{
		CompletionTime: body.CompletionTime,
		ID:             cargo.TrackingID(body.TrackingID),
		Voyage:         voyage.Number(body.VoyageNumber),
		Location:       location.UNLocode(body.Location),
		EventType:      stringToEventType(body.EventType),
	}, nil
}

func stringToEventType(s string) cargo.HandlingEventType {
	types := map[string]cargo.HandlingEventType{
		cargo.Receive.String(): cargo.Receive,
		cargo.Load.String():    cargo.Load,
		cargo.Unload.String():  cargo.Unload,
		cargo.Customs.String(): cargo.Customs,
		cargo.Claim.String():   cargo.Claim,
	}
	return types[s]
}

func encodeResponse(ctx context.Context, w http.ResponseWriter, response registerIncidentResponse) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}

type errorer interface {
	error() error
}

// encode errors from business-logic
func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	switch err {
	case cargo.ErrUnknown:
		w.WriteHeader(http.StatusNotFound)
	case ErrInvalidArgument:
		w.WriteHeader(http.StatusBadRequest)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": err.Error(),
	})
}
