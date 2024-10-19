package awslambda

import (
	"context"
)

// DecodeRequestFunc extracts a user-domain request object from an
// AWS Lambda payload.
type DecodeRequestFunc[REQ any] func(context.Context, []byte) (REQ, error)

// EncodeResponseFunc encodes the passed response object into []byte,
// ready to be sent as AWS Lambda response.
type EncodeResponseFunc[RES any] func(context.Context, RES) ([]byte, error)

// ErrorEncoder is responsible for encoding an error.
type ErrorEncoder func(ctx context.Context, err error) ([]byte, error)
