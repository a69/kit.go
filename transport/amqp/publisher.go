package amqp

import (
	"context"
	"time"

	"github.com/a69/kit.go/endpoint"
	amqp "github.com/rabbitmq/amqp091-go"
)

// The golang AMQP implementation requires the []byte representation of
// correlation id strings to have a maximum length of 255 bytes.
const maxCorrelationIdLength = 255

// Publisher wraps an AMQP channel and queue, and provides a method that
// implements endpoint.Endpoint.
type Publisher[REQ any, RES any] struct {
	ch        Channel
	q         *amqp.Queue
	enc       EncodeRequestFunc[REQ]
	dec       DecodeResponseFunc[RES]
	before    []RequestFunc
	after     []PublisherResponseFunc
	deliverer Deliverer[REQ, RES]
	timeout   time.Duration
}

// NewPublisher constructs a usable Publisher for a single remote method.
func NewPublisher[REQ any, RES any](
	ch Channel,
	q *amqp.Queue,
	enc EncodeRequestFunc[REQ],
	dec DecodeResponseFunc[RES],
	options ...PublisherOption[REQ, RES],
) *Publisher[REQ, RES] {
	p := &Publisher[REQ, RES]{
		ch:        ch,
		q:         q,
		enc:       enc,
		dec:       dec,
		deliverer: DefaultDeliverer[REQ, RES],
		timeout:   10 * time.Second,
	}
	for _, option := range options {
		option(p)
	}
	return p
}

// PublisherOption sets an optional parameter for clients.
type PublisherOption[REQ any, RES any] func(*Publisher[REQ, RES])

// PublisherBefore sets the RequestFuncs that are applied to the outgoing AMQP
// request before it's invoked.
func PublisherBefore[REQ any, RES any](before ...RequestFunc) PublisherOption[REQ, RES] {
	return func(p *Publisher[REQ, RES]) { p.before = append(p.before, before...) }
}

// PublisherAfter sets the ClientResponseFuncs applied to the incoming AMQP
// request prior to it being decoded. This is useful for obtaining anything off
// of the response and adding onto the context prior to decoding.
func PublisherAfter[REQ any, RES any](after ...PublisherResponseFunc) PublisherOption[REQ, RES] {
	return func(p *Publisher[REQ, RES]) { p.after = append(p.after, after...) }
}

// PublisherDeliverer sets the deliverer function that the Publisher invokes.
func PublisherDeliverer[REQ any, RES any](deliverer Deliverer[REQ, RES]) PublisherOption[REQ, RES] {
	return func(p *Publisher[REQ, RES]) { p.deliverer = deliverer }
}

// PublisherTimeout sets the available timeout for an AMQP request.
func PublisherTimeout[REQ any, RES any](timeout time.Duration) PublisherOption[REQ, RES] {
	return func(p *Publisher[REQ, RES]) { p.timeout = timeout }
}

// Endpoint returns a usable endpoint that invokes the remote endpoint.
func (p *Publisher[REQ, RES]) Endpoint() endpoint.Endpoint[REQ, RES] {
	return func(ctx context.Context, request REQ) (res RES, err error) {
		ctx, cancel := context.WithTimeout(ctx, p.timeout)
		defer cancel()

		pub := amqp.Publishing{
			ReplyTo:       p.q.Name,
			CorrelationId: randomString(randInt(5, maxCorrelationIdLength)),
		}

		if err = p.enc(ctx, &pub, request); err != nil {
			return
		}

		for _, f := range p.before {
			// Affect only amqp.Publishing
			ctx = f(ctx, &pub, nil)
		}

		deliv, err := p.deliverer(ctx, *p, &pub)
		if err != nil {
			return
		}

		for _, f := range p.after {
			ctx = f(ctx, deliv)
		}
		response, err := p.dec(ctx, deliv)
		if err != nil {
			return
		}

		return response, nil
	}
}

// Deliverer is invoked by the Publisher to publish the specified Publishing, and to
// retrieve the appropriate response Delivery object.
type Deliverer[REQ any, RES any] func(
	context.Context,
	Publisher[REQ, RES],
	*amqp.Publishing,
) (*amqp.Delivery, error)

// DefaultDeliverer is a deliverer that publishes the specified Publishing
// and returns the first Delivery object with the matching correlationId.
// If the context times out while waiting for a reply, an error will be returned.
func DefaultDeliverer[REQ any, RES any](
	ctx context.Context,
	p Publisher[REQ, RES],
	pub *amqp.Publishing,
) (*amqp.Delivery, error) {
	err := p.ch.Publish(
		getPublishExchange(ctx),
		getPublishKey(ctx),
		false, //mandatory
		false, //immediate
		*pub,
	)
	if err != nil {
		return nil, err
	}
	autoAck := getConsumeAutoAck(ctx)

	msg, err := p.ch.Consume(
		p.q.Name,
		"", //consumer
		autoAck,
		false, //exclusive
		false, //noLocal
		false, //noWait
		getConsumeArgs(ctx),
	)
	if err != nil {
		return nil, err
	}

	for {
		select {
		case d := <-msg:
			if d.CorrelationId == pub.CorrelationId {
				if !autoAck {
					d.Ack(false) //multiple
				}
				return &d, nil
			}

		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

}

// SendAndForgetDeliverer delivers the supplied publishing and
// returns a nil response.
// When using this deliverer please ensure that the supplied DecodeResponseFunc and
// PublisherResponseFunc are able to handle nil-type responses.
func SendAndForgetDeliverer[REQ any, RES any](
	ctx context.Context,
	p Publisher[REQ, RES],
	pub *amqp.Publishing,
) (*amqp.Delivery, error) {
	err := p.ch.Publish(
		getPublishExchange(ctx),
		getPublishKey(ctx),
		false, //mandatory
		false, //immediate
		*pub,
	)
	return nil, err
}
