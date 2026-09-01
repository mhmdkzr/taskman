package agent

import (
	"context"
	"fmt"

	"github.com/mhmdkzr/taskman/internal/events"
	"github.com/mhmdkzr/taskman/internal/publisher"
	"github.com/mhmdkzr/taskman/internal/store"
	"github.com/zendev-sh/goai"
)

// runObjectOnSession is Session.Run's structured-output counterpart: the
// model can still call tools across multiple steps, but the final step must
// return a value of T matching its JSON schema (auto-derived from T's struct
// tags) instead of free text. It can't be a Session method — Go doesn't
// allow generic methods — so RunObject and ContinueSessionObject both build
// a *Session themselves and call through to this. Use structured output over
// plain text whenever the caller needs to make a decision on the result
// (e.g. "did this pass?") rather than display it — it removes the need to
// prompt for a specific text shape and hand-parse the answer.
func runObjectOnSession[T any](ctx context.Context, s *Session, prompt string) (T, *Result, error) {
	var zero T
	s.messages = append(s.messages, goai.UserMessage(prompt))
	s.publish(events.TurnStarted{SessionID: s.sessionID, Prompt: prompt})
	r, err := goai.GenerateObject[T](ctx, s.model(), s.goaiOptions(s.messages)...)
	if err != nil {
		return zero, nil, err
	}
	if r != nil && len(r.ResponseMessages) > 0 {
		s.messages = append(s.messages, r.ResponseMessages...)
	}
	res := fromGoaiObjectResult(r)
	res.Messages = s.messages
	res.SessionID = s.sessionID
	s.publish(events.TurnFinished{
		SessionID:    s.sessionID,
		Prompt:       prompt,
		Text:         res.Text,
		Reasoning:    res.Reasoning,
		FinishReason: res.FinishReason,
		Usage:        usageToEvent(res.Usage),
		Steps:        len(res.Steps),
	})
	return r.Object, res, nil
}

// ContinueSessionObject is ContinueSession's structured-output counterpart:
// it runs prompt in the context of an existing session's history, and
// appends the outcome as the session's next turn.
func ContinueSessionObject[T any](ctx context.Context, st *store.Store, o Options, pub publisher.Publisher, sessionID, prompt string) (T, *Result, string, error) {
	var zero T
	tr, err := store.GetSessionTranscript(ctx, st.RW(), sessionID)
	if err != nil {
		return zero, nil, "", fmt.Errorf("continue session object: %w", err)
	}

	messages, err := transcriptToMessages(tr)
	if err != nil {
		return zero, nil, "", err
	}

	s, err := NewSession(o, pub, sessionID)
	if err != nil {
		return zero, nil, "", err
	}
	s.messages = messages

	base := len(s.messages)
	obj, res, err := runObjectOnSession[T](ctx, s, prompt)
	if err != nil {
		return zero, nil, "", err
	}

	run := store.Run{
		Prompt:   prompt,
		Messages: messagesToStore(res.Steps, s.messages[base:]),
	}
	if err := store.AppendRun(ctx, st.RW(), sessionID, run); err != nil {
		return zero, nil, "", fmt.Errorf("continue session object: %w", err)
	}
	return obj, res, sessionID, nil
}

// fromGoaiObjectResult converts a goai structured-output result into an
// agent-owned Result, the same shape fromGoaiResult produces for GenerateText
// — Text/Reasoning are simply empty here, since GenerateObject's answer lives
// in the parsed T instead.
func fromGoaiObjectResult[T any](r *goai.ObjectResult[T]) *Result {
	if r == nil {
		return nil
	}
	return &Result{
		Usage:        usageFromGoai(r.Usage),
		FinishReason: string(r.FinishReason),
		Messages:     r.ResponseMessages,
		Steps:        stepsFromGoai(r.Steps),
	}
}
