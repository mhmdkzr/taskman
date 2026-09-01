package events

// GenerationFinished is published when the model tool loop of a turn stops.
type GenerationFinished struct {
	SessionID      string `json:"session_id"`
	StepsExhausted bool   `json:"steps_exhausted"`
	TotalSteps     int    `json:"total_steps"`
	TotalUsage     Usage  `json:"total_usage"`
	FinishReason   string `json:"finish_reason"`
	StoppedBy      string `json:"stopped_by"`
}

func (GenerationFinished) Subject() string { return "agent.generation.finished" }

func (e GenerationFinished) MsgID() string { return eventID(e) }
