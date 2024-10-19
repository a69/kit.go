package lb

import (
	"math/rand"

	"github.com/a69/kit.go/endpoint"
	"github.com/a69/kit.go/sd"
)

// NewRandom returns a load balancer that selects services randomly.
func NewRandom[REQ any, RES any](s sd.Endpointer[REQ, RES], seed int64) Balancer[REQ, RES] {
	return &random[REQ, RES]{
		s: s,
		r: rand.New(rand.NewSource(seed)),
	}
}

type random[REQ any, RES any] struct {
	s sd.Endpointer[REQ, RES]
	r *rand.Rand
}

func (r *random[REQ, RES]) Endpoint() (endpoint.Endpoint[REQ, RES], error) {
	endpoints, err := r.s.Endpoints()
	if err != nil {
		return nil, err
	}
	if len(endpoints) <= 0 {
		return nil, ErrNoEndpoints
	}
	return endpoints[r.r.Intn(len(endpoints))], nil
}
