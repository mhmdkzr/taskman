-- SQLite Pragmas
PRAGMA journal_mode=WAL;
PRAGMA synchronous=NORMAL;
PRAGMA foreign_keys=ON;
PRAGMA temp_store=MEMORY;
PRAGMA busy_timeout=5000;

--
-- Every session runs as some agent: agent_id supplies the session's model
-- and system prompt (rendered from the agent's prompt template) at creation
-- time. parent_session_id links a dispatched subagent's session back to the
-- session that spawned it; it is NULL for a top-level, user-initiated session.
CREATE TABLE IF NOT EXISTS sessions (
    session_id        TEXT        PRIMARY KEY,
    session_title     TEXT,
    agent_id          TEXT        NOT NULL REFERENCES agents(agent_id) ON DELETE RESTRICT,
    parent_session_id TEXT        REFERENCES sessions(session_id) ON DELETE RESTRICT,
    model_id          TEXT        NOT NULL REFERENCES models(model_id) ON DELETE RESTRICT,
    system_prompt     TEXT        NOT NULL,
    provider_options  TEXT,
    created_at        TEXT        NOT NULL,
    deleted_at        TEXT,

    CONSTRAINT agent_sessions_title_check
        CHECK (session_title IS NULL OR (length(session_title) > 0 AND length(session_title) <= 255)),

    CONSTRAINT agent_sessions_provider_options_json_check
        CHECK (provider_options IS NULL OR json_valid(provider_options))
) STRICT;

CREATE INDEX IF NOT EXISTS idx_agent_sessions_agent
    ON sessions (agent_id);

CREATE INDEX IF NOT EXISTS idx_agent_sessions_parent
    ON sessions (parent_session_id) WHERE parent_session_id IS NOT NULL;

-- status tracks a turn's crash-recovery lifecycle: a turn is inserted as
-- 'running' before its model/tool-call loop starts (the write-ahead marker)
-- and updated to 'completed' once that loop returns normally. A row still
-- 'running' after a process restart was orphaned by a crash mid-turn; boot-time
-- reconciliation (see sessions.ReconcileInterrupted) closes it out as
-- 'interrupted', synthesizing a valid result from events so the
-- stored conversation never has a dangling tool call. result is NULL only
-- while status='running'.
CREATE TABLE IF NOT EXISTS turns (
    turn_id      TEXT        PRIMARY KEY,
    session_id   TEXT        NOT NULL REFERENCES sessions(session_id) ON DELETE RESTRICT,
    prompt       TEXT        NOT NULL,
    result       TEXT,
    status       TEXT        NOT NULL DEFAULT 'completed'
                              CHECK (status IN ('running', 'completed', 'interrupted')),
    created_at   TEXT        NOT NULL,
    completed_at TEXT,

    CONSTRAINT turns_result_json_check
        CHECK (result IS NULL OR json_valid(result)),

    CONSTRAINT turns_result_status_check
        CHECK ((status = 'running') = (result IS NULL))
) STRICT;

CREATE INDEX IF NOT EXISTS turns_session
    ON turns (session_id, created_at);

CREATE INDEX IF NOT EXISTS turns_status
    ON turns (status) WHERE status = 'running';

-- Append-only log of tool-call lifecycle events within a still-running turn,
-- written incrementally (via goai hooks) as each step/tool call happens - not
-- batched at turn-end. This is what lets boot-time reconciliation recover a
-- turn orphaned by a mid-turn crash: it replays these events into a valid
-- partial conversation instead of discarding the whole turn. seq orders events
-- within a turn (assigned by an in-process counter, since concurrent tool
-- calls in the same step can otherwise write out of logical order).
CREATE TABLE IF NOT EXISTS events (
    event_id   TEXT    NOT NULL PRIMARY KEY,
    turn_id    TEXT    NOT NULL REFERENCES turns(turn_id) ON DELETE RESTRICT,
    seq        INTEGER NOT NULL,
    event_type TEXT    NOT NULL CHECK (event_type IN ('step_finish', 'tool_call_start', 'tool_call_result')),
    payload    TEXT    NOT NULL,
    created_at TEXT    NOT NULL,

    CONSTRAINT session_turn_events_payload_json_check
        CHECK (json_valid(payload))
) STRICT;

CREATE INDEX IF NOT EXISTS idx_session_turn_events_turn
    ON events (turn_id, seq);

-- A row is inserted by the ask tool when the model asks the user a question
-- mid-turn, blocking that tool call until answered. status is 'pending'
-- until a human answers via the web UI, which sets selected/custom and moves
-- it to 'answered' - the polling tool call is watching for exactly that
-- transition (see internal/agent/tools/ask).
CREATE TABLE IF NOT EXISTS session_asks (
    ask_id       TEXT    NOT NULL PRIMARY KEY,
    session_id   TEXT    NOT NULL REFERENCES sessions(session_id) ON DELETE RESTRICT,
    turn_id      TEXT    NOT NULL REFERENCES turns(turn_id)     ON DELETE RESTRICT,
    question     TEXT    NOT NULL,
    options      TEXT,
    multi_select INTEGER NOT NULL CHECK (multi_select IN (0, 1)),
    status       TEXT    NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'answered')),
    selected     TEXT,
    custom       TEXT,
    created_at   TEXT    NOT NULL,
    answered_at  TEXT,

    CONSTRAINT session_asks_options_json_check
        CHECK (options IS NULL OR json_valid(options)),

    CONSTRAINT session_asks_selected_json_check
        CHECK (selected IS NULL OR json_valid(selected))
) STRICT;

CREATE INDEX IF NOT EXISTS idx_session_asks_turn
    ON session_asks (turn_id);

CREATE INDEX IF NOT EXISTS idx_session_asks_pending
    ON session_asks (status) WHERE status = 'pending';

CREATE TABLE IF NOT EXISTS token_usage (
    session_id         TEXT    NOT NULL REFERENCES sessions(session_id) ON DELETE RESTRICT,
    turn_id            TEXT    NOT NULL REFERENCES turns(turn_id)     ON DELETE RESTRICT,
    input_tokens       INTEGER NOT NULL,
    output_tokens      INTEGER NOT NULL,
    total_tokens       INTEGER NOT NULL,
    reasoning_tokens   INTEGER NOT NULL,
    cache_read_tokens  INTEGER NOT NULL,
    cache_write_tokens INTEGER NOT NULL,
    created_at         TEXT    NOT NULL,

    PRIMARY KEY (session_id, turn_id)
) STRICT;

CREATE INDEX IF NOT EXISTS idx_token_usage_session
    ON token_usage (session_id, created_at);

CREATE INDEX IF NOT EXISTS idx_token_usage_turn
    ON token_usage (turn_id);

CREATE TABLE IF NOT EXISTS session_tools (
    session_id TEXT NOT NULL REFERENCES sessions(session_id) ON DELETE RESTRICT,
    tool_id    TEXT NOT NULL REFERENCES tools(tool_id) ON DELETE RESTRICT,

    PRIMARY KEY (session_id, tool_id)
) STRICT;

--
CREATE TABLE IF NOT EXISTS tasks (
    id               TEXT PRIMARY KEY,
    definition       TEXT NOT NULL,
    specification    TEXT NOT NULL,
    state            TEXT NOT NULL,
    importance       INTEGER NOT NULL CHECK (importance BETWEEN 1 AND 5),
    urgency          INTEGER NOT NULL CHECK (urgency BETWEEN 1 AND 5),
    complexity       INTEGER NOT NULL CHECK (complexity BETWEEN 1 AND 5),
    effort           INTEGER NOT NULL CHECK (effort BETWEEN 1 AND 5),
    risk             INTEGER NOT NULL CHECK (risk BETWEEN 1 AND 5),
    autonomy         INTEGER NOT NULL CHECK (autonomy BETWEEN 1 AND 5),
    model            TEXT NOT NULL,
    reasoning_effort TEXT,
    commit_hash      TEXT NOT NULL,
    branch           TEXT,
    failure_reason   TEXT,
    pipeline_step    TEXT NOT NULL DEFAULT '',
    created_at       TEXT NOT NULL,
    updated_at       TEXT,
    deleted_at       TEXT
) STRICT;

CREATE INDEX IF NOT EXISTS idx_tasks_state
    ON tasks (state);

CREATE TABLE IF NOT EXISTS tasks_labels (
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE RESTRICT,
    label   TEXT NOT NULL,

    PRIMARY KEY (task_id, label)
) STRICT;

CREATE TABLE IF NOT EXISTS tasks_sessions (
    task_id       TEXT NOT NULL REFERENCES tasks(id) ON DELETE RESTRICT,
    session_id    TEXT NOT NULL REFERENCES sessions(session_id) ON DELETE RESTRICT,

    PRIMARY KEY (task_id, session_id)
) STRICT;

-- One row per review attempt made against a task by the pipeline's
-- pipeline-review agent - an append-only log, never updated in place, so a
-- crash between a review finishing and the next pipeline stage never loses
-- what the review found (see internal/pipeline). attempt orders attempts
-- for a task starting at 1.
CREATE TABLE IF NOT EXISTS task_review_results (
    id          TEXT    PRIMARY KEY,
    task_id     TEXT    NOT NULL REFERENCES tasks(id) ON DELETE RESTRICT,
    session_id  TEXT    REFERENCES sessions(session_id) ON DELETE RESTRICT,
    attempt     INTEGER NOT NULL CHECK (attempt > 0),
    approved    INTEGER NOT NULL CHECK (approved IN (0, 1)),
    findings    TEXT    NOT NULL,
    created_at  TEXT    NOT NULL,

    CONSTRAINT task_review_results_findings_json_check
        CHECK (json_valid(findings)),

    CONSTRAINT task_review_results_task_attempt_unique
        UNIQUE (task_id, attempt)
) STRICT;

CREATE INDEX IF NOT EXISTS idx_task_review_results_task
    ON task_review_results (task_id, attempt);

-- Maps a GitHub issue to the task the pipeline created for it, so a later
-- Start call resumes that existing task (see internal/pipeline.Start)
-- instead of creating a duplicate for the same still-open, still-labeled
-- issue. One row per issue the pipeline has ever picked up.
CREATE TABLE IF NOT EXISTS pipeline_issue_tasks (
    owner        TEXT    NOT NULL,
    repo         TEXT    NOT NULL,
    issue_number INTEGER NOT NULL,
    task_id      TEXT    NOT NULL REFERENCES tasks(id) ON DELETE RESTRICT,

    PRIMARY KEY (owner, repo, issue_number)
) STRICT;
