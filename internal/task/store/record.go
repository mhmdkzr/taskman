package store

import (
	"encoding/json"
	"fmt"

	"github.com/mhmdkzr/taskman/internal/task"
)

func encodeEvent(event task.TaskEvent) ([]byte, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("encode %s event: %w", event.Kind(), err)
	}
	return data, nil
}

func decodeAs[T task.TaskEvent](data []byte) (task.TaskEvent, error) {
	var event T
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, err
	}
	return event, nil
}

// decodeEvent decodes data into the concrete task.TaskEvent matching kind.
func decodeEvent(kind task.EventKind, data []byte) (task.TaskEvent, error) {
	switch kind {
	case task.EventSpecificationSubmitted:
		return decodeAs[task.SpecificationSubmitted](data)
	case task.EventImplementationCompleted:
		return decodeAs[task.ImplementationCompleted](data)
	case task.EventVerificationPassed:
		return decodeAs[task.VerificationPassed](data)
	case task.EventVerificationFailed:
		return decodeAs[task.VerificationFailed](data)
	case task.EventSpecificationReviewAgentApproved:
		return decodeAs[task.SpecificationReviewAgentApproved](data)
	case task.EventSpecificationReviewAgentRejected:
		return decodeAs[task.SpecificationReviewAgentRejected](data)
	case task.EventSpecificationReviewHumanApproved:
		return decodeAs[task.SpecificationReviewHumanApproved](data)
	case task.EventSpecificationReviewHumanRejected:
		return decodeAs[task.SpecificationReviewHumanRejected](data)
	case task.EventImplementationReviewAgentApproved:
		return decodeAs[task.ImplementationReviewAgentApproved](data)
	case task.EventImplementationReviewAgentRejected:
		return decodeAs[task.ImplementationReviewAgentRejected](data)
	case task.EventImplementationReviewHumanApproved:
		return decodeAs[task.ImplementationReviewHumanApproved](data)
	case task.EventImplementationReviewHumanRejected:
		return decodeAs[task.ImplementationReviewHumanRejected](data)
	case task.EventCommitRecorded:
		return decodeAs[task.CommitRecorded](data)
	case task.EventMergeCompleted:
		return decodeAs[task.MergeCompleted](data)
	case task.EventEscalated:
		return decodeAs[task.Escalated](data)
	case task.EventAbandoned:
		return decodeAs[task.Abandoned](data)
	default:
		return nil, fmt.Errorf("%w: unknown event kind %q", errCorruptLog, kind)
	}
}
