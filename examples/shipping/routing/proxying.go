package routing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/a69/kit.go/circuitbreaker"
	"github.com/a69/kit.go/endpoint"
	kithttp "github.com/a69/kit.go/transport/http"

	"github.com/a69/kit.go/examples/shipping/cargo"
	"github.com/a69/kit.go/examples/shipping/location"
	"github.com/a69/kit.go/examples/shipping/voyage"
)

type proxyService struct {
	context.Context
	FetchRoutesEndpoint endpoint.Endpoint[fetchRoutesRequest, fetchRoutesResponse]
	Service
}

func (s proxyService) FetchRoutesForSpecification(rs cargo.RouteSpecification) []cargo.Itinerary {
	response, err := s.FetchRoutesEndpoint(s.Context, fetchRoutesRequest{
		From: string(rs.Origin),
		To:   string(rs.Destination),
	})
	if err != nil {
		return []cargo.Itinerary{}
	}

	var itineraries []cargo.Itinerary
	for _, r := range response.Paths {
		var legs []cargo.Leg
		for _, e := range r.Edges {
			legs = append(legs, cargo.Leg{
				VoyageNumber:   voyage.Number(e.Voyage),
				LoadLocation:   location.UNLocode(e.Origin),
				UnloadLocation: location.UNLocode(e.Destination),
				LoadTime:       e.Departure,
				UnloadTime:     e.Arrival,
			})
		}

		itineraries = append(itineraries, cargo.Itinerary{Legs: legs})
	}

	return itineraries
}

// ServiceMiddleware defines a middleware for a routing service.
type ServiceMiddleware func(Service) Service

// NewProxyingMiddleware returns a new instance of a proxying middleware.
func NewProxyingMiddleware(ctx context.Context, proxyURL string) ServiceMiddleware {
	return func(next Service) Service {
		var e endpoint.Endpoint[fetchRoutesRequest, fetchRoutesResponse]
		e = makeFetchRoutesEndpoint(ctx, proxyURL)
		e = circuitbreaker.Hystrix[fetchRoutesRequest, fetchRoutesResponse]("fetch-routes")(e)
		return proxyService{ctx, e, next}
	}
}

type fetchRoutesRequest struct {
	From string
	To   string
}

type fetchRoutesResponse struct {
	Paths []struct {
		Edges []struct {
			Origin      string    `json:"origin"`
			Destination string    `json:"destination"`
			Voyage      string    `json:"voyage"`
			Departure   time.Time `json:"departure"`
			Arrival     time.Time `json:"arrival"`
		} `json:"edges"`
	} `json:"paths"`
}

func makeFetchRoutesEndpoint(ctx context.Context, instance string) endpoint.Endpoint[fetchRoutesRequest, fetchRoutesResponse] {
	u, err := url.Parse(instance)
	if err != nil {
		panic(err)
	}
	if u.Path == "" {
		u.Path = "/paths"
	}
	return kithttp.NewClient[fetchRoutesRequest, fetchRoutesResponse](
		"GET", u,
		encodeFetchRoutesRequest,
		decodeFetchRoutesResponse,
	).Endpoint()
}

func decodeFetchRoutesResponse(_ context.Context, resp *http.Response) (fetchRoutesResponse, error) {
	var response fetchRoutesResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fetchRoutesResponse{}, err
	}
	return response, nil
}

func encodeFetchRoutesRequest(_ context.Context, r *http.Request, request *fetchRoutesRequest) error {

	vals := r.URL.Query()
	vals.Add("from", request.From)
	vals.Add("to", request.To)
	r.URL.RawQuery = vals.Encode()

	return nil
}
