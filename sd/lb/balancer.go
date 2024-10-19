package lb

import (
	"errors"

	"github.com/a69/kit.go/endpoint"
)

// Balancer yields endpoints according to some heuristic.
type Balancer[REQ any, RES any] interface {
	Endpoint() (endpoint.Endpoint[REQ, RES], error)
}

// ErrNoEndpoints is returned when no qualifying endpoints are available.
var ErrNoEndpoints = errors.New("no endpoints available")
