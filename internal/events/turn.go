package events

// TurnStarted is published when a session turn begins.
type TurnStarted struct {
	SessionID string `json:"session_id"`
	Prompt    string `json:"prompt"`
}

func (TurnStarted) Subject() string { return "agent.turn.started" }

func (e TurnStarted) MsgID() string { return eventID(e) }

// TurnFinished is published when a session turn completes (non-streaming).
type TurnFinished struct {
	SessionID    string `json:"session_id"`
	Prompt       string `json:"prompt"`
	Text         string `json:"text"`
	Reasoning    string `json:"reasoning,omitempty"`
	FinishReason string `json:"finish_reason"`
	Usage        Usage  `json:"usage"`
	Steps        int    `json:"steps"`
}

func (TurnFinished) Subject() string { return "agent.turn.finished" }

func (e TurnFinished) MsgID() string { return eventID(e) }
