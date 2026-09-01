-- schema.sql — storage for agent sessions.
--
-- Data model overview
-- -------------------
--   sessions   one conversation; identity (id + created_at) plus a
--              leaf_message_id anchor into the shared message chain
--   forks      lineage: which session a session branched from, and where
--   options    the run options fixed for a session (model, effort, max steps)
--   tools      registered tool definitions (reusable)
--   session_tools  M:N join of sessions and tools
--   messages   every transcript entry (system/user/assistant/tool), stored
--              once and shared. Messages form a chain via parent_id; a session
--              owns exactly one root->leaf path through that chain, anchored by
--              its leaf_message_id. Content is a JSON array of parts; turn_number
--              is the turn a message belongs to (NULL on the session-wide system
--              message); `number` is the tool-loop step within the turn.
--   scheduler / recurrences  scheduled runs
--
-- Forks reference, not copy: a fork's transcript is the walk from its leaf
-- back through parent_id, which reaches into the parent session's chain.
-- Reconstruction: walk parent_id from the session's leaf_message_id to the
-- root and reverse; group by turn_number (system message excluded) into turns,
-- and within a turn each assistant message starts a step (messages.number) that
-- its tool messages share.

PRAGMA journal_mode=WAL;
PRAGMA synchronous=NORMAL;
PRAGMA foreign_keys=ON;
PRAGMA temp_store=MEMORY;
PRAGMA busy_timeout=5000;

-- ----------------------------------------------------------------------
-- sessions: a conversation. Identity and timestamp plus the leaf anchor into
-- the shared message chain: the session's newest message. Its transcript is
-- everything reachable by walking parent_id back from the leaf. leaf_message_id
-- is NULL until the session has its first message.
-- ----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS sessions (
    id              TEXT PRIMARY KEY,   -- UUIDv7; its embedded timestamp is created_at
    created_at      TEXT NOT NULL,
    leaf_message_id TEXT REFERENCES messages(id) ON DELETE RESTRICT
) STRICT;

-- ----------------------------------------------------------------------
-- forks: lineage. session_id is the fork (child); parent_id is where it
-- branched from; turn_number is the source turn the fork replaced. One row
-- per branch (a session has at most one parent). parent_id is nullable so
-- the fork's lineage row survives its parent's deletion; only the parent
-- reference is cleared (the fork's transcript is unaffected, since it walks
-- the shared message chain, not the parent session row).
-- ----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS forks (
    session_id  TEXT PRIMARY KEY REFERENCES sessions(id) ON DELETE CASCADE,
    parent_id   TEXT REFERENCES sessions(id) ON DELETE SET NULL,
    turn_number INTEGER NOT NULL,
    created_at  TEXT NOT NULL
) STRICT;

-- ----------------------------------------------------------------------
-- options: the run options fixed for a session.
-- ----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS options (
    session_id       TEXT PRIMARY KEY REFERENCES sessions(id) ON DELETE CASCADE,
    model            TEXT NOT NULL,
    reasoning_effort TEXT NOT NULL,
    max_steps        INTEGER NOT NULL
) STRICT;

-- ----------------------------------------------------------------------
-- tools: a registered tool definition. Identified by name; the input schema
-- is stored as JSON text. Referenced by session_tools.
-- ----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS tools (
    id                    TEXT PRIMARY KEY,
    name                  TEXT NOT NULL UNIQUE,
    description           TEXT,
    input_schema          TEXT,
    provider_defined_type TEXT
) STRICT;

-- ----------------------------------------------------------------------
-- session_tools: which tools are available in a session.
-- ----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS session_tools (
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    tool_id    TEXT NOT NULL REFERENCES tools(id),
    PRIMARY KEY (session_id, tool_id)
) STRICT;

-- ----------------------------------------------------------------------
-- messages: transcript entries, stored once and shared across sessions. Roles:
-- system | user | assistant | tool.
--   system    : the chain root of a session (parent_id NULL, turn_number NULL)
--   user      : a turn's prompt (turn_number set)
--   assistant : a step's result (turn_number set, number set to the step)
--   tool      : a tool result for the step's call (turn_number set, number set)
-- parent_id links each message to its predecessor on the chain; the root has
-- parent_id NULL. ON DELETE RESTRICT makes mid-chain deletion impossible: a
-- message can only be removed once nothing (as a parent or as a session leaf)
-- references it, so transcripts cannot be severed. content is a JSON array of
-- parts (text, reasoning, tool-call, tool-result, file, ...); tool
-- name/input/output/error live on the parts themselves.
-- Token usage is per-message, recorded only on the assistant message of a step
-- (the provider reports usage per generation, not per message); all other
-- messages store NULL, so a row of NULLs means "no usage recorded here", not
-- "used zero tokens".
-- ----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS messages (
    id                 TEXT PRIMARY KEY,
    parent_id          TEXT REFERENCES messages(id) ON DELETE RESTRICT,
    role               TEXT NOT NULL,
    content            TEXT NOT NULL,
    number             INTEGER,
    turn_number        INTEGER,
    finish_reason      TEXT,
    input_tokens       INTEGER,
    output_tokens      INTEGER,
    total_tokens       INTEGER,
    reasoning_tokens   INTEGER,
    cache_read_tokens  INTEGER,
    cache_write_tokens INTEGER,
    provider_options   TEXT,
    created_at         TEXT NOT NULL
) STRICT;

-- ----------------------------------------------------------------------
-- recurrences: a fixed-interval repeating schedule. Each occurrence is its own
-- row in scheduler (linked by recurrence_id), so delivery keeps the one-shot
-- exactly-once machinery: the schedule advances when an occurrence fires or is
-- skipped, never by mutating a fired row in place. status is active until
-- cancelled; cancelling a recurrence also cancels its pending next occurrence.
-- ----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS recurrences (
    id          TEXT    PRIMARY KEY,
    subject     TEXT    NOT NULL,
    payload     TEXT    NOT NULL,
    interval_ms INTEGER NOT NULL,
    created_at  TEXT    NOT NULL,
    status      TEXT    NOT NULL DEFAULT 'active' CHECK (status IN ('active','cancelled'))
) STRICT;

-- ----------------------------------------------------------------------
-- scheduler: one-time durable messages. A message is published (as JSON) on
-- its subject once its publish_at deadline passes; a message found far past
-- its deadline is expired instead, its id announced on scheduler.expired (or,
-- for a recurring occurrence, skipped and announced on scheduler.recurring.
-- missed). created_at mirrors the timestamp embedded in the UUIDv7 id (when
-- the message was scheduled); publish_at is the caller-supplied target publish
-- time (unix ms, typically in the future); published_at is written after the
-- payload is actually sent, so fired rows with a NULL published_at can be
-- replayed for at-least-once delivery. status transitions pending -> fired
-- (claimed for delivery), expired, or cancelled (explicitly withdrawn before
-- it fired). recurrence_id links an occurrence to its parent recurrence; it is
-- NULL for one-shot messages.
-- ----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS scheduler (
    id            TEXT    PRIMARY KEY,
    subject       TEXT    NOT NULL,
    payload       TEXT    NOT NULL,
    created_at    TEXT    NOT NULL,
    publish_at    INTEGER NOT NULL,
    status        TEXT    NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','fired','expired','cancelled')),
    published_at  TEXT,
    recurrence_id TEXT    REFERENCES recurrences(id)
) STRICT;

CREATE INDEX IF NOT EXISTS idx_scheduler_pending  ON scheduler(publish_at)   WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_scheduler_unmarked ON scheduler(published_at) WHERE status = 'fired' AND published_at IS NULL;

-- ----------------------------------------------------------------------
-- tasks: one (codebase, task) -> commit pipeline run. status advances
-- created -> started -> completed -> reviewed as the pipeline progresses;
-- nothing here enforces that order, the orchestrator does. session_id /
-- review_session_id reference sessions(id) once the pipeline creates them
-- (the execution session and, separately, the adversarial-review session);
-- both are nullable because a freshly created task has neither yet.
-- Slice-valued fields (tags, dependencies, "where", invariants, packages,
-- completed_when) are stored as JSON arrays. Token usage is flattened into
-- columns the same way messages.* stores it: NULL means "not recorded yet",
-- not "zero tokens" (see usage.TokenUsage). Timestamps are RFC3339Nano —
-- deliberately not the sessions table's Go time.String() convention, since
-- nothing decodes a task's timing from its id the way UUIDv7 session ids
-- are decoded, so a standard, unambiguous format is worth more here than
-- matching that table's incidental format.
-- ----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS tasks (
    id                 TEXT    PRIMARY KEY,   -- UUIDv7; its embedded timestamp is created_at
    title              TEXT    NOT NULL,
    status             TEXT    NOT NULL DEFAULT 'created' CHECK (status IN ('created','started','completed','reviewed')),
    task_type          TEXT    NOT NULL,
    tags               TEXT,
    dependencies       TEXT,
    what               TEXT    NOT NULL,
    "where"            TEXT,
    why                TEXT    NOT NULL,
    how                TEXT    NOT NULL,
    invariants         TEXT,
    urgency            TEXT    NOT NULL CHECK (urgency IN ('low','medium','high')),
    importance         TEXT    NOT NULL CHECK (importance IN ('low','medium','high')),
    risk               TEXT    CHECK (risk IS NULL OR risk IN ('low','medium','high')),
    packages           TEXT    NOT NULL,
    completed_when     TEXT    NOT NULL,
    commit_type        TEXT,
    commit_message     TEXT,
    commit_hash        TEXT,
    model              TEXT,
    variant            TEXT,
    session_id         TEXT    REFERENCES sessions(id) ON DELETE SET NULL,
    review_session_id  TEXT    REFERENCES sessions(id) ON DELETE SET NULL,
    tokens_input       INTEGER,
    tokens_output      INTEGER,
    tokens_total       INTEGER,
    tokens_reasoning   INTEGER,
    tokens_cache_read  INTEGER,
    tokens_cache_write INTEGER,
    created_at         TEXT    NOT NULL,
    started_at         TEXT,
    completed_at       TEXT,
    reviewed_at        TEXT
) STRICT;

CREATE INDEX IF NOT EXISTS idx_tasks_status     ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks(created_at);

-- ----------------------------------------------------------------------
-- Indexes for the common reconstruction and lookup paths.
-- ----------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_sessions_created_at ON sessions(created_at);
CREATE INDEX IF NOT EXISTS idx_messages_parent     ON messages(parent_id);
CREATE INDEX IF NOT EXISTS idx_session_tools_tool  ON session_tools(tool_id);
