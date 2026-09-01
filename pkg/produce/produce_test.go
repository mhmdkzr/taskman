package produce

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/mhmdkzr/taskman/pkg/natsembed"
)

type eventProduceValid struct {
	Name string `json:"name"`
}

func (e eventProduceValid) MsgID() string {
	return "name=" + e.Name
}

type eventProduceMarshal struct {
	Ch chan int `json:"ch"`
}

func (e eventProduceMarshal) MsgID() string {
	return "marshal-error"
}

type eventProduceStream struct {
	ID string `json:"id"`
}

func (e eventProduceStream) MsgID() string {
	return "id=" + e.ID
}

func TestProduce_ValidatesInputs(t *testing.T) {
	ctx := context.Background()

	if err := Produce[eventProduceValid](ctx, nil, "subject", eventProduceValid{}); err == nil {
		t.Fatalf("expected nil jetstream validation error")
	}

	nc, js, err := natsembed.Connect()
	if err != nil {
		t.Fatalf("natsembed connect: %v", err)
	}
	t.Cleanup(nc.Close)

	if err := Produce(ctx, js, "", eventProduceValid{}); err == nil {
		t.Fatalf("expected empty subject validation error")
	}

	if err := Produce(ctx, js, "subject", eventProduceMarshal{Ch: make(chan int)}); err == nil {
		t.Fatalf("expected marshal validation error")
	}
}

func TestProduce_PublishesToExpectedStream(t *testing.T) {
	ctx := context.Background()

	nc, js, err := natsembed.Connect()
	if err != nil {
		t.Fatalf("natsembed connect: %v", err)
	}
	t.Cleanup(nc.Close)

	const (
		streamName = "TEST_STREAM"
		subject    = "test.subject"
	)

	stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     streamName,
		Subjects: []string{subject},
	})
	if err != nil {
		t.Fatalf("create stream: %v", err)
	}

	want := eventProduceStream{ID: "evt-1"}
	if err := Produce(ctx, js, subject, want); err != nil {
		t.Fatalf("produce: %v", err)
	}

	msgs, err := stream.GetMsg(ctx, 1)
	if err != nil {
		t.Fatalf("get msg: %v", err)
	}

	var got eventProduceStream
	if err := json.Unmarshal(msgs.Data, &got); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	if got != want {
		t.Fatalf("event mismatch: got=%+v want=%+v", got, want)
	}
}

func TestProduce_SetsMsgIDHeader(t *testing.T) {
	ctx := context.Background()

	nc, js, err := natsembed.Connect()
	if err != nil {
		t.Fatalf("natsembed connect: %v", err)
	}
	t.Cleanup(nc.Close)

	nonce := time.Now().UnixNano()
	streamName := fmt.Sprintf("TEST_STREAM_DEDUP_%d", nonce)
	subject := fmt.Sprintf("test.dedup.%d", nonce)
	stream, err := js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     streamName,
		Subjects: []string{subject},
	})
	if err != nil {
		t.Fatalf("create stream: %v", err)
	}

	want := eventProduceStream{ID: "evt-1"}
	if err := Produce(ctx, js, subject, want); err != nil {
		t.Fatalf("first produce: %v", err)
	}

	gotMsg, err := stream.GetMsg(ctx, 1)
	if err != nil {
		t.Fatalf("get msg: %v", err)
	}
	if got := gotMsg.Header.Get("Nats-Msg-Id"); got != want.MsgID() {
		t.Fatalf("unexpected msg id header: got=%q want=%q", got, want.MsgID())
	}
}

func TestProduce_SubjectWithoutStreamReturnsError(t *testing.T) {
	ctx := context.Background()

	nc, js, err := natsembed.Connect()
	if err != nil {
		t.Fatalf("natsembed connect: %v", err)
	}
	t.Cleanup(nc.Close)

	const subject = "missing.subject"
	if err := Produce(ctx, js, subject, eventProduceValid{Name: "x"}); err == nil {
		t.Fatalf("expected publish error for subject without stream")
	}
}
