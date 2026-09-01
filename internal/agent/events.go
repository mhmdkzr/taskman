package agent

import (
	"github.com/mhmdkzr/taskman/internal/events"
	"github.com/mhmdkzr/taskman/internal/usage"
	"github.com/zendev-sh/goai/provider"
)

func usageEvent(u provider.Usage) events.Usage {
	return events.Usage{
		InputTokens:      u.InputTokens,
		OutputTokens:     u.OutputTokens,
		TotalTokens:      u.TotalTokens,
		ReasoningTokens:  u.ReasoningTokens,
		CacheReadTokens:  u.CacheReadTokens,
		CacheWriteTokens: u.CacheWriteTokens,
	}
}

func usageToEvent(u usage.TokenUsage) events.Usage {
	return events.Usage{
		InputTokens:      u.InputTokens,
		OutputTokens:     u.OutputTokens,
		TotalTokens:      u.TotalTokens,
		ReasoningTokens:  u.ReasoningTokens,
		CacheReadTokens:  u.CacheReadTokens,
		CacheWriteTokens: u.CacheWriteTokens,
	}
}

func toolCallEvent(tc provider.ToolCall) events.ToolCall {
	return events.ToolCall{ID: tc.ID, Name: tc.Name, Input: string(tc.Input)}
}
