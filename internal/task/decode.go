package task

import (
	"encoding/json"
	"fmt"
)

func decodeAs[T TaskEvent](data []byte) (TaskEvent, error) {
	var event T
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, err
	}
	return event, nil
}

// eventDecoders is the one place mapping each EventKind to the concrete Go
// type its Kind() returns - the closed set of TaskEvent types, keyed by their
// own identifier. A caller that only has a stored EventKind and raw bytes
// (internal/task/store) decodes through DecodeEvent instead of keeping a
// second copy of this list that could drift from Kind()'s own switch.
var eventDecoders = map[EventKind]func([]byte) (TaskEvent, error){
	EventSpecificationSubmitted:            decodeAs[SpecificationSubmitted],
	EventImplementationCompleted:           decodeAs[ImplementationCompleted],
	EventVerificationPassed:                decodeAs[VerificationPassed],
	EventVerificationFailed:                decodeAs[VerificationFailed],
	EventCommitRecorded:                    decodeAs[CommitRecorded],
	EventMergeCompleted:                    decodeAs[MergeCompleted],
	EventEscalated:                         decodeAs[Escalated],
	EventAbandoned:                         decodeAs[Abandoned],
	EventUnblocked:                         decodeAs[Unblocked],
	EventSpecificationReviewAgentApproved:  decodeAs[SpecificationReviewAgentApproved],
	EventSpecificationReviewAgentRejected:  decodeAs[SpecificationReviewAgentRejected],
	EventSpecificationReviewHumanApproved:  decodeAs[SpecificationReviewHumanApproved],
	EventSpecificationReviewHumanRejected:  decodeAs[SpecificationReviewHumanRejected],
	EventImplementationReviewAgentApproved: decodeAs[ImplementationReviewAgentApproved],
	EventImplementationReviewAgentRejected: decodeAs[ImplementationReviewAgentRejected],
	EventImplementationReviewHumanApproved: decodeAs[ImplementationReviewHumanApproved],
	EventImplementationReviewHumanRejected: decodeAs[ImplementationReviewHumanRejected],
	EventLabelsUpdated:                     decodeAs[LabelsUpdated],
}

// DecodeEvent decodes data into the concrete TaskEvent matching kind, per
// eventDecoders. It returns ErrUnknownEventKind for a kind with no
// registered decoder; any other error is data.(json.Unmarshal) failing on a
// malformed payload for an otherwise-known kind.
func DecodeEvent(kind EventKind, data []byte) (TaskEvent, error) {
	decode, ok := eventDecoders[kind]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownEventKind, kind)
	}
	return decode(data)
}
