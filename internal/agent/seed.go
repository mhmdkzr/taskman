package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"uuid"

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
	"github.com/mhmdkzr/loop/internal/app/config"
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
func Seed(ctx context.Context, db *sql.DB, cfg config.ProviderConfig) error {
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
	return err
}

func upsertModel(ctx context.Context, db *sql.DB, providerName, modelName string) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO models (model_id, provider_id, model_name, context_window, has_vision)
		SELECT ?, provider_id, ?, 0, 0
		FROM model_providers
		WHERE provider_name = ?
		ON CONFLICT (provider_id, model_name) DO NOTHING`,
		uuid.NewV7().String(), modelName, providerName)
	return err
}

func seedProviders(ctx context.Context, db *sql.DB) error {
	return nil
}

func seedModels(ctx context.Context, db *sql.DB) error {
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
		newToolSeed[psql.Input, tools.Output](psql.Name, psql.Description),
		newToolSeed[git.Input, tools.Output](git.Name, git.Description),
		newToolSeed[golang.Input, tools.Output](golang.Name, golang.Description),
		newToolSeed[curl.Input, tools.Output](curl.Name, curl.Description),
		newToolSeed[deno.Input, tools.Output](deno.Name, deno.Description),
		newToolSeed[nats.Input, tools.Output](nats.Name, nats.Description),
		newToolSeed[datetime.Input, datetime.Output](datetime.Name, datetime.Description),
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
