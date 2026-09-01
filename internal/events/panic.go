package events

// Panic is published when a hook or tool callback panics.
type Panic struct {
	SessionID string `json:"session_id"`
	Phase     string `json:"phase"`
	Value     string `json:"value"`
	Stack     string `json:"stack"`
}

func (Panic) Subject() string { return "agent.panic" }

func (e Panic) MsgID() string { return eventID(e) }
