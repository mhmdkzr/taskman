// Package task wraps internal/task's SQLite-backed backlog as goai.Tool
// values: an agent can file a new task, search the backlog, read one in
// full, or edit its spec while it's still unstarted. It does not expose the
// lifecycle transitions (start/complete/review) — those are orchestrator-
// driven, not agent-discretionary, so a task's own execution can't change
// its status out from under the pipeline running it.
package task

import (
	"database/sql"

	"github.com/zendev-sh/goai"
)

// Tools returns the full backlog-management tool set bound to db.
func Tools(db *sql.DB) []goai.Tool {
	return []goai.Tool{
		CreateTool(db),
		SearchTool(db),
		GetTool(db),
		EditTool(db),
	}
}
