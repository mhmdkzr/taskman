package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"uuid"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
	agenttool "github.com/mhmdkzr/loop/internal/agent/tools/agent"
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
	notesdelete "github.com/mhmdkzr/loop/internal/agent/tools/notes/delete"
	notesedit "github.com/mhmdkzr/loop/internal/agent/tools/notes/edit"
	noteslink "github.com/mhmdkzr/loop/internal/agent/tools/notes/link"
	noteslist "github.com/mhmdkzr/loop/internal/agent/tools/notes/list"
	notesread "github.com/mhmdkzr/loop/internal/agent/tools/notes/read"
	notessearch "github.com/mhmdkzr/loop/internal/agent/tools/notes/search"
	noteswrite "github.com/mhmdkzr/loop/internal/agent/tools/notes/write"
	"github.com/mhmdkzr/loop/internal/agent/tools/psql"
	"github.com/mhmdkzr/loop/internal/agent/tools/query"
	tgread "github.com/mhmdkzr/loop/internal/agent/tools/telegram/read"
	tgsend "github.com/mhmdkzr/loop/internal/agent/tools/telegram/send"
	"github.com/mhmdkzr/loop/internal/agent/tools/todo"
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
		newToolSeed[noteswrite.Input, noteswrite.Output](noteswrite.Name, noteswrite.Description),
		newToolSeed[notesdelete.Input, notesdelete.Output](notesdelete.Name, notesdelete.Description),
		newToolSeed[notesedit.Input, notesedit.Output](notesedit.Name, notesedit.Description),
		newToolSeed[noteslink.Input, noteslink.Output](noteslink.Name, noteslink.Description),
		newToolSeed[noteslist.Input, noteslist.Output](noteslist.Name, noteslist.Description),
		newToolSeed[notesread.Input, notesread.Output](notesread.Name, notesread.Description),
		newToolSeed[notessearch.Input, notessearch.Output](notessearch.Name, notessearch.Description),
		newToolSeed[git.Input, tools.Output](git.Name, git.Description),
		newToolSeed[golang.Input, tools.Output](golang.Name, golang.Description),
		newToolSeed[bash.Input, tools.Output](bash.Name, bash.Description),
		newToolSeed[curl.Input, tools.Output](curl.Name, curl.Description),
		newToolSeed[deno.Input, tools.Output](deno.Name, deno.Description),
		newToolSeed[nats.Input, tools.Output](nats.Name, nats.Description),
		newToolSeed[datetime.Input, datetime.Output](datetime.Name, datetime.Description),
		newToolSeed[todo.Input, todo.Output](todo.Name, todo.Description),
		newToolSeed[agenttool.Input, agenttool.Output](agenttool.Name, agenttool.Description),
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
