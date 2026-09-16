package task

import (
	"errors"
	"testing"
)

func TestDecodeEventRoundTrips(t *testing.T) {
	event, err := DecodeEvent(EventAbandoned, []byte(`{"reason":"moot","at":"2024-01-01T00:00:00Z"}`))
	if err != nil {
		t.Fatalf("DecodeEvent() error = %v", err)
	}
	abandoned, ok := event.(Abandoned)
	if !ok {
		t.Fatalf("DecodeEvent() = %T, want Abandoned", event)
	}
	if abandoned.Reason != "moot" {
		t.Fatalf("Reason = %q, want %q", abandoned.Reason, "moot")
	}
}

func TestDecodeEventRejectsUnknownKind(t *testing.T) {
	if _, err := DecodeEvent(EventKind("not_a_real_kind"), []byte(`{}`)); !errors.Is(err, ErrUnknownEventKind) {
		t.Fatalf("DecodeEvent() error = %v, want ErrUnknownEventKind", err)
	}
}

func TestDecodeEventRejectsMalformedPayload(t *testing.T) {
	if _, err := DecodeEvent(EventAbandoned, []byte(`not json`)); err == nil {
		t.Fatal("DecodeEvent() error = nil, want a JSON decode error")
	}
}

// TestEventDecodersCoverEveryKind pins the invariant DecodeEvent depends on:
// every EventKind a Kind() method can return has a matching decoder, so a
// new event type can't be wired into the workflow while forgetting to
// register how to decode it back out of storage.
func TestEventDecodersCoverEveryKind(t *testing.T) {
	kinds := []EventKind{
		SpecificationSubmitted{}.Kind(),
		ImplementationCompleted{}.Kind(),
		VerificationPassed{}.Kind(),
		VerificationFailed{}.Kind(),
		CommitRecorded{}.Kind(),
		MergeCompleted{}.Kind(),
		Escalated{}.Kind(),
		Abandoned{}.Kind(),
		Unblocked{}.Kind(),
		SpecificationReviewAgentApproved{}.Kind(),
		SpecificationReviewAgentRejected{}.Kind(),
		SpecificationReviewHumanApproved{}.Kind(),
		SpecificationReviewHumanRejected{}.Kind(),
		ImplementationReviewAgentApproved{}.Kind(),
		ImplementationReviewAgentRejected{}.Kind(),
		ImplementationReviewHumanApproved{}.Kind(),
		ImplementationReviewHumanRejected{}.Kind(),
	}
	if len(kinds) != len(eventDecoders) {
		t.Fatalf("len(eventDecoders) = %d, want %d (one per known EventKind)", len(eventDecoders), len(kinds))
	}
	for _, kind := range kinds {
		if _, ok := eventDecoders[kind]; !ok {
			t.Fatalf("eventDecoders is missing a decoder for %q", kind)
		}
	}
}
