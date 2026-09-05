# Agent

The agent package resolves configured tools and manages the tool definitions stored in SQLite.

`Seed` upserts the provider names supported by GoAI plus the configured
`opencode` OpenAI-compatible endpoint, attaches its configured base URL and
API-key environment-variable name, upserts the configured model, and
then upserts the name, description, and generated input/output JSON Schemas for
every tool registered by `Tools`. Application startup runs this seed after
database migrations, so provider and tool records stay synchronized with
configuration and compiled definitions.
