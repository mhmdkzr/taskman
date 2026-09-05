package agent

import (
	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
	"github.com/mhmdkzr/loop/internal/agent/tools/curl"
	"github.com/mhmdkzr/loop/internal/agent/tools/datetime"
	"github.com/mhmdkzr/loop/internal/agent/tools/deno"
	"github.com/mhmdkzr/loop/internal/agent/tools/files/edit"
	"github.com/mhmdkzr/loop/internal/agent/tools/files/glob"
	"github.com/mhmdkzr/loop/internal/agent/tools/files/patch"
	"github.com/mhmdkzr/loop/internal/agent/tools/files/read"
	"github.com/mhmdkzr/loop/internal/agent/tools/files/rg"
	"github.com/mhmdkzr/loop/internal/agent/tools/files/write"
	"github.com/mhmdkzr/loop/internal/agent/tools/git"
	golang "github.com/mhmdkzr/loop/internal/agent/tools/go"
	"github.com/mhmdkzr/loop/internal/agent/tools/nats"
	"github.com/mhmdkzr/loop/internal/agent/tools/psql"
)

// Tools returns the registry of every implemented agent tool, keyed by the
// tool_name stored in the tools table. A dispatched subagent resolves a
// filtered subset of it (Resolve); the top-level agent resolves all of it
// (All), since it needs access to everything.
//
// browser_*, telegram_*, and web_search are not wired in yet: they need a
// shared, long-lived client built from config (browser.Client, telegram.Client,
// websearch.Client) rather than the per-dispatch tools.Deps this registry
// hands every constructor.
func Tools() tools.Registry {
	return tools.Registry{
		read.Name:     func(tools.Deps) goai.Tool { return read.Tool() },
		write.Name:    func(tools.Deps) goai.Tool { return write.Tool() },
		edit.Name:     func(tools.Deps) goai.Tool { return edit.Tool() },
		patch.Name:    func(tools.Deps) goai.Tool { return patch.Tool() },
		glob.Name:     func(tools.Deps) goai.Tool { return glob.Tool() },
		rg.Name:       func(tools.Deps) goai.Tool { return rg.Tool() },
		psql.Name:     func(tools.Deps) goai.Tool { return psql.Tool() },
		git.Name:      func(tools.Deps) goai.Tool { return git.Tool() },
		golang.Name:   func(tools.Deps) goai.Tool { return golang.Tool() },
		curl.Name:     func(tools.Deps) goai.Tool { return curl.Tool() },
		deno.Name:     func(tools.Deps) goai.Tool { return deno.Tool() },
		nats.Name:     func(tools.Deps) goai.Tool { return nats.Tool() },
		datetime.Name: func(tools.Deps) goai.Tool { return datetime.Tool() },
	}
}
