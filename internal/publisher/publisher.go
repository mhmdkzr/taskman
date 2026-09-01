package publisher

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Publisher is the sole gateway to the NATS bus: everything that sends or
// consumes messages goes through it, never the raw nats/JetStream APIs. The
// one exception is the raw connection returned by Conn, which the server uses
// for core request/reply and server-capability reads.
//
// Published messages carry a dedup id in the Nats-Msg-Id header; the TASKMAN
// stream deduplicates re-publishes of the same id within its dedup window, so
// delivery is exactly-once.
type Publisher struct {
	nc *nats.Conn
	js jetstream.JetStream
}

// NewPublisher returns a Publisher publishing through nc's JetStream context.
// An nc without JetStream yields a Publisher whose operations all fail.
func NewPublisher(nc *nats.Conn) Publisher {
	js, err := jetstream.New(nc)
	if err != nil {
		return Publisher{}
	}
	return Publisher{nc: nc, js: js}
}

// Conn returns the underlying NATS connection. It is nil on a zero Publisher.
func (p Publisher) Conn() *nats.Conn { return p.nc }

// Publish sends an event on its subject as JSON, tagged with its dedup id.
func (p Publisher) Publish(ctx context.Context, event Event[any]) error {
	if p.js == nil {
		return fmt.Errorf("publisher: jetstream is nil")
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return p.PublishMsg(ctx, event.Subject(), data, event.MsgID())
}

// PublishMsg sends a raw payload on subject with msgID as its dedup id.
func (p Publisher) PublishMsg(ctx context.Context, subject string, data []byte, msgID string) error {
	if p.js == nil {
		return fmt.Errorf("publisher: jetstream is nil")
	}
	msg := nats.NewMsg(subject)
	msg.Data = data
	msg.Header.Set(nats.MsgIdHdr, msgID)
	_, err := p.js.PublishMsg(ctx, msg)
	return err
}

// Message is one JetStream message delivered to a Subscribe handler. Call Ack
// once the message has been processed.
type Message struct {
	msg jetstream.Msg
}

// Data returns the message body.
func (m Message) Data() []byte { return m.msg.Data() }

// Subject returns the subject the message was published on.
func (m Message) Subject() string { return m.msg.Subject() }

// Ack tells the stream the message was processed, so it is not redelivered.
func (m Message) Ack() error { return m.msg.Ack() }

// Subscription is an active consumption of the TASKMAN stream. Stop drains the
// subscription and waits for it to shut down.
type Subscription struct {
	cc jetstream.ConsumeContext
}

// Stop stops consuming and blocks until the subscription has fully shut down.
func (s *Subscription) Stop() {
	if s == nil || s.cc == nil {
		return
	}
	s.cc.Drain()
	<-s.cc.Closed()
}

// Subscribe opens a durable consumer on the TASKMAN stream filtered to subject,
// delivering matching messages to handler. It is idempotent: a restart binds
// to the existing durable consumer and resumes from where it left off.
func (p Publisher) Subscribe(ctx context.Context, subject, name string, handler func(Message)) (*Subscription, error) {
	if p.js == nil {
		return nil, fmt.Errorf("publisher: jetstream is nil")
	}
	cons, err := p.js.CreateOrUpdateConsumer(ctx, StreamName, jetstream.ConsumerConfig{
		Name:          name,
		Durable:       name,
		FilterSubject: subject,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return nil, fmt.Errorf("subscribe %s: %w", subject, err)
	}
	cc, err := cons.Consume(func(msg jetstream.Msg) {
		handler(Message{msg: msg})
	})
	if err != nil {
		return nil, fmt.Errorf("subscribe %s: %w", subject, err)
	}
	return &Subscription{cc: cc}, nil
}
