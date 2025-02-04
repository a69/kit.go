// Package client provides a profilesvc client based on a predefined Consul
// service name and relevant tags. Users must only provide the address of a
// Consul server.
package client

import (
	"io"
	"time"

	consulapi "github.com/hashicorp/consul/api"

	"github.com/a69/kit.go/endpoint"
	"github.com/a69/kit.go/examples/profilesvc"
	"github.com/a69/kit.go/log"
	"github.com/a69/kit.go/sd"
	"github.com/a69/kit.go/sd/consul"
	"github.com/a69/kit.go/sd/lb"
)

// New returns a service that's load-balanced over instances of profilesvc found
// in the provided Consul server. The mechanism of looking up profilesvc
// instances in Consul is hard-coded into the client.
func New(consulAddr string, logger log.Logger) (profilesvc.Service, error) {
	apiclient, err := consulapi.NewClient(&consulapi.Config{
		Address: consulAddr,
	})
	if err != nil {
		return nil, err
	}

	// As the implementer of profilesvc, we declare and enforce these
	// parameters for all of the profilesvc consumers.
	var (
		consulService = "profilesvc"
		consulTags    = []string{"prod"}
		passingOnly   = true
		retryMax      = 3
		retryTimeout  = 500 * time.Millisecond
	)

	var (
		sdclient  = consul.NewClient(apiclient)
		instancer = consul.NewInstancer(sdclient, logger, consulService, consulTags, passingOnly)
		endpoints profilesvc.Endpoints
	)
	{
		factory := factoryFor(profilesvc.MakePostProfileEndpoint)
		endpointer := sd.NewEndpointer(instancer, factory, logger)
		balancer := lb.NewRoundRobin[profilesvc.PostProfileRequest, profilesvc.PostProfileResponse](endpointer)
		retry := lb.Retry(retryMax, retryTimeout, balancer)
		endpoints.PostProfileEndpoint = retry
	}
	{
		factory := factoryFor(profilesvc.MakeGetProfileEndpoint)
		endpointer := sd.NewEndpointer(instancer, factory, logger)
		balancer := lb.NewRoundRobin[profilesvc.GetProfileRequest, profilesvc.GetProfileResponse](endpointer)
		retry := lb.Retry(retryMax, retryTimeout, balancer)
		endpoints.GetProfileEndpoint = retry
	}
	{
		factory := factoryFor(profilesvc.MakePutProfileEndpoint)
		endpointer := sd.NewEndpointer(instancer, factory, logger)
		balancer := lb.NewRoundRobin[profilesvc.PutProfileRequest, profilesvc.PutProfileResponse](endpointer)
		retry := lb.Retry(retryMax, retryTimeout, balancer)
		endpoints.PutProfileEndpoint = retry
	}
	{
		factory := factoryFor(profilesvc.MakePatchProfileEndpoint)
		endpointer := sd.NewEndpointer(instancer, factory, logger)
		balancer := lb.NewRoundRobin[profilesvc.PatchProfileRequest, profilesvc.PatchProfileResponse](endpointer)
		retry := lb.Retry(retryMax, retryTimeout, balancer)
		endpoints.PatchProfileEndpoint = retry
	}
	{
		factory := factoryFor(profilesvc.MakeDeleteProfileEndpoint)
		endpointer := sd.NewEndpointer(instancer, factory, logger)
		balancer := lb.NewRoundRobin[profilesvc.DeleteProfileRequest, profilesvc.DeleteProfileResponse](endpointer)
		retry := lb.Retry(retryMax, retryTimeout, balancer)
		endpoints.DeleteProfileEndpoint = retry
	}
	{
		factory := factoryFor(profilesvc.MakeGetAddressesEndpoint)
		endpointer := sd.NewEndpointer(instancer, factory, logger)
		balancer := lb.NewRoundRobin[profilesvc.GetAddressesRequest, profilesvc.GetAddressesResponse](endpointer)
		retry := lb.Retry(retryMax, retryTimeout, balancer)
		endpoints.GetAddressesEndpoint = retry
	}
	{
		factory := factoryFor(profilesvc.MakeGetAddressEndpoint)
		endpointer := sd.NewEndpointer(instancer, factory, logger)
		balancer := lb.NewRoundRobin[profilesvc.GetAddressRequest, profilesvc.GetAddressResponse](endpointer)
		retry := lb.Retry(retryMax, retryTimeout, balancer)
		endpoints.GetAddressEndpoint = retry
	}
	{
		factory := factoryFor(profilesvc.MakePostAddressEndpoint)
		endpointer := sd.NewEndpointer(instancer, factory, logger)
		balancer := lb.NewRoundRobin[profilesvc.PostAddressRequest, profilesvc.PostAddressResponse](endpointer)
		retry := lb.Retry(retryMax, retryTimeout, balancer)
		endpoints.PostAddressEndpoint = retry
	}
	{
		factory := factoryFor(profilesvc.MakeDeleteAddressEndpoint)
		endpointer := sd.NewEndpointer(instancer, factory, logger)
		balancer := lb.NewRoundRobin[profilesvc.DeleteAddressRequest, profilesvc.DeleteAddressResponse](endpointer)
		retry := lb.Retry(retryMax, retryTimeout, balancer)
		endpoints.DeleteAddressEndpoint = retry
	}

	return endpoints, nil
}

func factoryFor[REQ any, RES any](makeEndpoint func(profilesvc.Service) endpoint.Endpoint[REQ, RES]) sd.Factory[REQ, RES] {
	return func(instance string) (endpoint.Endpoint[REQ, RES], io.Closer, error) {
		service, err := profilesvc.MakeClientEndpoints(instance)
		if err != nil {
			return nil, nil, err
		}
		return makeEndpoint(service), nil, nil
	}
}
