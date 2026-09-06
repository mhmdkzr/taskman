package agent

import (
	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
	agenttool "github.com/mhmdkzr/loop/internal/agent/tools/agent"
	"github.com/mhmdkzr/loop/internal/agent/tools/bash"
	"github.com/mhmdkzr/loop/internal/agent/tools/browser"
	"github.com/mhmdkzr/loop/internal/agent/tools/browser/back"
	"github.com/mhmdkzr/loop/internal/agent/tools/browser/click"
	"github.com/mhmdkzr/loop/internal/agent/tools/browser/extract"
	"github.com/mhmdkzr/loop/internal/agent/tools/browser/fill"
	"github.com/mhmdkzr/loop/internal/agent/tools/browser/forward"
	"github.com/mhmdkzr/loop/internal/agent/tools/browser/navigate"
	"github.com/mhmdkzr/loop/internal/agent/tools/browser/reset"
	"github.com/mhmdkzr/loop/internal/agent/tools/browser/scroll"
	"github.com/mhmdkzr/loop/internal/agent/tools/curl"
	"github.com/mhmdkzr/loop/internal/agent/tools/datetime"
	"github.com/mhmdkzr/loop/internal/agent/tools/deno"
	"github.com/mhmdkzr/loop/internal/agent/tools/files/edit"
	"github.com/mhmdkzr/loop/internal/agent/tools/files/glob"
	"github.com/mhmdkzr/loop/internal/agent/tools/files/grep"
	"github.com/mhmdkzr/loop/internal/agent/tools/files/patch"
	"github.com/mhmdkzr/loop/internal/agent/tools/files/read"
	"github.com/mhmdkzr/loop/internal/agent/tools/files/rg"
	"github.com/mhmdkzr/loop/internal/agent/tools/files/write"
	"github.com/mhmdkzr/loop/internal/agent/tools/git"
	golang "github.com/mhmdkzr/loop/internal/agent/tools/go"
	"github.com/mhmdkzr/loop/internal/agent/tools/history"
	"github.com/mhmdkzr/loop/internal/agent/tools/nats"
	notesdelete "github.com/mhmdkzr/loop/internal/agent/tools/notes/delete"
	notesedit "github.com/mhmdkzr/loop/internal/agent/tools/notes/edit"
	noteslink "github.com/mhmdkzr/loop/internal/agent/tools/notes/link"
	noteslist "github.com/mhmdkzr/loop/internal/agent/tools/notes/list"
	notesread "github.com/mhmdkzr/loop/internal/agent/tools/notes/read"
	notessearch "github.com/mhmdkzr/loop/internal/agent/tools/notes/search"
	noteswrite "github.com/mhmdkzr/loop/internal/agent/tools/notes/write"
	"github.com/mhmdkzr/loop/internal/agent/tools/psql"
	"github.com/mhmdkzr/loop/internal/agent/tools/query"
	"github.com/mhmdkzr/loop/internal/agent/tools/telegram"
	telegramread "github.com/mhmdkzr/loop/internal/agent/tools/telegram/read"
	telegrams "github.com/mhmdkzr/loop/internal/agent/tools/telegram/send"
	"github.com/mhmdkzr/loop/internal/agent/tools/todo"
	"github.com/mhmdkzr/loop/internal/agent/tools/websearch"
)

// Tools returns the registry of every implemented agent tool, keyed by the
// tool_name stored in the tools table. A dispatched subagent resolves a
// filtered subset of it (Resolve); the top-level agent resolves all of it
// (All), since it needs access to everything.
func Tools() tools.Registry {
	return tools.Registry{
		agenttool.Name: func(deps tools.Deps) goai.Tool {
			return agenttool.Tool(deps.Store, deps.Config.Provider, Tools(), deps.SessionID, deps.Configured)
		},
		read.Name:         func(tools.Deps) goai.Tool { return read.Tool() },
		write.Name:        func(tools.Deps) goai.Tool { return write.Tool() },
		edit.Name:         func(tools.Deps) goai.Tool { return edit.Tool() },
		patch.Name:        func(tools.Deps) goai.Tool { return patch.Tool() },
		glob.Name:         func(tools.Deps) goai.Tool { return glob.Tool() },
		rg.Name:           func(tools.Deps) goai.Tool { return rg.Tool() },
		grep.Name:         func(tools.Deps) goai.Tool { return grep.Tool() },
		psql.Name:         func(tools.Deps) goai.Tool { return psql.Tool() },
		bash.Name:         func(tools.Deps) goai.Tool { return bash.Tool() },
		query.Name:        func(deps tools.Deps) goai.Tool { return query.Tool(deps) },
		history.Name:      func(deps tools.Deps) goai.Tool { return history.Tool(deps) },
		noteswrite.Name:   func(deps tools.Deps) goai.Tool { return noteswrite.Tool(deps) },
		notesdelete.Name:  func(deps tools.Deps) goai.Tool { return notesdelete.Tool(deps) },
		notesedit.Name:    func(deps tools.Deps) goai.Tool { return notesedit.Tool(deps) },
		noteslink.Name:    func(deps tools.Deps) goai.Tool { return noteslink.Tool(deps) },
		noteslist.Name:    func(deps tools.Deps) goai.Tool { return noteslist.Tool(deps) },
		notesread.Name:    func(deps tools.Deps) goai.Tool { return notesread.Tool(deps) },
		notessearch.Name:  func(deps tools.Deps) goai.Tool { return notessearch.Tool(deps) },
		git.Name:          func(tools.Deps) goai.Tool { return git.Tool() },
		golang.Name:       func(tools.Deps) goai.Tool { return golang.Tool() },
		curl.Name:         func(tools.Deps) goai.Tool { return curl.Tool() },
		deno.Name:         func(tools.Deps) goai.Tool { return deno.Tool() },
		nats.Name:         func(tools.Deps) goai.Tool { return nats.Tool() },
		datetime.Name:     func(tools.Deps) goai.Tool { return datetime.Tool() },
		todo.Name:         func(deps tools.Deps) goai.Tool { return todo.Tool(deps) },
		navigate.Name:     func(deps tools.Deps) goai.Tool { return deps.Configured[navigate.Name] },
		extract.Name:      func(deps tools.Deps) goai.Tool { return deps.Configured[extract.Name] },
		click.Name:        func(deps tools.Deps) goai.Tool { return deps.Configured[click.Name] },
		fill.Name:         func(deps tools.Deps) goai.Tool { return deps.Configured[fill.Name] },
		scroll.Name:       func(deps tools.Deps) goai.Tool { return deps.Configured[scroll.Name] },
		back.Name:         func(deps tools.Deps) goai.Tool { return deps.Configured[back.Name] },
		forward.Name:      func(deps tools.Deps) goai.Tool { return deps.Configured[forward.Name] },
		reset.Name:        func(deps tools.Deps) goai.Tool { return deps.Configured[reset.Name] },
		telegramread.Name: func(deps tools.Deps) goai.Tool { return deps.Configured[telegramread.Name] },
		telegrams.Name:    func(deps tools.Deps) goai.Tool { return deps.Configured[telegrams.Name] },
		websearch.Name:    func(deps tools.Deps) goai.Tool { return deps.Configured[websearch.Name] },
	}
}

// ConfiguredTools binds integrations that require long-lived runtime clients.
func ConfiguredTools(b *browser.Client, t *telegram.Client, w *websearch.Client) map[string]goai.Tool {
	return map[string]goai.Tool{
		navigate.Name:     navigate.Tool(b),
		extract.Name:      extract.Tool(b),
		click.Name:        click.Tool(b),
		fill.Name:         fill.Tool(b),
		scroll.Name:       scroll.Tool(b),
		back.Name:         back.Tool(b),
		forward.Name:      forward.Tool(b),
		reset.Name:        reset.Tool(b),
		telegramread.Name: telegramread.Tool(t),
		telegrams.Name:    telegrams.Tool(t),
		websearch.Name:    websearch.Tool(w),
	}
}
