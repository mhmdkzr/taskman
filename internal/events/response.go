package events

// Response is published when a model call completes.
type Response struct {
	SessionID    string `json:"session_id"`
	Latency      string `json:"latency"`
	Usage        Usage  `json:"usage"`
	FinishReason string `json:"finish_reason"`
	Error        string `json:"error,omitempty"`
	StatusCode   int    `json:"status_code"`
}

func (Response) Subject() string { return "agent.model.response" }

func (e Response) MsgID() string { return eventID(e) }
