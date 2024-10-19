package tracking

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/mux"

	kitlog "github.com/a69/kit.go/log"
	kittransport "github.com/a69/kit.go/transport"
	kithttp "github.com/a69/kit.go/transport/http"

	"github.com/a69/kit.go/examples/shipping/cargo"
)

// MakeHandler returns a handler for the tracking service.
func MakeHandler(ts Service, logger kitlog.Logger) http.Handler {
	r := mux.NewRouter()

	opts := []kithttp.ServerOption[trackCargoRequest, trackCargoResponse]{
		kithttp.ServerErrorHandler[trackCargoRequest, trackCargoResponse](kittransport.NewLogErrorHandler(logger)),
		kithttp.ServerErrorEncoder[trackCargoRequest, trackCargoResponse](encodeError),
	}

	trackCargoHandler := kithttp.NewServer(
		makeTrackCargoEndpoint(ts),
		decodeTrackCargoRequest,
		encodeResponse,
		opts...,
	)

	r.Handle("/tracking/v1/cargos/{id}", trackCargoHandler).Methods("GET")

	return r
}

func decodeTrackCargoRequest(_ context.Context, r *http.Request) (trackCargoRequest, error) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		return trackCargoRequest{}, errors.New("bad route")
	}
	return trackCargoRequest{ID: id}, nil
}

func encodeResponse(ctx context.Context, w http.ResponseWriter, response trackCargoResponse) error {

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
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
