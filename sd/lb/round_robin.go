package lb

import (
	"sync/atomic"

	"github.com/a69/kit.go/endpoint"
	"github.com/a69/kit.go/sd"
)

// NewRoundRobin returns a load balancer that returns services in sequence.
func NewRoundRobin[REQ any, RES any](s sd.Endpointer[REQ, RES]) Balancer[REQ, RES] {
	return &roundRobin[REQ, RES]{
		s: s,
		c: 0,
	}
}

type roundRobin[REQ any, RES any] struct {
	s sd.Endpointer[REQ, RES]
	c uint64
}

func (rr *roundRobin[REQ, RES]) Endpoint() (endpoint.Endpoint[REQ, RES], error) {
	endpoints, err := rr.s.Endpoints()
	if err != nil {
		return nil, err
	}
	if len(endpoints) <= 0 {
		return nil, ErrNoEndpoints
	}
	old := atomic.AddUint64(&rr.c, 1) - 1
	idx := old % uint64(len(endpoints))
	return endpoints[idx], nil
}
