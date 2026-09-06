package agent

import (
	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
	agentcreate "github.com/mhmdkzr/loop/internal/agent/tools/agent/create"
	agentdelete "github.com/mhmdkzr/loop/internal/agent/tools/agent/delete"
	agentget "github.com/mhmdkzr/loop/internal/agent/tools/agent/get"
	agentlist "github.com/mhmdkzr/loop/internal/agent/tools/agent/list"
	agentrun "github.com/mhmdkzr/loop/internal/agent/tools/agent/run"
	agentupdate "github.com/mhmdkzr/loop/internal/agent/tools/agent/update"
	"github.com/mhmdkzr/loop/internal/agent/tools/ask"
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
	"github.com/mhmdkzr/loop/internal/agent/tools/psql"
	"github.com/mhmdkzr/loop/internal/agent/tools/query"
	taskcreate "github.com/mhmdkzr/loop/internal/agent/tools/task/create"
	taskdelete "github.com/mhmdkzr/loop/internal/agent/tools/task/delete"
	taskget "github.com/mhmdkzr/loop/internal/agent/tools/task/get"
	tasklist "github.com/mhmdkzr/loop/internal/agent/tools/task/list"
	taskupdate "github.com/mhmdkzr/loop/internal/agent/tools/task/update"
	"github.com/mhmdkzr/loop/internal/agent/tools/telegram"
	tgread "github.com/mhmdkzr/loop/internal/agent/tools/telegram/read"
	tgsend "github.com/mhmdkzr/loop/internal/agent/tools/telegram/send"
	toolget "github.com/mhmdkzr/loop/internal/agent/tools/tool/get"
	toollist "github.com/mhmdkzr/loop/internal/agent/tools/tool/list"
	"github.com/mhmdkzr/loop/internal/agent/tools/websearch"
)

// Tools returns the registry of every implemented agent tool, keyed by the
// tool_name stored in the tools table. A dispatched subagent resolves a
// filtered subset of it (Resolve); the top-level agent resolves all of it
// (All), since it needs access to everything.
func Tools() tools.Registry {
	return tools.Registry{
		agentrun.Name: func(deps tools.Deps) goai.Tool {
			return agentrun.Tool(deps.Store, deps.Config.Provider, Tools(), deps.SessionID, deps.Configured)
		},
		agentcreate.Name: agentcreate.Tool,
		agentget.Name:    agentget.Tool,
		agentlist.Name:   agentlist.Tool,
		agentupdate.Name: agentupdate.Tool,
		agentdelete.Name: agentdelete.Tool,
		ask.Name:         ask.Tool,
		read.Name:        func(tools.Deps) goai.Tool { return read.Tool() },
		write.Name:       func(tools.Deps) goai.Tool { return write.Tool() },
		edit.Name:        func(tools.Deps) goai.Tool { return edit.Tool() },
		patch.Name:       func(tools.Deps) goai.Tool { return patch.Tool() },
		glob.Name:        func(tools.Deps) goai.Tool { return glob.Tool() },
		rg.Name:          func(tools.Deps) goai.Tool { return rg.Tool() },
		grep.Name:        func(tools.Deps) goai.Tool { return grep.Tool() },
		psql.Name:        func(tools.Deps) goai.Tool { return psql.Tool() },
		bash.Name:        func(tools.Deps) goai.Tool { return bash.Tool() },
		query.Name:       query.Tool,
		history.Name:     history.Tool,
		git.Name:         func(tools.Deps) goai.Tool { return git.Tool() },
		golang.Name:      func(tools.Deps) goai.Tool { return golang.Tool() },
		curl.Name:        func(tools.Deps) goai.Tool { return curl.Tool() },
		deno.Name:        func(tools.Deps) goai.Tool { return deno.Tool() },
		nats.Name:        func(tools.Deps) goai.Tool { return nats.Tool() },
		datetime.Name:    func(tools.Deps) goai.Tool { return datetime.Tool() },
		taskcreate.Name:  taskcreate.Tool,
		taskget.Name:     taskget.Tool,
		tasklist.Name:    tasklist.Tool,
		taskupdate.Name:  taskupdate.Tool,
		taskdelete.Name:  taskdelete.Tool,
		toolget.Name:     toolget.Tool,
		toollist.Name:    toollist.Tool,
		navigate.Name:    func(deps tools.Deps) goai.Tool { return deps.Configured[navigate.Name] },
		extract.Name:     func(deps tools.Deps) goai.Tool { return deps.Configured[extract.Name] },
		click.Name:       func(deps tools.Deps) goai.Tool { return deps.Configured[click.Name] },
		fill.Name:        func(deps tools.Deps) goai.Tool { return deps.Configured[fill.Name] },
		scroll.Name:      func(deps tools.Deps) goai.Tool { return deps.Configured[scroll.Name] },
		back.Name:        func(deps tools.Deps) goai.Tool { return deps.Configured[back.Name] },
		forward.Name:     func(deps tools.Deps) goai.Tool { return deps.Configured[forward.Name] },
		reset.Name:       func(deps tools.Deps) goai.Tool { return deps.Configured[reset.Name] },
		tgread.Name:      func(deps tools.Deps) goai.Tool { return deps.Configured[tgread.Name] },
		tgsend.Name:      func(deps tools.Deps) goai.Tool { return deps.Configured[tgsend.Name] },
		websearch.Name:   func(deps tools.Deps) goai.Tool { return deps.Configured[websearch.Name] },
	}
}

// ConfiguredTools binds integrations that require long-lived runtime clients.
func ConfiguredTools(b *browser.Client, t *telegram.Client, w *websearch.Client) map[string]goai.Tool {
	return map[string]goai.Tool{
		navigate.Name:  navigate.Tool(b),
		extract.Name:   extract.Tool(b),
		click.Name:     click.Tool(b),
		fill.Name:      fill.Tool(b),
		scroll.Name:    scroll.Tool(b),
		back.Name:      back.Tool(b),
		forward.Name:   forward.Tool(b),
		reset.Name:     reset.Tool(b),
		tgread.Name:    tgread.Tool(t),
		tgsend.Name:    tgsend.Tool(t),
		websearch.Name: websearch.Tool(w),
	}
}
