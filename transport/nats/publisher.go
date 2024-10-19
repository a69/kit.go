package nats

import (
	"context"
	"encoding/json"
	"github.com/a69/kit.go/endpoint"
	"github.com/nats-io/nats.go"
	"time"
)

// Publisher wraps a URL and provides a method that implements endpoint.Endpoint.
type Publisher[REQ any, RES any] struct {
	publisher *nats.Conn
	subject   string
	enc       EncodeRequestFunc[REQ]
	dec       DecodeResponseFunc[RES]
	before    []RequestFunc
	after     []PublisherResponseFunc
	timeout   time.Duration
}

// NewPublisher constructs a usable Publisher for a single remote method.
func NewPublisher[REQ any, RES any](
	publisher *nats.Conn,
	subject string,
	enc EncodeRequestFunc[REQ],
	dec DecodeResponseFunc[RES],
	options ...PublisherOption[REQ, RES],
) *Publisher[REQ, RES] {
	p := &Publisher[REQ, RES]{
		publisher: publisher,
		subject:   subject,
		enc:       enc,
		dec:       dec,
		timeout:   10 * time.Second,
	}
	for _, option := range options {
		option(p)
	}
	return p
}

// PublisherOption sets an optional parameter for clients.
type PublisherOption[REQ any, RES any] func(*Publisher[REQ, RES])

// PublisherBefore sets the RequestFuncs that are applied to the outgoing NATS
// request before it's invoked.
func PublisherBefore[REQ any, RES any](before ...RequestFunc) PublisherOption[REQ, RES] {
	return func(p *Publisher[REQ, RES]) { p.before = append(p.before, before...) }
}

// PublisherAfter sets the ClientResponseFuncs applied to the incoming NATS
// request prior to it being decoded. This is useful for obtaining anything off
// of the response and adding onto the context prior to decoding.
func PublisherAfter[REQ any, RES any](after ...PublisherResponseFunc) PublisherOption[REQ, RES] {
	return func(p *Publisher[REQ, RES]) { p.after = append(p.after, after...) }
}

// PublisherTimeout sets the available timeout for NATS request.
func PublisherTimeout[REQ any, RES any](timeout time.Duration) PublisherOption[REQ, RES] {
	return func(p *Publisher[REQ, RES]) { p.timeout = timeout }
}

// Endpoint returns a usable endpoint that invokes the remote endpoint.
func (p Publisher[REQ, RES]) Endpoint() endpoint.Endpoint[REQ, RES] {
	return func(ctx context.Context, request REQ) (response RES, err error) {
		ctx, cancel := context.WithTimeout(ctx, p.timeout)
		defer cancel()

		msg := nats.Msg{Subject: p.subject}

		if err = p.enc(ctx, &msg, request); err != nil {
			return
		}

		for _, f := range p.before {
			ctx = f(ctx, &msg)
		}

		resp, err := p.publisher.RequestWithContext(ctx, msg.Subject, msg.Data)
		if err != nil {
			return
		}

		for _, f := range p.after {
			ctx = f(ctx, resp)
		}

		return p.dec(ctx, resp)
	}
}

// EncodeJSONRequest is an EncodeRequestFunc that serializes the request as a
// JSON object to the Data of the Msg. Many JSON-over-NATS services can use it as
// a sensible default.
func EncodeJSONRequest[REQ any](_ context.Context, msg *nats.Msg, request REQ) error {
	b, err := json.Marshal(request)
	if err != nil {
		return err
	}

	msg.Data = b

	return nil
}
