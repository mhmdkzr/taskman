package events

// StepFinished is published when a generation step completes.
type StepFinished struct {
	SessionID    string     `json:"session_id"`
	Number       int        `json:"number"`
	Text         string     `json:"text"`
	Reasoning    string     `json:"reasoning,omitempty"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	FinishReason string     `json:"finish_reason"`
	Usage        Usage      `json:"usage"`
}

func (StepFinished) Subject() string { return "agent.step.finished" }

func (e StepFinished) MsgID() string { return eventID(e) }
