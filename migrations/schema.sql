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

CREATE TABLE IF NOT EXISTS session_turns (
    turn_id    TEXT        PRIMARY KEY,
    session_id TEXT        NOT NULL REFERENCES agent_sessions(session_id) ON DELETE RESTRICT,
    prompt     TEXT        NOT NULL,
    result     TEXT        NOT NULL,
    created_at TEXT        NOT NULL,

    CONSTRAINT session_turns_result_json_check
        CHECK (json_valid(result))
) STRICT;

CREATE INDEX IF NOT EXISTS idx_session_turns_session
    ON session_turns (session_id, created_at);

CREATE TABLE IF NOT EXISTS session_todos (
    todo_id    TEXT        PRIMARY KEY,
    session_id TEXT        NOT NULL REFERENCES agent_sessions(session_id) ON DELETE RESTRICT,
    content    TEXT        NOT NULL,
    status     TEXT        NOT NULL CHECK (status IN ('pending', 'in_progress', 'completed', 'cancelled')),
    priority   TEXT        NOT NULL CHECK (priority IN ('high', 'medium', 'low')),
    position   INTEGER     NOT NULL,
    created_at TEXT        NOT NULL,
    updated_at TEXT        NOT NULL,

    CONSTRAINT session_todos_content_check
        CHECK (length(content) > 0)
) STRICT;

CREATE INDEX IF NOT EXISTS idx_session_todos_session
    ON session_todos (session_id, position);

CREATE TABLE IF NOT EXISTS session_tools (
    session_id TEXT NOT NULL REFERENCES agent_sessions(session_id) ON DELETE RESTRICT,
    tool_id    TEXT NOT NULL REFERENCES tools(tool_id) ON DELETE RESTRICT,

    PRIMARY KEY (session_id, tool_id)
) STRICT;

--
CREATE TABLE IF NOT EXISTS tasks (
    task_id    TEXT        PRIMARY KEY,
    task_name  TEXT        NOT NULL,
    task       TEXT        NOT NULL,
    status     TEXT        NOT NULL CHECK (status IN ('backlog', 'queued', 'started', 'completed', 'cancelled')),
    created_at TEXT        NOT NULL,
    updated_at TEXT,
    deleted_at TEXT,

    CONSTRAINT tasks_name_check
        CHECK (length(task_name) > 0 AND length(task_name) <= 255),

    CONSTRAINT tasks_task_json_check
        CHECK (json_valid(task))
) STRICT;

CREATE INDEX IF NOT EXISTS idx_tasks_status_created
    ON tasks (status, created_at DESC) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS task_sessions (
    task_id    TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    session_id TEXT NOT NULL REFERENCES agent_sessions(session_id) ON DELETE RESTRICT,

    PRIMARY KEY (task_id, session_id)
) STRICT;

CREATE INDEX IF NOT EXISTS idx_task_sessions_session
    ON task_sessions (session_id);
