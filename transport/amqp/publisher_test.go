package amqp_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	amqptransport "github.com/a69/kit.go/transport/amqp"
	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	defaultContentType     = ""
	defaultContentEncoding = ""
)

// TestBadEncode tests if encode errors are handled properly.
func TestBadEncode(t *testing.T) {
	ch := &mockChannel{f: nullFunc}
	q := &amqp.Queue{Name: "some queue"}
	pub := amqptransport.NewPublisher(
		ch,
		q,
		func(context.Context, *amqp.Publishing, struct{}) error { return errors.New("err!") },
		func(context.Context, *amqp.Delivery) (response struct{}, err error) { return struct{}{}, nil },
	)
	errChan := make(chan error, 1)
	var err error
	go func() {
		_, err := pub.Endpoint()(context.Background(), struct{}{})
		errChan <- err

	}()
	select {
	case err = <-errChan:
		break

	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timed out waiting for result")
	}
	if err == nil {
		t.Error("expected error")
	}
	if want, have := "err!", err.Error(); want != have {
		t.Errorf("want %s, have %s", want, have)
	}
}

// TestBadDecode tests if decode errors are handled properly.
func TestBadDecode(t *testing.T) {
	cid := "correlation"
	ch := &mockChannel{
		f: nullFunc,
		c: make(chan amqp.Publishing, 1),
		deliveries: []amqp.Delivery{
			amqp.Delivery{
				CorrelationId: cid,
			},
		},
	}
	q := &amqp.Queue{Name: "some queue"}

	pub := amqptransport.NewPublisher(
		ch,
		q,
		func(context.Context, *amqp.Publishing, struct{}) error { return nil },
		func(context.Context, *amqp.Delivery) (response struct{}, err error) {
			return struct{}{}, errors.New("err!")
		},
		amqptransport.PublisherBefore[struct{}, struct{}](
			amqptransport.SetCorrelationID(cid),
		),
	)

	var err error
	errChan := make(chan error, 1)
	go func() {
		_, err := pub.Endpoint()(context.Background(), struct{}{})
		errChan <- err

	}()

	select {
	case err = <-errChan:
		break

	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timed out waiting for result")
	}

	if err == nil {
		t.Error("expected error")
	}
	if want, have := "err!", err.Error(); want != have {
		t.Errorf("want %s, have %s", want, have)
	}
}

// TestPublisherTimeout ensures that the publisher timeout mechanism works.
func TestPublisherTimeout(t *testing.T) {
	ch := &mockChannel{
		f:          nullFunc,
		c:          make(chan amqp.Publishing, 1),
		deliveries: []amqp.Delivery{}, // no reply from mock subscriber
	}
	q := &amqp.Queue{Name: "some queue"}

	pub := amqptransport.NewPublisher(
		ch,
		q,
		func(context.Context, *amqp.Publishing, struct{}) error { return nil },
		func(context.Context, *amqp.Delivery) (response struct{}, err error) {
			return struct{}{}, nil
		},
		amqptransport.PublisherTimeout[struct{}, struct{}](50*time.Millisecond),
	)

	var err error
	errChan := make(chan error, 1)
	go func() {
		_, err := pub.Endpoint()(context.Background(), struct{}{})
		errChan <- err

	}()

	select {
	case err = <-errChan:
		break

	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for result")
	}

	if err == nil {
		t.Error("expected error")
	}
	if want, have := context.DeadlineExceeded.Error(), err.Error(); want != have {
		t.Errorf("want %s, have %s", want, have)
	}
}

func TestSuccessfulPublisher(t *testing.T) {
	cid := "correlation"
	mockReq := testReq{437}
	mockRes := testRes{
		Squadron: mockReq.Squadron,
		Name:     names[mockReq.Squadron],
	}
	b, err := json.Marshal(mockRes)
	if err != nil {
		t.Fatal(err)
	}
	reqChan := make(chan amqp.Publishing, 1)
	ch := &mockChannel{
		f: nullFunc,
		c: reqChan,
		deliveries: []amqp.Delivery{
			amqp.Delivery{
				CorrelationId: cid,
				Body:          b,
			},
		},
	}
	q := &amqp.Queue{Name: "some queue"}

	pub := amqptransport.NewPublisher(
		ch,
		q,
		testReqEncoder,
		testResDeliveryDecoder,
		amqptransport.PublisherBefore[testReq, testRes](
			amqptransport.SetCorrelationID(cid),
		),
	)
	var publishing amqp.Publishing
	var res testRes
	var ok bool
	resChan := make(chan interface{}, 1)
	errChan := make(chan error, 1)
	go func() {
		res, err := pub.Endpoint()(context.Background(), mockReq)
		if err != nil {
			errChan <- err
		} else {
			resChan <- res
		}
	}()

	select {
	case publishing = <-reqChan:
		break

	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for request")
	}
	if want, have := defaultContentType, publishing.ContentType; want != have {
		t.Errorf("want %s, have %s", want, have)
	}
	if want, have := defaultContentEncoding, publishing.ContentEncoding; want != have {
		t.Errorf("want %s, have %s", want, have)
	}

	select {
	case response := <-resChan:
		res, ok = response.(testRes)
		if !ok {
			t.Error("failed to assert endpoint response type")
		}
		break

	case err = <-errChan:
		break

	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for result")
	}

	if err != nil {
		t.Fatal(err)
	}
	if want, have := mockRes.Name, res.Name; want != have {
		t.Errorf("want %s, have %s", want, have)
	}
}

// TestSendAndForgetPublisher tests that the SendAndForgetDeliverer is working
func TestSendAndForgetPublisher(t *testing.T) {
	ch := &mockChannel{
		f:          nullFunc,
		c:          make(chan amqp.Publishing, 1),
		deliveries: []amqp.Delivery{}, // no reply from mock subscriber
	}
	q := &amqp.Queue{Name: "some queue"}

	pub := amqptransport.NewPublisher(
		ch,
		q,
		func(context.Context, *amqp.Publishing, struct{}) error { return nil },
		func(context.Context, *amqp.Delivery) (response struct{}, err error) {
			return struct{}{}, nil
		},
		amqptransport.PublisherDeliverer[struct{}, struct{}](amqptransport.SendAndForgetDeliverer[struct{}, struct{}]),
		amqptransport.PublisherTimeout[struct{}, struct{}](50*time.Millisecond),
	)

	var err error
	errChan := make(chan error, 1)
	finishChan := make(chan bool, 1)
	go func() {
		_, err := pub.Endpoint()(context.Background(), struct{}{})
		if err != nil {
			errChan <- err
		} else {
			finishChan <- true
		}

	}()

	select {
	case <-finishChan:
		break
	case err = <-errChan:
		t.Errorf("unexpected error %s", err)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for result")
	}

}
