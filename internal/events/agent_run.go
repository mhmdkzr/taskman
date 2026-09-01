package events

// AgentRunStarted is published when a message on agent.run is consumed and its
// agent begins running. SessionID is empty until the run is persisted.
type AgentRunStarted struct {
	SessionID string `json:"session_id"`
	Prompt    string `json:"prompt"`
	Model     string `json:"model"`
}

func (AgentRunStarted) Subject() string { return "agent.run.started" }

func (e AgentRunStarted) MsgID() string { return eventID(e) }

// AgentRunFinished is published when a scheduled agent run completes and is
// persisted.
type AgentRunFinished struct {
	SessionID string `json:"session_id"`
	Prompt    string `json:"prompt"`
	Text      string `json:"text"`
	Usage     Usage  `json:"usage"`
	Steps     int    `json:"steps"`
}

func (AgentRunFinished) Subject() string { return "agent.run.finished" }

func (e AgentRunFinished) MsgID() string { return eventID(e) }

// AgentRunFailed is published when a scheduled agent run cannot be started or
// fails.
type AgentRunFailed struct {
	SessionID string `json:"session_id"`
	Prompt    string `json:"prompt"`
	Error     string `json:"error"`
}

func (AgentRunFailed) Subject() string { return "agent.run.failed" }

func (e AgentRunFailed) MsgID() string { return eventID(e) }
