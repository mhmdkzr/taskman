package events

// SubagentSpawned is published when a sub-agent session is created.
type SubagentSpawned struct {
	SessionID string `json:"session_id"`
	Depth     int    `json:"depth"`
	Prompt    string `json:"prompt"`
	Model     string `json:"model"`
	Wait      bool   `json:"wait"`
}

func (SubagentSpawned) Subject() string { return "agent.subagent.spawned" }

func (e SubagentSpawned) MsgID() string { return eventID(e) }

// SubagentFinished is published when a sub-agent completes successfully.
type SubagentFinished struct {
	SessionID string `json:"session_id"`
	Text      string `json:"text"`
	Usage     Usage  `json:"usage"`
	Steps     int    `json:"steps"`
}

func (SubagentFinished) Subject() string { return "agent.subagent.finished" }

func (e SubagentFinished) MsgID() string { return eventID(e) }

// SubagentFailed is published when a sub-agent fails or panics.
type SubagentFailed struct {
	SessionID string `json:"session_id"`
	Prompt    string `json:"prompt"`
	Error     string `json:"error"`
}

func (SubagentFailed) Subject() string { return "agent.subagent.failed" }

func (e SubagentFailed) MsgID() string { return eventID(e) }
