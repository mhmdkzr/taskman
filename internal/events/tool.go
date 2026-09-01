package events

// ToolCall is a model-requested tool invocation.
type ToolCall struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Input string `json:"input"`
}

// ToolCallStarted is published before a tool executes.
type ToolCallStarted struct {
	SessionID string `json:"session_id"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	Step      int    `json:"step"`
	Input     string `json:"input"`
}

func (ToolCallStarted) Subject() string { return "agent.tool.started" }

func (e ToolCallStarted) MsgID() string { return eventID(e) }

// ToolCalled is published after a tool executes with its outcome.
type ToolCalled struct {
	SessionID string `json:"session_id"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	Step      int    `json:"step"`
	Input     string `json:"input"`
	Output    string `json:"output"`
	Error     string `json:"error,omitempty"`
	Duration  string `json:"duration"`
	Skipped   bool   `json:"skipped"`
}

func (ToolCalled) Subject() string { return "agent.tool.finished" }

func (e ToolCalled) MsgID() string { return eventID(e) }
