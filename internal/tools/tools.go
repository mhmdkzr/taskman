package tools

import (
	"github.com/mhmdkzr/taskman/internal/codebase"
	codebasetools "github.com/mhmdkzr/taskman/internal/tools/codebase"
	"github.com/mhmdkzr/taskman/internal/tools/telegram"
	"github.com/zendev-sh/goai"
)

// Tools returns all registered tools bound to repo: the codebase tools
// (read, edit, glob, grep — see internal/tools/codebase), plus the telegram
// tools when tg carries credentials.
func Tools(repo codebase.Repository, tg *telegram.Client) []goai.Tool {
	out := codebasetools.Tools(repo)
	if tg.Configured() {
		out = append(out, telegram.SendTool(tg), telegram.ReadTool(tg))
	}
	return out
}
