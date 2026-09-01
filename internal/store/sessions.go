package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mhmdkzr/taskman/internal/usage"
)

// Per-message token consumption (usage.TokenUsage) lives inline on messages;
// a session's total is a SUM over its messages.

// Tool mirrors a row of tools.
type Tool struct {
	ID                  string
	Name                string
	Description         string
	InputSchema         string
	ProviderDefinedType string
}

// Session mirrors a session plus its options, tools and fork lineage. The
// system prompt is derived from the system message at the root of the session's
// chain. LeafMessageID is the session's anchor into the shared message chain
// (its newest message); the transcript is the walk from it back to the root.
type Session struct {
	ID              string
	CreatedAt       string
	LeafMessageID   string
	Model           string
	ReasoningEffort string
	MaxSteps        int
	SystemPrompt    string
	Tools           []Tool
	ParentID        string
	ForkTurnNumber  int
}

// Turn is one user prompt -> agent response cycle. Turns are not stored as
// rows; they are the grouping of a session's chain by turn_number, so a Turn is
// derived from its prompt message: ID is the prompt message's id, Number its
// turn_number, and CreatedAt the prompt's timestamp.
type Turn struct {
	ID        string
	Number    int
	CreatedAt string
}

// Step is the read-side view of one tool-loop iteration: the assistant message
// carrying a given number, plus its finish reason and usage.
type Step struct {
	Number       int
	FinishReason string
	Usage        usage.TokenUsage
	CreatedAt    string
}

// Message is a transcript entry in the shared chain. ParentID links it to its
// predecessor (empty on the chain root). TurnNumber is the turn it belongs to
// (0 on the session-wide system message); Number is the tool-loop step within
// the turn; finish_reason and usage are set on assistant messages.
type Message struct {
	ID              string
	ParentID        string
	Role            string
	Number          int
	TurnNumber      int
	FinishReason    string
	Usage           usage.TokenUsage
	ProviderOptions string
	CreatedAt       string
	Parts           []Part
}

// Part is one content element of a message. Tool name/input live on tool-call
// parts; output/error live on tool-result parts.
type Part struct {
	Type            string
	Text            string
	URL             string
	ToolCallID      string
	ToolName        string
	ToolInput       string
	ToolOutput      string
	ToolError       string
	CacheControl    string
	CacheControlTTL string
	Detail          string
	MediaType       string
	Filename        string
	RemoteRef       string
	ProviderOptions string
}

// HistoryEntry is one text part of a conversation turn, flattened across
// sessions. Entries are ordered newest session first and chronologically
// within each turn.
type HistoryEntry struct {
	SessionID        string
	SessionCreatedAt string
	TurnNumber       int
	Role             string
	Text             string
}

// Transcript is a full session: options, system prompt, tools, and every turn
// with its messages grouped into steps.
type Transcript struct {
	Session *Session
	Turns   []TurnTranscript
}

type TurnTranscript struct {
	Turn   Turn
	Prompt Message
	Steps  []StepTranscript
}

type StepTranscript struct {
	Step     Step
	Messages []Message
}

// Run is the data needed to persist one completed run (one turn).
type Run struct {
	Model           string
	ReasoningEffort string
	MaxSteps        int
	SystemPrompt    string
	Tools           []Tool
	Prompt          string
	Messages        []Message
}

// NewSessionID generates a fresh session id. The id is a UUIDv7 whose embedded
// timestamp becomes the session's created_at.
func NewSessionID() (string, error) {
	id, _ := newIDWithTime()
	if id == "" {
		return "", fmt.Errorf("generate session id")
	}
	return id, nil
}

// CreateSession inserts a session with the given pre-generated id, its options,
// system prompt (as the root system message) and tools. The id must come from
// NewSessionID so its embedded timestamp reflects session start.
func CreateSession(ctx context.Context, db *sql.DB, sessionID, model, reasoningEffort string, maxSteps int, systemPrompt string, tools []Tool) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create session: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := createSession(ctx, tx, sessionID, model, reasoningEffort, maxSteps, systemPrompt, tools); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit create session: %w", err)
	}
	return nil
}

func createSession(ctx context.Context, q DBTx, sessionID, model, reasoningEffort string, maxSteps int, systemPrompt string, tools []Tool) error {
	now := timestampFromID(sessionID)

	// The system message, when present, is the root of the session's chain and
	// its initial leaf. It must exist before the session row references it.
	var leaf any
	if systemPrompt != "" {
		msgID, msgNow := newIDWithTime()
		if err := insertMessage(ctx, q, msgID, "", Message{
			Role:      "system",
			CreatedAt: msgNow,
			Parts:     []Part{{Type: "text", Text: systemPrompt}},
		}); err != nil {
			return fmt.Errorf("insert system message: %w", err)
		}
		leaf = msgID
	}

	if _, err := q.ExecContext(ctx, `INSERT INTO sessions (id, created_at, leaf_message_id) VALUES (?, ?, ?)`,
		sessionID, now, leaf); err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	if _, err := q.ExecContext(ctx, `INSERT INTO options (session_id, model, reasoning_effort, max_steps)
		VALUES (?, ?, ?, ?)`,
		sessionID, model, reasoningEffort, maxSteps); err != nil {
		return fmt.Errorf("insert options: %w", err)
	}

	for _, t := range tools {
		if t.ID == "" {
			t.ID = t.Name
		}
		if err := upsertTool(ctx, q, t); err != nil {
			return err
		}
		if _, err := q.ExecContext(ctx, `INSERT INTO session_tools (session_id, tool_id) VALUES (?, ?)`,
			sessionID, t.ID); err != nil {
			return fmt.Errorf("link session tool %s: %w", t.Name, err)
		}
	}

	return nil
}

// GetSession loads a session and its options, system prompt, tools and lineage.
func GetSession(ctx context.Context, db *sql.DB, sessionID string) (*Session, error) {
	var s Session
	var parentID sql.NullString
	var forkTurn sql.NullInt64
	var leaf sql.NullString
	err := db.QueryRowContext(ctx, `
		SELECT s.id, s.created_at, s.leaf_message_id, o.model, o.reasoning_effort, o.max_steps, f.parent_id, f.turn_number
		FROM sessions s
		JOIN options o ON o.session_id = s.id
		LEFT JOIN forks f ON f.session_id = s.id
		WHERE s.id = ?`, sessionID).
		Scan(&s.ID, &s.CreatedAt, &leaf, &s.Model, &s.ReasoningEffort, &s.MaxSteps, &parentID, &forkTurn)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	if err != nil {
		return nil, fmt.Errorf("load session: %w", err)
	}
	s.LeafMessageID = leaf.String
	s.ParentID = parentID.String
	if forkTurn.Valid {
		s.ForkTurnNumber = int(forkTurn.Int64)
	}

	if s.SystemPrompt, err = systemPrompt(ctx, db, &s); err != nil {
		return nil, err
	}
	s.Tools, err = sessionTools(ctx, db, sessionID)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// ListSessions returns all sessions, newest first. Tools and the system prompt
// are not loaded; use GetSession for the full view.
func ListSessions(ctx context.Context, db *sql.DB) ([]Session, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT s.id, s.created_at, s.leaf_message_id, o.model, o.reasoning_effort, o.max_steps, f.parent_id, f.turn_number
		FROM sessions s
		JOIN options o ON o.session_id = s.id
		LEFT JOIN forks f ON f.session_id = s.id
		ORDER BY s.created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	var out []Session
	for rows.Next() {
		var s Session
		var parentID sql.NullString
		var forkTurn sql.NullInt64
		var leaf sql.NullString
		if err := rows.Scan(&s.ID, &s.CreatedAt, &leaf, &s.Model, &s.ReasoningEffort, &s.MaxSteps, &parentID, &forkTurn); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		s.LeafMessageID = leaf.String
		s.ParentID = parentID.String
		if forkTurn.Valid {
			s.ForkTurnNumber = int(forkTurn.Int64)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// systemPrompt walks a session's chain to its root; the root is the system
// message when the session (or a session it forked from) has one.
func systemPrompt(ctx context.Context, q DBTx, s *Session) (string, error) {
	cur := s.LeafMessageID
	for cur != "" {
		var id string
		var parent sql.NullString
		var role, content string
		if err := q.QueryRowContext(ctx,
			`SELECT id, parent_id, role, content FROM messages WHERE id = ?`, cur).
			Scan(&id, &parent, &role, &content); err != nil {
			return "", fmt.Errorf("walk system prompt: %w", err)
		}
		if !parent.Valid {
			if role == "system" {
				return partsText(unmarshalParts(content)), nil
			}
			return "", nil
		}
		cur = parent.String
	}
	return "", nil
}

func sessionTools(ctx context.Context, db *sql.DB, sessionID string) ([]Tool, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT t.id, t.name, t.description, t.input_schema, t.provider_defined_type
		FROM session_tools st
		JOIN tools t ON st.tool_id = t.id
		WHERE st.session_id = ?
		ORDER BY t.name`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("load session tools: %w", err)
	}
	defer rows.Close()

	var out []Tool
	for rows.Next() {
		var t Tool
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.InputSchema, &t.ProviderDefinedType); err != nil {
			return nil, fmt.Errorf("scan session tool: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteSession removes a session and the messages it alone references: its
// options, fork lineage and tool links cascade; shared messages survive
// because other sessions still reference them. Messages no longer reachable
// from any session's leaf are garbage-collected tail-first (deepest orphan
// first), which the RESTRICT foreign keys permit because each deletion is a
// childless, non-leaf message.
func DeleteSession(ctx context.Context, db *sql.DB, sessionID string) error {
	res, err := db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, sessionID)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	for {
		res, err := db.ExecContext(ctx, `
			DELETE FROM messages
			WHERE id NOT IN (
				SELECT leaf_message_id FROM sessions WHERE leaf_message_id IS NOT NULL
				UNION
				SELECT parent_id FROM messages WHERE parent_id IS NOT NULL
			)`)
		if err != nil {
			return fmt.Errorf("gc messages: %w", err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n == 0 {
			return nil
		}
	}
}

// SaveRun persists a complete run as one transaction: the session, its options,
// system message, tools, one turn and every message. Either everything is
// written or nothing is. sessionID is the pre-generated id of the new session.
func SaveRun(ctx context.Context, db *sql.DB, sessionID string, run Run) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin save run: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := createSession(ctx, tx, sessionID, run.Model, run.ReasoningEffort, run.MaxSteps, run.SystemPrompt, run.Tools); err != nil {
		return err
	}
	if err := appendRun(ctx, tx, sessionID, run); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit save run: %w", err)
	}
	return nil
}

// AppendRun persists a run as the next turn of an existing session, in one
// transaction. The session's options and tools are left unchanged.
func AppendRun(ctx context.Context, db *sql.DB, sessionID string, run Run) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin append run: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := appendRun(ctx, tx, sessionID, run); err != nil {
		return err
	}
	return tx.Commit()
}

// appendRun writes one turn (the user prompt plus its messages) inside an
// existing transaction, chaining them onto the session's leaf. The next turn
// number is the leaf's turn_number plus one, which makes a fork "replace" its
// fork turn naturally: the fork's leaf still sits at the end of turn atTurn-1,
// so its first run gets turn atTurn.
func appendRun(ctx context.Context, q DBTx, sessionID string, run Run) error {
	var leaf sql.NullString
	if err := q.QueryRowContext(ctx, `SELECT leaf_message_id FROM sessions WHERE id = ?`, sessionID).Scan(&leaf); err != nil {
		return fmt.Errorf("load session leaf: %w", err)
	}
	lastTurn := 0
	if leaf.Valid {
		var tn sql.NullInt64
		if err := q.QueryRowContext(ctx, `SELECT turn_number FROM messages WHERE id = ?`, leaf.String).Scan(&tn); err != nil {
			return fmt.Errorf("load leaf turn: %w", err)
		}
		if tn.Valid {
			lastTurn = int(tn.Int64)
		}
	}
	number := lastTurn + 1

	parent := ""
	if leaf.Valid {
		parent = leaf.String
	}

	userMsgID, userMsgNow := newIDWithTime()
	if err := insertMessage(ctx, q, userMsgID, parent, Message{
		Role:       "user",
		TurnNumber: number,
		CreatedAt:  userMsgNow,
		Parts:      []Part{{Type: "text", Text: run.Prompt}},
	}); err != nil {
		return fmt.Errorf("insert user message: %w", err)
	}
	lastID := userMsgID

	for _, m := range run.Messages {
		msgID, msgNow := newIDWithTime()
		m.CreatedAt = msgNow
		m.TurnNumber = number
		if err := insertMessage(ctx, q, msgID, lastID, m); err != nil {
			return err
		}
		lastID = msgID
	}

	if _, err := q.ExecContext(ctx, `UPDATE sessions SET leaf_message_id = ? WHERE id = ?`, lastID, sessionID); err != nil {
		return fmt.Errorf("update session leaf: %w", err)
	}

	return nil
}

func insertMessage(ctx context.Context, q DBTx, id, parentID string, m Message) error {
	if m.CreatedAt == "" {
		m.CreatedAt = time.Now().UTC().String()
	}
	var number any
	if m.Number > 0 {
		number = m.Number
	}
	var turnNumber any
	if m.TurnNumber > 0 {
		turnNumber = m.TurnNumber
	}
	args := []any{id, nullStr(parentID), m.Role, marshalParts(m.Parts), number, turnNumber, nullStr(m.FinishReason)}
	args = append(args, usageArgs(m.Usage)...)
	args = append(args, nullStr(m.ProviderOptions), m.CreatedAt)
	if _, err := q.ExecContext(ctx, `INSERT INTO messages
		(id, parent_id, role, content, number, turn_number, finish_reason,
		 input_tokens, output_tokens, total_tokens, reasoning_tokens, cache_read_tokens, cache_write_tokens,
		 provider_options, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		args...); err != nil {
		return fmt.Errorf("insert message: %w", err)
	}
	return nil
}

// usageArgs renders a message's usage for storage. A message that owns no usage
// (anything but the assistant message of a step) stores NULL in every usage
// column, so "no usage recorded" is distinguishable from "used zero tokens".
func usageArgs(u usage.TokenUsage) []any {
	if u == (usage.TokenUsage{}) {
		return []any{nil, nil, nil, nil, nil, nil}
	}
	return []any{u.InputTokens, u.OutputTokens, u.TotalTokens, u.ReasoningTokens,
		u.CacheReadTokens, u.CacheWriteTokens}
}

// ForkSession creates a new session (id newID) that shares the source session's
// transcript up to atTurn-1 by reference: no message rows are copied, and the
// fork's leaf_message_id points at the source's last message before the fork
// point. Walking from that leaf reaches back through the source's chain, so the
// fork's transcript is the shared prefix plus its own turns. The next turn
// appended to the fork gets number atTurn.
func ForkSession(ctx context.Context, db *sql.DB, sourceID, newID string, atTurn int) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin fork session: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var model, effort string
	var maxSteps int
	err = tx.QueryRowContext(ctx,
		`SELECT model, reasoning_effort, max_steps FROM options WHERE session_id = ?`, sourceID).
		Scan(&model, &effort, &maxSteps)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("session not found: %s", sourceID)
	}
	if err != nil {
		return fmt.Errorf("load source options: %w", err)
	}

	var srcLeaf sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT leaf_message_id FROM sessions WHERE id = ?`, sourceID).Scan(&srcLeaf); err != nil {
		return fmt.Errorf("load source leaf: %w", err)
	}
	path, err := chainPath(ctx, tx, srcLeaf.String)
	if err != nil {
		return err
	}
	if n := len(turnsFromPath(path)); atTurn < 1 || atTurn > n {
		return fmt.Errorf("turn %d not found in session %s", atTurn, sourceID)
	}

	// The fork's leaf is the deepest shared message: the last one (including the
	// system message) whose turn is before the fork point. For atTurn == 1 that
	// is the system message, or no message at all when the source has none.
	forkPoint := ""
	for _, m := range path {
		if m.TurnNumber <= 0 || m.TurnNumber < atTurn {
			forkPoint = m.ID
		}
	}

	now := timestampFromID(newID)
	var leaf any
	if forkPoint != "" {
		leaf = forkPoint
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO sessions (id, created_at, leaf_message_id) VALUES (?, ?, ?)`,
		newID, now, leaf); err != nil {
		return fmt.Errorf("insert forked session: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO options (session_id, model, reasoning_effort, max_steps)
		VALUES (?, ?, ?, ?)`, newID, model, effort, maxSteps); err != nil {
		return fmt.Errorf("insert forked options: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO session_tools (session_id, tool_id) SELECT ?, tool_id FROM session_tools WHERE session_id = ?`,
		newID, sourceID); err != nil {
		return fmt.Errorf("copy session tools: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO forks (session_id, parent_id, turn_number, created_at)
		VALUES (?, ?, ?, ?)`, newID, sourceID, atTurn, now); err != nil {
		return fmt.Errorf("insert fork lineage: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit fork session: %w", err)
	}
	return nil
}

// GetSessionTranscript reconstructs a full session: options, system prompt,
// tools, and every turn with its messages grouped into steps.
func GetSessionTranscript(ctx context.Context, db *sql.DB, sessionID string) (*Transcript, error) {
	sess, err := GetSession(ctx, db, sessionID)
	if err != nil {
		return nil, err
	}

	path, err := chainPath(ctx, db, sess.LeafMessageID)
	if err != nil {
		return nil, err
	}
	return &Transcript{Session: sess, Turns: turnsFromPath(path)}, nil
}

// chainPath walks parent_id from start (the session's leaf, "" when it has no
// messages) back to the chain root, returning the messages root-first.
func chainPath(ctx context.Context, q DBTx, start string) ([]Message, error) {
	if start == "" {
		return nil, nil
	}
	var rev []Message
	cur := start
	for cur != "" {
		m, err := getMessage(ctx, q, cur)
		if err != nil {
			return nil, err
		}
		rev = append(rev, m)
		cur = m.ParentID
	}
	out := make([]Message, len(rev))
	for i := range rev {
		out[i] = rev[len(rev)-1-i]
	}
	return out, nil
}

func getMessage(ctx context.Context, q DBTx, id string) (Message, error) {
	row := q.QueryRowContext(ctx, `
		SELECT id, parent_id, role, content, number, turn_number, finish_reason,
		       input_tokens, output_tokens, total_tokens, reasoning_tokens, cache_read_tokens, cache_write_tokens,
		       provider_options, created_at
		FROM messages WHERE id = ?`, id)
	m, err := scanMessage(row)
	if err != nil {
		return Message{}, fmt.Errorf("load message %s: %w", id, err)
	}
	return m, nil
}

// turnsFromPath groups a session's chain into turns by turn_number. The system
// message (turn_number 0) is excluded; each turn's Prompt is its first message
// (the user prompt).
func turnsFromPath(path []Message) []TurnTranscript {
	var out []TurnTranscript
	var msgs []Message
	turnNum := 0
	for _, m := range path {
		if m.TurnNumber <= 0 {
			continue
		}
		if len(msgs) > 0 && m.TurnNumber != turnNum {
			out = append(out, turnTranscript(turnNum, msgs))
			msgs = nil
		}
		msgs = append(msgs, m)
		turnNum = m.TurnNumber
	}
	if len(msgs) > 0 {
		out = append(out, turnTranscript(turnNum, msgs))
	}
	return out
}

func turnTranscript(num int, msgs []Message) TurnTranscript {
	tt := TurnTranscript{Turn: Turn{ID: msgs[0].ID, Number: num, CreatedAt: msgs[0].CreatedAt}}
	tt.Prompt = msgs[0]
	tt.Steps = groupSteps(msgs[1:])
	return tt
}

// groupSteps groups a turn's assistant/tool messages into steps by number: each
// assistant message opens a step; the tool messages that follow belong to it.
func groupSteps(msgs []Message) []StepTranscript {
	var steps []StepTranscript
	for _, m := range msgs {
		switch m.Role {
		case "assistant":
			steps = append(steps, StepTranscript{
				Step:     Step{Number: m.Number, FinishReason: m.FinishReason, Usage: m.Usage, CreatedAt: m.CreatedAt},
				Messages: []Message{m},
			})
		case "tool":
			if len(steps) == 0 {
				continue
			}
			steps[len(steps)-1].Messages = append(steps[len(steps)-1].Messages, m)
		}
	}
	return steps
}

// scanMessage reads one message row. Messages without usage (system, user,
// tool) store NULL in the usage columns; those scan to the zero Usage.
func scanMessage(row interface{ Scan(...any) error }) (Message, error) {
	var m Message
	var parentID, finishReason, providerOpts sql.NullString
	var number, turnNumber sql.NullInt64
	var inTok, outTok, totalTok, reasTok, cacheRead, cacheWrite sql.NullInt64
	var content string
	if err := row.Scan(&m.ID, &parentID, &m.Role, &content, &number, &turnNumber, &finishReason,
		&inTok, &outTok, &totalTok, &reasTok, &cacheRead, &cacheWrite,
		&providerOpts, &m.CreatedAt); err != nil {
		return Message{}, fmt.Errorf("scan message: %w", err)
	}
	m.ParentID = parentID.String
	m.FinishReason = finishReason.String
	m.ProviderOptions = providerOpts.String
	if number.Valid {
		m.Number = int(number.Int64)
	}
	if turnNumber.Valid {
		m.TurnNumber = int(turnNumber.Int64)
	}
	m.Usage = usage.TokenUsage{
		InputTokens:      int(inTok.Int64),
		OutputTokens:     int(outTok.Int64),
		TotalTokens:      int(totalTok.Int64),
		ReasoningTokens:  int(reasTok.Int64),
		CacheReadTokens:  int(cacheRead.Int64),
		CacheWriteTokens: int(cacheWrite.Int64),
	}
	m.Parts = unmarshalParts(content)
	return m, nil
}

// SearchHistory returns conversation text across sessions, newest first. Only
// text parts are returned; reasoning and tool parts are excluded. query matches
// turns whose text contains it (case-insensitive); limit bounds the number of
// turns returned.
func SearchHistory(ctx context.Context, db *sql.DB, sessionID, query string, limit int) ([]HistoryEntry, error) {
	if limit < 1 {
		return nil, fmt.Errorf("search history: limit must be positive")
	}
	query = strings.ToLower(query)

	sessions, err := ListSessions(ctx, db)
	if err != nil {
		return nil, err
	}

	var out []HistoryEntry
	seen := 0
	for _, s := range sessions {
		if sessionID != "" && s.ID != sessionID {
			continue
		}
		path, err := chainPath(ctx, db, s.LeafMessageID)
		if err != nil {
			return nil, err
		}
		turns := turnsFromPath(path)
		// Newest session first; within a session, turns newest first.
		for i := len(turns) - 1; i >= 0; i-- {
			tt := turns[i]
			var parts []HistoryEntry
			parts = append(parts, messageTextEntries(s, tt.Turn.Number, tt.Prompt)...)
			for _, st := range tt.Steps {
				for _, m := range st.Messages {
					parts = append(parts, messageTextEntries(s, tt.Turn.Number, m)...)
				}
			}
			matched := query == ""
			if !matched {
				for _, p := range parts {
					if strings.Contains(strings.ToLower(p.Text), query) {
						matched = true
						break
					}
				}
			}
			if matched {
				out = append(out, parts...)
				seen++
				if seen >= limit {
					return out, nil
				}
			}
		}
	}
	return out, nil
}

// messageTextEntries flattens one message's text parts into history entries.
func messageTextEntries(s Session, turnNumber int, m Message) []HistoryEntry {
	var out []HistoryEntry
	for _, p := range m.Parts {
		if p.Type == "text" {
			out = append(out, HistoryEntry{
				SessionID:        s.ID,
				SessionCreatedAt: s.CreatedAt,
				TurnNumber:       turnNumber,
				Role:             m.Role,
				Text:             p.Text,
			})
		}
	}
	return out
}

// DBTx is the subset of *sql.DB and *sql.Tx shared by every store operation,
// so single-operation writes and whole-run transactions reuse the same code.
type DBTx interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func upsertTool(ctx context.Context, q DBTx, t Tool) error {
	if t.ID == "" {
		t.ID = t.Name
	}
	if _, err := q.ExecContext(ctx, `INSERT INTO tools (id, name, description, input_schema, provider_defined_type)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			description = excluded.description,
			input_schema = excluded.input_schema,
			provider_defined_type = excluded.provider_defined_type`,
		t.ID, t.Name, t.Description, t.InputSchema, t.ProviderDefinedType); err != nil {
		return fmt.Errorf("upsert tool %s: %w", t.Name, err)
	}
	return nil
}

// marshalParts serializes a message's parts as a JSON array.
func marshalParts(parts []Part) string {
	if len(parts) == 0 {
		return "[]"
	}
	b, err := json.Marshal(parts)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// unmarshalParts parses a message's content JSON array.
func unmarshalParts(s string) []Part {
	var parts []Part
	if err := json.Unmarshal([]byte(s), &parts); err != nil {
		return nil
	}
	return parts
}

// partsText concatenates the text of a message's text parts.
func partsText(parts []Part) string {
	var b strings.Builder
	for _, p := range parts {
		if p.Type == "text" {
			b.WriteString(p.Text)
		}
	}
	return b.String()
}

func newIDWithTime() (id string, timestamp string) {
	u, err := uuid.NewV7()
	if err != nil {
		return "", time.Now().UTC().String()
	}
	return u.String(), unixMillisFromUUIDV7(u).UTC().String()
}

// timestampFromID decodes the UTC timestamp embedded in a UUIDv7 session id.
func timestampFromID(id string) string {
	u, err := uuid.Parse(id)
	if err != nil {
		return time.Now().UTC().String()
	}
	return unixMillisFromUUIDV7(u).UTC().String()
}

// unixMillisFromUUIDV7 decodes the Unix time in milliseconds that a UUIDv7 embeds in
// its first 48 bits (big-endian).
func unixMillisFromUUIDV7(u uuid.UUID) time.Time {
	ms := int64(u[0])<<40 | int64(u[1])<<32 | int64(u[2])<<24 |
		int64(u[3])<<16 | int64(u[4])<<8 | int64(u[5])
	return time.UnixMilli(ms)
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
