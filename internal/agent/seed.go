package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"uuid"

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
	tgread "github.com/mhmdkzr/loop/internal/agent/tools/telegram/read"
	tgsend "github.com/mhmdkzr/loop/internal/agent/tools/telegram/send"
	toolget "github.com/mhmdkzr/loop/internal/agent/tools/tool/get"
	toollist "github.com/mhmdkzr/loop/internal/agent/tools/tool/list"
	"github.com/mhmdkzr/loop/internal/agent/tools/websearch"
	"github.com/mhmdkzr/loop/internal/app/config"
	"github.com/mhmdkzr/loop/internal/store"
)

type toolSeed struct {
	Name         string
	Description  string
	InputSchema  json.RawMessage
	OutputSchema json.RawMessage
}

const (
	currentProviderName = "opencode"
	currentAPIKeyEnv    = "PROVIDER_API_KEY_OPENCODE"

	// operatorAgentName is loop's default top-level agent, seeded once so
	// there's always at least one agent to start a session against.
	operatorAgentName = "operator"
	operatorPrompt    = "You are operator, loop's default top-level agent. Help the user " +
		"accomplish tasks using the tools available to you, dispatching subagents for " +
		"work that benefits from running independently."
)

var providerNames = []string{
	"openai",
	"anthropic",
	"google",
	"bedrock",
	"azure",
	"vertex",
	"mistral",
	"xai",
	"groq",
	"cohere",
	"minimax",
	"deepseek",
	"fireworks",
	"together",
	"deepinfra",
	"openrouter",
	"requesty",
	"perplexity",
	"cerebras",
	"ollama",
	"vllm",
	"runpod",
	"cloudflare",
	"fptcloud",
	"nvidia",
	"llamacpp",
	"compat",
	"opencode",
}

// Seed writes the configured provider and model, plus the definitions for
// every tool currently available through Tools.
func Seed(ctx context.Context, st *store.Store, cfg config.ProviderConfig) error {
	db := st.RW()
	for _, name := range providerNames {
		baseURL := ""
		apiKeyEnv := ""
		if name == currentProviderName {
			baseURL = cfg.BaseURL
			apiKeyEnv = currentAPIKeyEnv
		}
		if err := upsertProvider(ctx, db, name, baseURL, apiKeyEnv); err != nil {
			return fmt.Errorf("seed provider %q: %w", name, err)
		}
	}
	if err := upsertModel(ctx, db, currentProviderName, cfg.Model); err != nil {
		return fmt.Errorf("seed model: %w", err)
	}
	if err := upsertTools(ctx, db, toolSeeds()); err != nil {
		return fmt.Errorf("seed tools: %w", err)
	}
	if err := seedOperatorAgent(ctx, st, cfg); err != nil {
		return fmt.Errorf("seed operator agent: %w", err)
	}
	return nil
}

// seedOperatorAgent creates the operator agent the first time loop starts
// against a fresh database, so there's always at least one agent to start a
// session against - agent creation itself is an agent tool (agentcreate),
// which needs an existing agent to run as, so it can't bootstrap itself.
func seedOperatorAgent(ctx context.Context, st *store.Store, cfg config.ProviderConfig) error {
	names, err := ListAgentNames(ctx, st)
	if err != nil {
		return fmt.Errorf("list agent names: %w", err)
	}
	if slices.Contains(names, operatorAgentName) {
		return nil
	}
	if err := CreateAgent(ctx, st, cfg, operatorAgentName, operatorPrompt); err != nil {
		return fmt.Errorf("create operator agent: %w", err)
	}
	return nil
}

func upsertProvider(ctx context.Context, db *sql.DB, name, baseURL, apiKeyEnv string) error {
	baseURLValue := sql.NullString{String: baseURL, Valid: baseURL != ""}
	apiKeyEnvValue := sql.NullString{String: apiKeyEnv, Valid: apiKeyEnv != ""}
	_, err := db.ExecContext(ctx, `
		INSERT INTO model_providers (provider_id, provider_name, base_url, api_key_env)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (provider_name) DO UPDATE SET
			base_url = EXCLUDED.base_url,
			api_key_env = EXCLUDED.api_key_env`,
		uuid.NewV7().String(), name, baseURLValue, apiKeyEnvValue)
	if err != nil {
		return fmt.Errorf("upsert provider: %w", err)
	}
	return nil
}

func upsertModel(ctx context.Context, db *sql.DB, providerName, modelName string) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO models (model_id, provider_id, model_name, context_window, has_vision)
		SELECT ?, provider_id, ?, 0, 0
		FROM model_providers
		WHERE provider_name = ?
		ON CONFLICT (provider_id, model_name) DO NOTHING`,
		uuid.NewV7().String(), modelName, providerName)
	if err != nil {
		return fmt.Errorf("upsert model: %w", err)
	}
	return nil
}

func toolSeeds() []toolSeed {
	return []toolSeed{
		newToolSeed[read.Input, read.Output](read.Name, read.Description),
		newToolSeed[write.Input, write.Output](write.Name, write.Description),
		newToolSeed[edit.Input, edit.Output](edit.Name, edit.Description),
		newToolSeed[glob.Input, glob.Output](glob.Name, glob.Description),
		newToolSeed[patch.Input, patch.Output](patch.Name, patch.Description),
		newToolSeed[rg.Input, tools.Output](rg.Name, rg.Description),
		newToolSeed[grep.Input, grep.Output](grep.Name, grep.Description),
		newToolSeed[psql.Input, tools.Output](psql.Name, psql.Description),
		newToolSeed[query.Input, query.Output](query.Name, query.Description),
		newToolSeed[history.Input, history.Output](history.Name, history.Description),
		newToolSeed[git.Input, tools.Output](git.Name, git.Description),
		newToolSeed[golang.Input, tools.Output](golang.Name, golang.Description),
		newToolSeed[bash.Input, tools.Output](bash.Name, bash.Description),
		newToolSeed[curl.Input, tools.Output](curl.Name, curl.Description),
		newToolSeed[deno.Input, tools.Output](deno.Name, deno.Description),
		newToolSeed[nats.Input, tools.Output](nats.Name, nats.Description),
		newToolSeed[datetime.Input, datetime.Output](datetime.Name, datetime.Description),
		newToolSeed[taskcreate.Input, taskcreate.Output](taskcreate.Name, taskcreate.Description),
		newToolSeed[taskget.Input, taskget.Output](taskget.Name, taskget.Description),
		newToolSeed[tasklist.Input, tasklist.Output](tasklist.Name, tasklist.Description),
		newToolSeed[taskupdate.Input, taskupdate.Output](taskupdate.Name, taskupdate.Description),
		newToolSeed[taskdelete.Input, taskdelete.Output](taskdelete.Name, taskdelete.Description),
		newToolSeed[agentrun.Input, agentrun.Output](agentrun.Name, agentrun.Description),
		newToolSeed[agentcreate.Input, agentcreate.Output](agentcreate.Name, agentcreate.Description),
		newToolSeed[agentget.Input, agentget.Output](agentget.Name, agentget.Description),
		newToolSeed[agentlist.Input, agentlist.Output](agentlist.Name, agentlist.Description),
		newToolSeed[agentupdate.Input, agentupdate.Output](agentupdate.Name, agentupdate.Description),
		newToolSeed[agentdelete.Input, agentdelete.Output](agentdelete.Name, agentdelete.Description),
		newToolSeed[ask.Input, ask.Output](ask.Name, ask.Description),
		newToolSeed[toolget.Input, toolget.Output](toolget.Name, toolget.Description),
		newToolSeed[toollist.Input, toollist.Output](toollist.Name, toollist.Description),
		newToolSeed[navigate.Input, navigate.Output](navigate.Name, navigate.Description),
		newToolSeed[extract.Input, extract.Output](extract.Name, extract.Description),
		newToolSeed[click.Input, click.Output](click.Name, click.Description),
		newToolSeed[fill.Input, fill.Output](fill.Name, fill.Description),
		newToolSeed[scroll.Input, scroll.Output](scroll.Name, scroll.Description),
		newToolSeed[back.Input, back.Output](back.Name, back.Description),
		newToolSeed[forward.Input, forward.Output](forward.Name, forward.Description),
		newToolSeed[reset.Input, reset.Output](reset.Name, reset.Description),
		newToolSeed[tgread.Input, tgread.Output](tgread.Name, tgread.Description),
		newToolSeed[tgsend.Input, tgsend.Output](tgsend.Name, tgsend.Description),
		newToolSeed[websearch.Input, websearch.Output](websearch.Name, websearch.Description),
	}
}

func newToolSeed[In, Out any](name, description string) toolSeed {
	return toolSeed{
		Name:         name,
		Description:  description,
		InputSchema:  goai.SchemaFrom[In](),
		OutputSchema: goai.SchemaFrom[Out](),
	}
}
