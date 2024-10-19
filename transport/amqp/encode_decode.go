package amqp

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

// DecodeRequestFunc extracts a user-domain request object from
// an AMQP Delivery object. It is designed to be used in AMQP Subscribers.
type DecodeRequestFunc[REQ any] func(context.Context, *amqp.Delivery) (request REQ, err error)

// EncodeRequestFunc encodes the passed request object into
// an AMQP Publishing object. It is designed to be used in AMQP Publishers.
type EncodeRequestFunc[REQ any] func(context.Context, *amqp.Publishing, REQ) error

// EncodeResponseFunc encodes the passed response object to
// an AMQP Publishing object. It is designed to be used in AMQP Subscribers.
type EncodeResponseFunc[RES any] func(context.Context, *amqp.Publishing, RES) error

// DecodeResponseFunc extracts a user-domain response object from
// an AMQP Delivery object. It is designed to be used in AMQP Publishers.
type DecodeResponseFunc[RES any] func(context.Context, *amqp.Delivery) (response RES, err error)
