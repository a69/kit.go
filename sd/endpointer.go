package sd

import (
	"time"

	"github.com/a69/kit.go/endpoint"
	"github.com/go-kit/log"
)

// Endpointer listens to a service discovery system and yields a set of
// identical endpoints on demand. An error indicates a problem with connectivity
// to the service discovery system, or within the system itself; an Endpointer
// may yield no endpoints without error.
type Endpointer[REQ any, RES any] interface {
	Endpoints() ([]endpoint.Endpoint[REQ, RES], error)
}

// FixedEndpointer yields a fixed set of endpoints.
type FixedEndpointer[REQ any, RES any] []endpoint.Endpoint[REQ, RES]

// Endpoints implements Endpointer.
func (s FixedEndpointer[REQ, RES]) Endpoints() ([]endpoint.Endpoint[REQ, RES], error) { return s, nil }

// NewEndpointer creates an Endpointer that subscribes to updates from Instancer src
// and uses factory f to create Endpoints. If src notifies of an error, the Endpointer
// keeps returning previously created Endpoints assuming they are still good, unless
// this behavior is disabled via InvalidateOnError option.
func NewEndpointer[REQ any, RES any](src Instancer, f Factory[REQ, RES], logger log.Logger, options ...EndpointerOption) *DefaultEndpointer[REQ, RES] {
	opts := endpointerOptions{}
	for _, opt := range options {
		opt(&opts)
	}
	se := &DefaultEndpointer[REQ, RES]{
		cache:     newEndpointCache(f, logger, opts),
		instancer: src,
		ch:        make(chan Event),
	}
	go se.receive()
	src.Register(se.ch)
	return se
}

// EndpointerOption allows control of endpointCache behavior.
type EndpointerOption func(*endpointerOptions)

// InvalidateOnError returns EndpointerOption that controls how the Endpointer
// behaves when then Instancer publishes an Event containing an error.
// Without this option the Endpointer continues returning the last known
// endpoints. With this option, the Endpointer continues returning the last
// known endpoints until the timeout elapses, then closes all active endpoints
// and starts returning an error. Once the Instancer sends a new update with
// valid resource instances, the normal operation is resumed.
func InvalidateOnError(timeout time.Duration) EndpointerOption {
	return func(opts *endpointerOptions) {
		opts.invalidateOnError = true
		opts.invalidateTimeout = timeout
	}
}

type endpointerOptions struct {
	invalidateOnError bool
	invalidateTimeout time.Duration
}

// DefaultEndpointer implements an Endpointer interface.
// When created with NewEndpointer function, it automatically registers
// as a subscriber to events from the Instances and maintains a list
// of active Endpoints.
type DefaultEndpointer[REQ any, RES any] struct {
	cache     *endpointCache[REQ, RES]
	instancer Instancer
	ch        chan Event
}

func (de *DefaultEndpointer[_, _]) receive() {
	for event := range de.ch {
		de.cache.Update(event)
	}
}

// Close deregisters DefaultEndpointer from the Instancer and stops the internal go-routine.
func (de *DefaultEndpointer[_, _]) Close() {
	de.instancer.Deregister(de.ch)
	close(de.ch)
}

// Endpoints implements Endpointer.
func (de *DefaultEndpointer[REQ, RES]) Endpoints() ([]endpoint.Endpoint[REQ, RES], error) {
	return de.cache.Endpoints()
}
