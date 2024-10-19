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
	errTypeAssertion = errors.New("type assertion error")
)

// mockChannel is a mock of *amqp.Channel.
type mockChannel struct {
	f          func(exchange, key string, mandatory, immediate bool)
	c          chan<- amqp.Publishing
	deliveries []amqp.Delivery
}

// Publish runs a test function f and sends resultant message to a channel.
func (ch *mockChannel) Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	ch.f(exchange, key, mandatory, immediate)
	ch.c <- msg
	return nil
}

var nullFunc = func(exchange, key string, mandatory, immediate bool) {
}

func (ch *mockChannel) Consume(queue, consumer string, autoAck, exclusive, noLocal, noWail bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	c := make(chan amqp.Delivery, len(ch.deliveries))
	for _, d := range ch.deliveries {
		c <- d
	}
	return c, nil
}

// TestSubscriberBadDecode checks if decoder errors are handled properly.
func TestSubscriberBadDecode(t *testing.T) {
	sub := amqptransport.NewSubscriber[struct{}, struct{}](
		func(context.Context, struct{}) (struct{}, error) { return struct{}{}, nil },
		func(context.Context, *amqp.Delivery) (struct{}, error) { return struct{}{}, errors.New("err!") },
		func(context.Context, *amqp.Publishing, struct{}) error {
			return nil
		},
		amqptransport.SubscriberErrorEncoder[struct{}, struct{}](amqptransport.ReplyErrorEncoder),
	)

	outputChan := make(chan amqp.Publishing, 1)
	ch := &mockChannel{f: nullFunc, c: outputChan}
	sub.ServeDelivery(ch)(&amqp.Delivery{})

	var msg amqp.Publishing
	select {
	case msg = <-outputChan:
		break

	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timed out waiting for publishing")
	}
	res, err := decodeSubscriberError(msg)
	if err != nil {
		t.Fatal(err)
	}
	if want, have := "err!", res.Error; want != have {
		t.Errorf("want %s, have %s", want, have)
	}
}

// TestSubscriberBadEndpoint checks if endpoint errors are handled properly.
func TestSubscriberBadEndpoint(t *testing.T) {
	sub := amqptransport.NewSubscriber[struct{}, struct{}](
		func(context.Context, struct{}) (struct{}, error) { return struct{}{}, errors.New("err!") },
		func(context.Context, *amqp.Delivery) (struct{}, error) { return struct{}{}, nil },
		func(context.Context, *amqp.Publishing, struct{}) error {
			return nil
		},
		amqptransport.SubscriberErrorEncoder[struct{}, struct{}](amqptransport.ReplyErrorEncoder),
	)

	outputChan := make(chan amqp.Publishing, 1)
	ch := &mockChannel{f: nullFunc, c: outputChan}
	sub.ServeDelivery(ch)(&amqp.Delivery{})

	var msg amqp.Publishing

	select {
	case msg = <-outputChan:
		break

	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timed out waiting for publishing")
	}

	res, err := decodeSubscriberError(msg)
	if err != nil {
		t.Fatal(err)
	}
	if want, have := "err!", res.Error; want != have {
		t.Errorf("want %s, have %s", want, have)
	}
}

// TestSubscriberBadEncoder checks if encoder errors are handled properly.
func TestSubscriberBadEncoder(t *testing.T) {
	sub := amqptransport.NewSubscriber(
		func(context.Context, struct{}) (struct{}, error) { return struct{}{}, nil },
		func(context.Context, *amqp.Delivery) (struct{}, error) { return struct{}{}, nil },
		func(context.Context, *amqp.Publishing, struct{}) error {
			return errors.New("err!")
		},
		amqptransport.SubscriberErrorEncoder[struct{}, struct{}](amqptransport.ReplyErrorEncoder),
	)

	outputChan := make(chan amqp.Publishing, 1)
	ch := &mockChannel{f: nullFunc, c: outputChan}
	sub.ServeDelivery(ch)(&amqp.Delivery{})

	var msg amqp.Publishing

	select {
	case msg = <-outputChan:
		break

	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timed out waiting for publishing")
	}

	res, err := decodeSubscriberError(msg)
	if err != nil {
		t.Fatal(err)
	}
	if want, have := "err!", res.Error; want != have {
		t.Errorf("want %s, have %s", want, have)
	}
}

// TestSubscriberSuccess checks if CorrelationId and ReplyTo are set properly
// and if the payload is encoded properly.
func TestSubscriberSuccess(t *testing.T) {
	cid := "correlation"
	replyTo := "sender"
	obj := testReq{
		Squadron: 436,
	}
	b, err := json.Marshal(obj)
	if err != nil {
		t.Fatal(err)
	}

	sub := amqptransport.NewSubscriber[testReq, testRes](
		testEndpoint,
		testReqDecoder,
		amqptransport.GenericEncodeJSONResponse[testRes],
		amqptransport.SubscriberErrorEncoder[testReq, testRes](amqptransport.ReplyErrorEncoder),
	)

	checkReplyToFunc := func(exchange, key string, mandatory, immediate bool) {
		if want, have := replyTo, key; want != have {
			t.Errorf("want %s, have %s", want, have)
		}
	}

	outputChan := make(chan amqp.Publishing, 1)
	ch := &mockChannel{f: checkReplyToFunc, c: outputChan}
	sub.ServeDelivery(ch)(&amqp.Delivery{
		CorrelationId: cid,
		ReplyTo:       replyTo,
		Body:          b,
	})

	var msg amqp.Publishing

	select {
	case msg = <-outputChan:
		break

	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timed out waiting for publishing")
	}

	if want, have := cid, msg.CorrelationId; want != have {
		t.Errorf("want %s, have %s", want, have)
	}

	// check if error is not thrown
	errRes, err := decodeSubscriberError(msg)
	if err != nil {
		t.Fatal(err)
	}
	if errRes.Error != "" {
		t.Error("Received error from subscriber", errRes.Error)
		return
	}

	// check obj vals
	response, err := testResDecoder(msg.Body)
	if err != nil {
		t.Fatal(err)
	}

	if want, have := obj.Squadron, response.Squadron; want != have {
		t.Errorf("want %d, have %d", want, have)
	}
	if want, have := names[obj.Squadron], response.Name; want != have {
		t.Errorf("want %s, have %s", want, have)
	}
}

// TestNopResponseSubscriber checks if setting responsePublisher to
// NopResponsePublisher works properly by disabling response.
func TestNopResponseSubscriber(t *testing.T) {
	cid := "correlation"
	replyTo := "sender"
	obj := testReq{
		Squadron: 436,
	}
	b, err := json.Marshal(obj)
	if err != nil {
		t.Fatal(err)
	}

	sub := amqptransport.NewSubscriber(
		testEndpoint,
		testReqDecoder,
		amqptransport.GenericEncodeJSONResponse[testRes],
		amqptransport.SubscriberResponsePublisher[testReq, testRes](amqptransport.NopResponsePublisher),
		amqptransport.SubscriberErrorEncoder[testReq, testRes](amqptransport.ReplyErrorEncoder),
	)

	checkReplyToFunc := func(exchange, key string, mandatory, immediate bool) {}

	outputChan := make(chan amqp.Publishing, 1)
	ch := &mockChannel{f: checkReplyToFunc, c: outputChan}
	sub.ServeDelivery(ch)(&amqp.Delivery{
		CorrelationId: cid,
		ReplyTo:       replyTo,
		Body:          b,
	})

	select {
	case <-outputChan:
		t.Fatal("Subscriber with NopResponsePublisher replied.")
	case <-time.After(100 * time.Millisecond):
		break
	}
}

// TestSubscriberMultipleBefore checks if options to set exchange, key, deliveryMode
// are working.
func TestSubscriberMultipleBefore(t *testing.T) {
	exchange := "some exchange"
	key := "some key"
	deliveryMode := uint8(127)
	contentType := "some content type"
	contentEncoding := "some content encoding"
	sub := amqptransport.NewSubscriber(
		func(context.Context, struct{}) (struct{}, error) { return struct{}{}, nil },
		func(context.Context, *amqp.Delivery) (struct{}, error) { return struct{}{}, nil },
		amqptransport.GenericEncodeJSONResponse[struct{}],
		amqptransport.SubscriberErrorEncoder[struct{}, struct{}](amqptransport.ReplyErrorEncoder),
		amqptransport.SubscriberBefore[struct{}, struct{}](
			amqptransport.SetPublishExchange(exchange),
			amqptransport.SetPublishKey(key),
			amqptransport.SetPublishDeliveryMode(deliveryMode),
			amqptransport.SetContentType(contentType),
			amqptransport.SetContentEncoding(contentEncoding),
		),
	)
	checkReplyToFunc := func(exch, k string, mandatory, immediate bool) {
		if want, have := exchange, exch; want != have {
			t.Errorf("want %s, have %s", want, have)
		}
		if want, have := key, k; want != have {
			t.Errorf("want %s, have %s", want, have)
		}
	}

	outputChan := make(chan amqp.Publishing, 1)
	ch := &mockChannel{f: checkReplyToFunc, c: outputChan}
	sub.ServeDelivery(ch)(&amqp.Delivery{})

	var msg amqp.Publishing

	select {
	case msg = <-outputChan:
		break

	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timed out waiting for publishing")
	}

	// check if error is not thrown
	errRes, err := decodeSubscriberError(msg)
	if err != nil {
		t.Fatal(err)
	}
	if errRes.Error != "" {
		t.Error("Received error from subscriber", errRes.Error)
		return
	}

	if want, have := contentType, msg.ContentType; want != have {
		t.Errorf("want %s, have %s", want, have)
	}

	if want, have := contentEncoding, msg.ContentEncoding; want != have {
		t.Errorf("want %s, have %s", want, have)
	}

	if want, have := deliveryMode, msg.DeliveryMode; want != have {
		t.Errorf("want %d, have %d", want, have)
	}
}

// TestDefaultContentMetaData checks that default ContentType and Content-Encoding
// is not set as mentioned by AMQP specification.
func TestDefaultContentMetaData(t *testing.T) {
	defaultContentType := ""
	defaultContentEncoding := ""
	sub := amqptransport.NewSubscriber(
		func(context.Context, struct{}) (struct{}, error) { return struct{}{}, nil },
		func(context.Context, *amqp.Delivery) (struct{}, error) { return struct{}{}, nil },
		amqptransport.GenericEncodeJSONResponse[struct{}],
		amqptransport.SubscriberErrorEncoder[struct{}, struct{}](amqptransport.ReplyErrorEncoder),
	)
	checkReplyToFunc := func(exch, k string, mandatory, immediate bool) {}
	outputChan := make(chan amqp.Publishing, 1)
	ch := &mockChannel{f: checkReplyToFunc, c: outputChan}
	sub.ServeDelivery(ch)(&amqp.Delivery{})

	var msg amqp.Publishing

	select {
	case msg = <-outputChan:
		break

	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timed out waiting for publishing")
	}

	// check if error is not thrown
	errRes, err := decodeSubscriberError(msg)
	if err != nil {
		t.Fatal(err)
	}
	if errRes.Error != "" {
		t.Error("Received error from subscriber", errRes.Error)
		return
	}

	if want, have := defaultContentType, msg.ContentType; want != have {
		t.Errorf("want %s, have %s", want, have)
	}
	if want, have := defaultContentEncoding, msg.ContentEncoding; want != have {
		t.Errorf("want %s, have %s", want, have)
	}
}

func decodeSubscriberError(pub amqp.Publishing) (amqptransport.DefaultErrorResponse, error) {
	var res amqptransport.DefaultErrorResponse
	err := json.Unmarshal(pub.Body, &res)
	return res, err
}

type testReq struct {
	Squadron int `json:"s"`
}
type testRes struct {
	Squadron int    `json:"s"`
	Name     string `json:"n"`
}

func testEndpoint(_ context.Context, request testReq) (testRes, error) {
	req := request
	name, prs := names[req.Squadron]
	if !prs {
		return testRes{}, errors.New("unknown squadron name")
	}
	res := testRes{
		Squadron: req.Squadron,
		Name:     name,
	}
	return res, nil
}

func testReqDecoder(_ context.Context, d *amqp.Delivery) (testReq, error) {
	var obj testReq
	err := json.Unmarshal(d.Body, &obj)
	return obj, err
}

func testReqEncoder(_ context.Context, p *amqp.Publishing, request testReq) error {
	b, err := json.Marshal(request)
	if err != nil {
		return err
	}
	p.Body = b
	return nil
}

func testResDeliveryDecoder(_ context.Context, d *amqp.Delivery) (testRes, error) {
	return testResDecoder(d.Body)
}

func testResDecoder(b []byte) (testRes, error) {
	var obj testRes
	err := json.Unmarshal(b, &obj)
	return obj, err
}

var names = map[int]string{
	424: "tiger",
	426: "thunderbird",
	429: "bison",
	436: "tusker",
	437: "husky",
}
