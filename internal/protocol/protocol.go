// Package protocol defines the wire contract between the taskman agent client
// and the agent server: the request/reply envelope exchanged on the request
// subject, plus the payloads of each command. It is the one place client and
// server must agree, so it has no dependencies on the rest of taskman.
package protocol

import (
	"encoding/json"
	"errors"
	"strings"
)

// Subject is the NATS subject client requests are sent on and the server
// subscribes to. It lives outside the TASKMAN JetStream stream (which covers
// agent.> and scheduler.>), so requests stay transient request/reply.
const Subject = "taskman.request"

// Command names.
const (
	CommandRun  = "run"
	CommandPing = "ping"
)

// Version is the taskman agent version reported by ping.
const Version = "0.1.0"

// Request is the envelope of a client command. Data holds the command's
// payload (e.g. RunRequest for run).
type Request struct {
	Command string          `json:"command"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Reply is the envelope of a server response. OK is false and Error set when
// the command failed; Data holds the command's reply payload otherwise.
type Reply struct {
	Command string          `json:"command"`
	OK      bool            `json:"ok"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// RunRequest is the run command's payload, mirroring the run CLI flags.
type RunRequest struct {
	Prompt          string `json:"prompt"`
	SessionID       string `json:"session_id,omitempty"`
	ForkFrom        string `json:"fork_from,omitempty"`
	ForkTurn        int    `json:"fork_turn,omitempty"`
	SystemPrompt    string `json:"system_prompt,omitempty"`
	Model           string `json:"model,omitempty"`
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
	MaxSteps        int    `json:"max_steps,omitempty"`
	AsJSON          bool   `json:"json,omitempty"`
}

// Validate checks that a RunRequest forms a coherent request, mirroring the
// rules the CLI used to enforce:
//
//   - Prompt is always required.
//   - SessionID and ForkFrom are mutually exclusive.
//   - ForkFrom requires ForkTurn, and ForkTurn requires ForkFrom (turn
//     numbers only make sense as the fork point of a session).
//   - ForkTurn must be positive: turn numbers are 1-based.
func (r RunRequest) Validate() error {
	if strings.TrimSpace(r.Prompt) == "" {
		return errors.New("prompt is required")
	}
	if r.ForkFrom != "" && r.SessionID != "" {
		return errors.New("fork and session are mutually exclusive")
	}
	if r.ForkFrom != "" && r.ForkTurn < 1 {
		return errors.New("turn is required with fork")
	}
	if r.ForkTurn > 0 && r.ForkFrom == "" {
		return errors.New("turn requires fork")
	}
	return nil
}

// RunReply is the run command's reply payload. RunOutput is the full run as
// JSON when the request asked for it (AsJSON).
type RunReply struct {
	SessionID string          `json:"session_id,omitempty"`
	Text      string          `json:"text,omitempty"`
	RunOutput json.RawMessage `json:"run_output,omitempty"`
}

// PingReply is the ping command's reply payload.
type PingReply struct {
	Version string `json:"version"`
	Uptime  string `json:"uptime"`
}
