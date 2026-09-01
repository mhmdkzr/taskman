package agent

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mhmdkzr/taskman/internal/usage"
	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider"
)

// Result is the outcome of a completed generation. Text excludes the model's
// reasoning; use Reasoning for that.
type Result struct {
	// SessionID is the session this run belongs to, generated at session start.
	SessionID string

	// Text is the final answer.
	Text string

	// Reasoning is the accumulated thinking across all steps.
	Reasoning string

	// Steps are the per-step results of the tool loop.
	Steps []Step

	// Usage is the aggregated token usage.
	Usage usage.TokenUsage

	// FinishReason is why generation stopped.
	FinishReason string

	// Messages is the full transcript of the run: system, user, assistant and
	// tool messages in order.
	Messages []provider.Message
}

// Step is the result of a single generation step in the tool loop.
type Step struct {
	// ID uniquely identifies the step.
	ID string

	// Timestamp is when the step was generated, derived from the UUIDv7.
	Timestamp string

	Number       int
	Text         string     `json:"Text,omitempty"`
	Reasoning    string     `json:"Reasoning,omitempty"`
	ToolCalls    []ToolCall `json:"ToolCalls,omitempty"`
	FinishReason string
	Usage        usage.TokenUsage
}

// ToolCall is a tool invocation. For completed invocations Output holds the
// tool's result and Error its failure, if any.
type ToolCall struct {
	ID     string
	Name   string
	Input  string
	Output string
	Error  string `json:"Error,omitempty"`
}

// OptionsOutput is the serializable form of Options, excluding Config. Tools
// are rendered as ToolOutput because goai.Tool carries a non-serializable
// Execute function.
type OptionsOutput struct {
	Model           string          `json:"Model,omitempty"`
	Tools           []ToolOutput    `json:"Tools,omitempty"`
	ReasoningEffort ReasoningEffort `json:"ReasoningEffort,omitempty"`
	MaxSteps        int             `json:"MaxSteps"`
}

// ToolOutput is the serializable form of a goai.Tool.
type ToolOutput struct {
	Name                string          `json:"Name"`
	Description         string          `json:"Description,omitempty"`
	InputSchema         json.RawMessage `json:"InputSchema,omitempty"`
	ProviderDefinedType string          `json:"ProviderDefinedType,omitempty"`
}

func optionsOutput(o Options) OptionsOutput {
	out := OptionsOutput{
		Model:           o.Model,
		ReasoningEffort: o.ReasoningEffort,
		MaxSteps:        o.MaxSteps,
	}
	for _, t := range o.Tools {
		out.Tools = append(out.Tools, ToolOutput{
			Name:                t.Name,
			Description:         t.Description,
			InputSchema:         t.InputSchema,
			ProviderDefinedType: t.ProviderDefinedType,
		})
	}
	return out
}

// RunOutput is the top-level serializable result of a single run. The message
// transcript is the single source of truth, grouped into Steps where each step
// bundles the messages it produced. Every message carries its own identity and
// metadata.
type RunOutput struct {
	ID        string           `json:"ID"`
	SessionID string           `json:"SessionID"`
	Timestamp string           `json:"Timestamp"`
	Options   OptionsOutput    `json:"Options"`
	Usage     usage.TokenUsage `json:"Usage,omitempty"`
	Steps     []stepOutput     `json:"Steps,omitempty"`
}

// stepOutput groups the messages produced by one step of the tool loop. The
// first group holds the pre-step messages (system, user).
type stepOutput struct {
	Messages []messageOutput `json:"Messages,omitempty"`
}

// NewRunOutput wraps a completed run's transcript, options, aggregated usage,
// and per-step metadata into a serializable RunOutput with a fresh run ID and
// timestamp.
func NewRunOutput(sessionID string, o Options, tokenUsage usage.TokenUsage, messages []provider.Message, steps []Step) RunOutput {
	id, ts := newStepID()
	return RunOutput{
		ID:        id,
		SessionID: sessionID,
		Timestamp: ts,
		Options:   optionsOutput(o),
		Usage:     tokenUsage,
		Steps:     stepsOutput(messages, steps),
	}
}

// messageOutput is the serializable form of a provider.Message. Each message
// of a step carries the step's ID, timestamp and number; assistant messages
// additionally carry the step's FinishReason and Usage, so no step-level
// information is lost.
type messageOutput struct {
	Role            string            `json:"Role"`
	Content         []partOutput      `json:"Content,omitempty"`
	ID              string            `json:"ID,omitempty"`
	Timestamp       string            `json:"Timestamp,omitempty"`
	Number          int               `json:"Number,omitempty"`
	FinishReason    string            `json:"FinishReason,omitempty"`
	Usage           *usage.TokenUsage `json:"Usage,omitempty"`
	ProviderOptions map[string]any    `json:"ProviderOptions,omitempty"`
}

// partOutput is the serializable form of a provider.Part.
type partOutput struct {
	Type            string               `json:"Type"`
	Text            string               `json:"Text,omitempty"`
	URL             string               `json:"URL,omitempty"`
	ToolCallID      string               `json:"ToolCallID,omitempty"`
	ToolName        string               `json:"ToolName,omitempty"`
	ToolInput       json.RawMessage      `json:"ToolInput,omitempty"`
	ToolOutput      string               `json:"ToolOutput,omitempty"`
	Error           string               `json:"Error,omitempty"`
	CacheControl    string               `json:"CacheControl,omitempty"`
	CacheControlTTL string               `json:"CacheControlTTL,omitempty"`
	Detail          string               `json:"Detail,omitempty"`
	MediaType       string               `json:"MediaType,omitempty"`
	Filename        string               `json:"Filename,omitempty"`
	RemoteRef       *remoteFileRefOutput `json:"RemoteRef,omitempty"`
	ProviderOptions map[string]any       `json:"ProviderOptions,omitempty"`
}

// remoteFileRefOutput is the serializable form of a provider.RemoteFileRef.
type remoteFileRefOutput struct {
	Provider  string `json:"Provider,omitempty"`
	ID        string `json:"ID,omitempty"`
	URI       string `json:"URI,omitempty"`
	Filename  string `json:"Filename,omitempty"`
	MediaType string `json:"MediaType,omitempty"`
	ExpiresAt string `json:"ExpiresAt,omitempty"`
}

// messagesOutput converts a transcript plus its steps into serializable
// messages, matching each assistant message to the step that produced it.
func stepsOutput(messages []provider.Message, steps []Step) []stepOutput {
	if len(messages) == 0 {
		return nil
	}
	errByCallID := make(map[string]string)
	for _, st := range steps {
		for _, tc := range st.ToolCalls {
			if tc.Error != "" {
				errByCallID[tc.ID] = tc.Error
			}
		}
	}
	out := make([]stepOutput, 0)
	group := make([]messageOutput, 0)
	stepIdx := 0
	flush := func() {
		if len(group) > 0 {
			out = append(out, stepOutput{Messages: group})
			group = nil
		}
	}
	for _, m := range messages {
		mo := messageOutput{
			Role:            string(m.Role),
			Content:         partsOutput(m.Content, errByCallID),
			ProviderOptions: m.ProviderOptions,
		}
		switch m.Role {
		case provider.RoleAssistant:
			if stepIdx < len(steps) {
				st := steps[stepIdx]
				mo.ID = st.ID
				mo.Timestamp = st.Timestamp
				mo.Number = st.Number
				mo.FinishReason = st.FinishReason
				mo.Usage = &st.Usage
				stepIdx++
			}
			flush()
		case provider.RoleTool:
			if stepIdx > 0 {
				st := steps[stepIdx-1]
				mo.ID = st.ID
				mo.Timestamp = st.Timestamp
				mo.Number = st.Number
			}
		default:
			mo.ID, mo.Timestamp = newStepID()
		}
		group = append(group, mo)
	}
	flush()
	return out
}

func partsOutput(parts []provider.Part, errByCallID map[string]string) []partOutput {
	if len(parts) == 0 {
		return nil
	}
	out := make([]partOutput, 0, len(parts))
	for _, p := range parts {
		po := partOutput{
			Type:            string(p.Type),
			Text:            p.Text,
			URL:             p.URL,
			ToolCallID:      p.ToolCallID,
			ToolName:        p.ToolName,
			ToolInput:       p.ToolInput,
			ToolOutput:      p.ToolOutput,
			CacheControl:    p.CacheControl,
			CacheControlTTL: p.CacheControlTTL,
			Detail:          p.Detail,
			MediaType:       p.MediaType,
			Filename:        p.Filename,
			RemoteRef:       remoteFileRefView(p.RemoteRef),
			ProviderOptions: p.ProviderOptions,
		}
		if p.Type == provider.PartToolResult {
			po.Error = errByCallID[p.ToolCallID]
		}
		out = append(out, po)
	}
	return out
}

func remoteFileRefView(r *provider.RemoteFileRef) *remoteFileRefOutput {
	if r == nil {
		return nil
	}
	var expiresAt string
	if !r.ExpiresAt.IsZero() {
		expiresAt = r.ExpiresAt.Format(time.RFC3339)
	}
	return &remoteFileRefOutput{
		Provider:  r.Provider,
		ID:        r.ID,
		URI:       r.URI,
		Filename:  r.Filename,
		MediaType: r.MediaType,
		ExpiresAt: expiresAt,
	}
}

// fromGoaiResult converts a goai result into an agent-owned Result.
func fromGoaiResult(r *goai.TextResult) *Result {
	if r == nil {
		return nil
	}
	return &Result{
		Text:         resultText(r),
		Reasoning:    r.Reasoning,
		Usage:        usageFromGoai(r.TotalUsage),
		FinishReason: string(r.FinishReason),
		Messages:     r.ResponseMessages,
		Steps:        stepsFromGoai(r.Steps),
	}
}

// stepsFromGoai converts goai's per-step results into agent-owned Steps,
// shared by fromGoaiResult (GenerateText) and fromGoaiObjectResult
// (GenerateObject) — both produce the same goai.StepResult shape.
func stepsFromGoai(steps []goai.StepResult) []Step {
	out := make([]Step, 0, len(steps))
	for _, st := range steps {
		id, timestamp := newStepID()
		step := Step{
			ID:           id,
			Timestamp:    timestamp,
			Number:       st.Number,
			Text:         st.Text,
			Reasoning:    st.Reasoning,
			FinishReason: string(st.FinishReason),
			Usage:        usageFromGoai(st.Usage),
		}
		for i, tc := range st.ToolCalls {
			call := ToolCall{
				ID:    tc.ID,
				Name:  tc.Name,
				Input: string(tc.Input),
			}
			if i < len(st.ToolResults) {
				tr := st.ToolResults[i]
				call.Output = tr.Output
				if tr.Error != nil {
					call.Error = tr.Error.Error()
				}
			}
			step.ToolCalls = append(step.ToolCalls, call)
		}
		out = append(out, step)
	}
	return out
}

// resultText returns the answer text excluding reasoning. goai's
// TextResult.Text includes reasoning for streaming; Steps[].Text never does.
func resultText(r *goai.TextResult) string {
	if len(r.Steps) == 0 {
		return r.Text
	}
	var b strings.Builder
	for _, st := range r.Steps {
		b.WriteString(st.Text)
	}
	return b.String()
}

func usageFromGoai(u provider.Usage) usage.TokenUsage {
	return usage.TokenUsage{
		InputTokens:      u.InputTokens,
		OutputTokens:     u.OutputTokens,
		TotalTokens:      u.TotalTokens,
		ReasoningTokens:  u.ReasoningTokens,
		CacheReadTokens:  u.CacheReadTokens,
		CacheWriteTokens: u.CacheWriteTokens,
	}
}

// newStepID returns a UUIDv7 and the timestamp embedded in it, so both are
// always in sync.
func newStepID() (id string, timestamp string) {
	u, err := uuid.NewV7()
	if err != nil {
		return "", time.Now().UTC().String()
	}
	return u.String(), unixMillisFromUUIDV7(u).UTC().String()
}

// unixMillisFromUUIDV7 decodes the Unix time in milliseconds that a UUIDv7 embeds in
// its first 48 bits (big-endian).
func unixMillisFromUUIDV7(u uuid.UUID) time.Time {
	ms := int64(u[0])<<40 | int64(u[1])<<32 | int64(u[2])<<24 |
		int64(u[3])<<16 | int64(u[4])<<8 | int64(u[5])
	return time.UnixMilli(ms)
}
