package sd

import (
	"io"
	"sort"
	"sync"
	"time"

	"github.com/a69/kit.go/endpoint"
	"github.com/go-kit/log"
)

// endpointCache collects the most recent set of instances from a service discovery
// system, creates endpoints for them using a factory function, and makes
// them available to consumers.
type endpointCache[REQ any, RES any] struct {
	options            endpointerOptions
	mtx                sync.RWMutex
	factory            Factory[REQ, RES]
	cache              map[string]endpointCloser[REQ, RES]
	err                error
	endpoints          []endpoint.Endpoint[REQ, RES]
	logger             log.Logger
	invalidateDeadline time.Time
	timeNow            func() time.Time
}

type endpointCloser[REQ any, RES any] struct {
	endpoint.Endpoint[REQ, RES]
	io.Closer
}

// newEndpointCache returns a new, empty endpointCache.
func newEndpointCache[REQ any, RES any](factory Factory[REQ, RES], logger log.Logger, options endpointerOptions) *endpointCache[REQ, RES] {
	return &endpointCache[REQ, RES]{
		options: options,
		factory: factory,
		cache:   map[string]endpointCloser[REQ, RES]{},
		logger:  logger,
		timeNow: time.Now,
	}
}

// Update should be invoked by clients with a complete set of current instance
// strings whenever that set changes. The cache manufactures new endpoints via
// the factory, closes old endpoints when they disappear, and persists existing
// endpoints if they survive through an update.
func (c *endpointCache[REQ, RES]) Update(event Event) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	// Happy path.
	if event.Err == nil {
		c.updateCache(event.Instances)
		c.err = nil
		return
	}

	// Sad path. Something's gone wrong in sd.
	c.logger.Log("err", event.Err)
	if !c.options.invalidateOnError {
		return // keep returning the last known endpoints on error
	}
	if c.err != nil {
		return // already in the error state, do nothing & keep original error
	}
	c.err = event.Err
	// set new deadline to invalidate Endpoints unless non-error Event is received
	c.invalidateDeadline = c.timeNow().Add(c.options.invalidateTimeout)
	return
}

func (c *endpointCache[REQ, RES]) updateCache(instances []string) {
	// Deterministic order (for later).
	sort.Strings(instances)

	// Produce the current set of services.
	cache := make(map[string]endpointCloser[REQ, RES], len(instances))
	for _, instance := range instances {
		// If it already exists, just copy it over.
		if sc, ok := c.cache[instance]; ok {
			cache[instance] = sc
			delete(c.cache, instance)
			continue
		}

		// If it doesn't exist, create it.
		service, closer, err := c.factory(instance)
		if err != nil {
			c.logger.Log("instance", instance, "err", err)
			continue
		}
		cache[instance] = endpointCloser[REQ, RES]{service, closer}
	}

	// Close any leftover endpoints.
	for _, sc := range c.cache {
		if sc.Closer != nil {
			sc.Closer.Close()
		}
	}

	// Populate the slice of endpoints.
	endpoints := make([]endpoint.Endpoint[REQ, RES], 0, len(cache))
	for _, instance := range instances {
		// A bad factory may mean an instance is not present.
		if _, ok := cache[instance]; !ok {
			continue
		}
		endpoints = append(endpoints, cache[instance].Endpoint)
	}

	// Swap and trigger GC for old copies.
	c.endpoints = endpoints
	c.cache = cache
}

// Endpoints yields the current set of (presumably identical) endpoints, ordered
// lexicographically by the corresponding instance string.
func (c *endpointCache[REQ, RES]) Endpoints() ([]endpoint.Endpoint[REQ, RES], error) {
	// in the steady state we're going to have many goroutines calling Endpoints()
	// concurrently, so to minimize contention we use a shared R-lock.
	c.mtx.RLock()

	if c.err == nil || c.timeNow().Before(c.invalidateDeadline) {
		defer c.mtx.RUnlock()
		return c.endpoints, nil
	}

	c.mtx.RUnlock()

	// in case of an error, switch to an exclusive lock.
	c.mtx.Lock()
	defer c.mtx.Unlock()

	// re-check condition due to a race between RUnlock() and Lock().
	if c.err == nil || c.timeNow().Before(c.invalidateDeadline) {
		return c.endpoints, nil
	}

	c.updateCache(nil) // close any remaining active endpoints
	return nil, c.err
}
