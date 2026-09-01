// Package usage defines the token-consumption shape shared by every layer
// that records it: an agent result, a persisted message/session, and a task.
// One canonical type here means those layers can alias it instead of each
// keeping their own copy in lockstep.
package usage

// TokenUsage is token consumption for one generation, one message, or the
// sum across a session/task.
type TokenUsage struct {
	InputTokens      int `json:"input_tokens"`
	OutputTokens     int `json:"output_tokens"`
	TotalTokens      int `json:"total_tokens"`
	ReasoningTokens  int `json:"reasoning_tokens"`
	CacheReadTokens  int `json:"cache_read_tokens"`
	CacheWriteTokens int `json:"cache_write_tokens"`
}

// Add returns the element-wise sum of two usages, for accumulating totals
// across messages, sessions, or a task's execution + review sessions.
func (u TokenUsage) Add(o TokenUsage) TokenUsage {
	return TokenUsage{
		InputTokens:      u.InputTokens + o.InputTokens,
		OutputTokens:     u.OutputTokens + o.OutputTokens,
		TotalTokens:      u.TotalTokens + o.TotalTokens,
		ReasoningTokens:  u.ReasoningTokens + o.ReasoningTokens,
		CacheReadTokens:  u.CacheReadTokens + o.CacheReadTokens,
		CacheWriteTokens: u.CacheWriteTokens + o.CacheWriteTokens,
	}
}
