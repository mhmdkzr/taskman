-- SQLite Pragmas
PRAGMA journal_mode=WAL;
PRAGMA synchronous=NORMAL;
PRAGMA foreign_keys=ON;
PRAGMA temp_store=MEMORY;
PRAGMA busy_timeout=5000;

--
CREATE TABLE IF NOT EXISTS model_providers (
    provider_id   TEXT PRIMARY KEY,
    provider_name TEXT NOT NULL UNIQUE,
    base_url      TEXT,
    api_key_env   TEXT,

    CONSTRAINT model_providers_provider_length_check
        CHECK (length(provider_name) <= 255)
) STRICT;

CREATE TABLE IF NOT EXISTS models (
    model_id       TEXT    PRIMARY KEY,
    provider_id    TEXT    NOT NULL REFERENCES model_providers(provider_id) ON DELETE RESTRICT,
    model_name     TEXT    NOT NULL,
    context_window INTEGER NOT NULL,
    has_vision     INTEGER NOT NULL CHECK (has_vision IN (0, 1)),
    thinking_options TEXT,

    CONSTRAINT models_model_length_check
        CHECK (length(model_name) <= 255),

    CONSTRAINT models_provider_model_unique
        UNIQUE (provider_id, model_name),

    CONSTRAINT models_thinking_options_json_check
        CHECK (thinking_options IS NULL OR json_valid(thinking_options))
) STRICT;

--
CREATE TABLE IF NOT EXISTS tools (
    tool_id          TEXT  PRIMARY KEY,
    tool_name        TEXT  NOT NULL,
    tool_description TEXT  NOT NULL,
    input_schema     TEXT  NOT NULL,
    output_schema    TEXT  NOT NULL,

    CONSTRAINT tools_name_length_check
        CHECK (length(tool_name) <= 255),

    CONSTRAINT tools_description_length_check
        CHECK (length(tool_description) <= 1024),

    CONSTRAINT tools_input_schema_json_check
        CHECK (json_valid(input_schema)),

    CONSTRAINT tools_output_schema_json_check
        CHECK (json_valid(output_schema))
) STRICT;

CREATE UNIQUE INDEX IF NOT EXISTS uniq_tools_name ON tools (tool_name);

--
CREATE TABLE IF NOT EXISTS prompt_templates (
    prompt_id     TEXT    PRIMARY KEY,
    prompt_name   TEXT    NOT NULL,
    template_body TEXT    NOT NULL,
    params_schema TEXT    NOT NULL,
    version       INTEGER NOT NULL,

    CONSTRAINT prompt_templates_name_length_check
        CHECK (length(prompt_name) <= 255),

    CONSTRAINT prompt_templates_version_check
        CHECK (version > 0),

    CONSTRAINT prompt_templates_params_schema_json_check
        CHECK (json_valid(params_schema))
) STRICT;

--
CREATE TABLE IF NOT EXISTS agents (
    agent_id   TEXT  PRIMARY KEY,
    agent_name TEXT  NOT NULL,
    prompt_id  TEXT  NOT NULL REFERENCES prompt_templates(prompt_id) ON DELETE RESTRICT,
    model_id   TEXT  NOT NULL REFERENCES models(model_id) ON DELETE RESTRICT,

    CONSTRAINT agents_name_length_check
        CHECK (length(agent_name) <= 255)
) STRICT;

CREATE UNIQUE INDEX IF NOT EXISTS uniq_agents_name
    ON agents (agent_name);

CREATE TABLE IF NOT EXISTS agent_tools (
    agent_id TEXT NOT NULL REFERENCES agents(agent_id) ON DELETE RESTRICT,
    tool_id  TEXT NOT NULL REFERENCES tools(tool_id) ON DELETE RESTRICT,

    PRIMARY KEY (agent_id, tool_id)
) STRICT;

--
-- Every session runs as some agent: agent_id supplies the session's model
-- and system prompt (rendered from the agent's prompt template) at creation
-- time. parent_session_id links a dispatched subagent's session back to the
-- session that spawned it; it is NULL for a top-level, user-initiated session.
CREATE TABLE IF NOT EXISTS agent_sessions (
    session_id        TEXT        PRIMARY KEY,
    session_title     TEXT,
    agent_id          TEXT        NOT NULL REFERENCES agents(agent_id) ON DELETE RESTRICT,
    parent_session_id TEXT        REFERENCES agent_sessions(session_id) ON DELETE RESTRICT,
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
    ON agent_sessions (agent_id);

CREATE INDEX IF NOT EXISTS idx_agent_sessions_parent
    ON agent_sessions (parent_session_id) WHERE parent_session_id IS NOT NULL;

-- status tracks a turn's crash-recovery lifecycle: a turn is inserted as
-- 'running' before its model/tool-call loop starts (the write-ahead marker)
-- and updated to 'completed' once that loop returns normally. A row still
-- 'running' after a process restart was orphaned by a crash mid-turn; boot-time
-- reconciliation (see sessions.ReconcileInterrupted) closes it out as
-- 'interrupted', synthesizing a valid result from session_turn_events so the
-- stored conversation never has a dangling tool call. result is NULL only
-- while status='running'.
CREATE TABLE IF NOT EXISTS session_turns (
    turn_id      TEXT        PRIMARY KEY,
    session_id   TEXT        NOT NULL REFERENCES agent_sessions(session_id) ON DELETE RESTRICT,
    prompt       TEXT        NOT NULL,
    result       TEXT,
    status       TEXT        NOT NULL DEFAULT 'completed'
                              CHECK (status IN ('running', 'completed', 'interrupted')),
    created_at   TEXT        NOT NULL,
    completed_at TEXT,

    CONSTRAINT session_turns_result_json_check
        CHECK (result IS NULL OR json_valid(result)),

    CONSTRAINT session_turns_result_status_check
        CHECK ((status = 'running') = (result IS NULL))
) STRICT;

CREATE INDEX IF NOT EXISTS idx_session_turns_session
    ON session_turns (session_id, created_at);

CREATE INDEX IF NOT EXISTS idx_session_turns_status
    ON session_turns (status) WHERE status = 'running';

-- Append-only log of tool-call lifecycle events within a still-running turn,
-- written incrementally (via goai hooks) as each step/tool call happens - not
-- batched at turn-end. This is what lets boot-time reconciliation recover a
-- turn orphaned by a mid-turn crash: it replays these events into a valid
-- partial conversation instead of discarding the whole turn. seq orders events
-- within a turn (assigned by an in-process counter, since concurrent tool
-- calls in the same step can otherwise write out of logical order).
CREATE TABLE IF NOT EXISTS session_turn_events (
    event_id   TEXT    NOT NULL PRIMARY KEY,
    turn_id    TEXT    NOT NULL REFERENCES session_turns(turn_id) ON DELETE RESTRICT,
    seq        INTEGER NOT NULL,
    event_type TEXT    NOT NULL CHECK (event_type IN ('step_finish', 'tool_call_start', 'tool_call_result')),
    payload    TEXT    NOT NULL,
    created_at TEXT    NOT NULL,

    CONSTRAINT session_turn_events_payload_json_check
        CHECK (json_valid(payload))
) STRICT;

CREATE INDEX IF NOT EXISTS idx_session_turn_events_turn
    ON session_turn_events (turn_id, seq);

CREATE TABLE IF NOT EXISTS token_usage (
    session_id         TEXT    NOT NULL REFERENCES agent_sessions(session_id) ON DELETE RESTRICT,
    turn_id            TEXT    NOT NULL REFERENCES session_turns(turn_id)     ON DELETE RESTRICT,
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
    session_id TEXT NOT NULL REFERENCES agent_sessions(session_id) ON DELETE RESTRICT,
    tool_id    TEXT NOT NULL REFERENCES tools(tool_id) ON DELETE RESTRICT,

    PRIMARY KEY (session_id, tool_id)
) STRICT;

--
CREATE TABLE IF NOT EXISTS session_todos (
    todo_id    TEXT        PRIMARY KEY,
    session_id TEXT        NOT NULL REFERENCES agent_sessions(session_id) ON DELETE RESTRICT,
    content    TEXT        NOT NULL,
    status     TEXT        NOT NULL CHECK (status IN ('pending', 'in_progress', 'completed', 'cancelled')),
    priority   TEXT        NOT NULL CHECK (priority IN ('high', 'medium', 'low')),
    position   INTEGER     NOT NULL,
    created_at TEXT        NOT NULL,
    updated_at TEXT,

    CONSTRAINT session_todos_content_check
        CHECK (length(content) > 0)
) STRICT;

CREATE INDEX IF NOT EXISTS idx_session_todos_session
    ON session_todos (session_id, position);

--
CREATE TABLE IF NOT EXISTS tasks (
    id            TEXT PRIMARY KEY,
    definition    TEXT NOT NULL,
    specification TEXT NOT NULL,
    state         TEXT NOT NULL,
    importance    INTEGER NOT NULL CHECK (importance BETWEEN 1 AND 5),
    urgency       INTEGER NOT NULL CHECK (urgency BETWEEN 1 AND 5),
    complexity    INTEGER NOT NULL CHECK (complexity BETWEEN 1 AND 5),
    effort        INTEGER NOT NULL CHECK (effort BETWEEN 1 AND 5),
    risk          INTEGER NOT NULL CHECK (risk BETWEEN 1 AND 5),
    autonomy      INTEGER NOT NULL CHECK (autonomy BETWEEN 1 AND 5),
    model         TEXT NOT NULL,
    commit_hash   TEXT NOT NULL,
    created_at    TEXT NOT NULL,
    updated_at    TEXT,
    deleted_at    TEXT
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
    session_id    TEXT NOT NULL REFERENCES agent_sessions(session_id) ON DELETE RESTRICT,

    PRIMARY KEY (task_id, session_id)
) STRICT;
