package events

// Request is published when a model call is initiated.
type Request struct {
	SessionID    string `json:"session_id"`
	Model        string `json:"model"`
	MessageCount int    `json:"message_count"`
	ToolCount    int    `json:"tool_count"`
	Timestamp    string `json:"timestamp"`
}

func (Request) Subject() string { return "agent.model.request" }

func (e Request) MsgID() string { return eventID(e) }
