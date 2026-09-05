// Package models synchronizes OpenCode's model catalog with SQLite.
package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"text/tabwriter"
	"uuid"

	"github.com/mhmdkzr/loop/internal/app/config"
)

const providerName = "opencode"

type catalogResponse struct {
	Data []catalogModel `json:"data"`
}

type catalogModel struct {
	ID string `json:"id"`
}

// ListedModel is one model currently stored in the local catalog.
type ListedModel struct {
	Provider        string                     `json:"provider"`
	Name            string                     `json:"name"`
	ThinkingOptions map[string]json.RawMessage `json:"thinking_options"`
}

// Refresh fetches the current OpenCode model catalog and upserts every model
// with its provider options. OpenCode's models endpoint does not publish
// thinking capabilities, so generic models receive the standard three levels
// and known OpenCode exceptions receive their provider-specific options.
func Refresh(ctx context.Context, db *sql.DB, cfg config.ProviderConfig) error {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return fmt.Errorf("refresh models: base URL is required")
	}
	if strings.TrimSpace(cfg.APIKeyOpenCode) == "" {
		return fmt.Errorf("refresh models: API key is required")
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(cfg.BaseURL, "/")+"/models", nil)
	if err != nil {
		return fmt.Errorf("refresh models: create request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+cfg.APIKeyOpenCode)
	request.Header.Set("Accept", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("refresh models: request catalog: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("refresh models: catalog returned HTTP %d", response.StatusCode)
	}

	var catalog catalogResponse
	if err := json.NewDecoder(response.Body).Decode(&catalog); err != nil {
		return fmt.Errorf("refresh models: decode catalog: %w", err)
	}
	if len(catalog.Data) == 0 {
		return fmt.Errorf("refresh models: catalog is empty")
	}

	providerID, err := providerID(ctx, db, cfg.BaseURL)
	if err != nil {
		return err
	}
	for _, model := range catalog.Data {
		if err := upsertModel(ctx, db, providerID, model.ID, thinkingOptions(model.ID)); err != nil {
			return fmt.Errorf("refresh models: upsert %q: %w", model.ID, err)
		}
	}
	return nil
}

// List writes the locally stored model catalog as JSON.
func List(ctx context.Context, db *sql.DB, output io.Writer, jsonOutput bool) error {
	rows, err := db.QueryContext(ctx, `
		SELECT p.provider_name, m.model_name, m.thinking_options
		FROM models m
		JOIN model_providers p ON p.provider_id = m.provider_id
		ORDER BY p.provider_name, m.model_name`)
	if err != nil {
		return fmt.Errorf("list models: query: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			slog.Error("close model rows", "error", closeErr)
		}
	}()

	models := make([]ListedModel, 0)
	for rows.Next() {
		var (
			model       ListedModel
			optionsJSON []byte
		)
		if err := rows.Scan(&model.Provider, &model.Name, &optionsJSON); err != nil {
			return fmt.Errorf("list models: scan: %w", err)
		}
		if len(optionsJSON) > 0 {
			if err := json.Unmarshal(optionsJSON, &model.ThinkingOptions); err != nil {
				return fmt.Errorf("list models: decode options: %w", err)
			}
		}
		models = append(models, model)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("list models: iterate: %w", err)
	}
	if jsonOutput {
		if err := json.NewEncoder(output).Encode(models); err != nil {
			return fmt.Errorf("list models: encode: %w", err)
		}
		return nil
	}
	writer := tabwriter.NewWriter(output, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(writer, "PROVIDER\tMODEL\tTHINKING OPTIONS"); err != nil {
		return fmt.Errorf("list models: write header: %w", err)
	}
	for _, model := range models {
		if _, err := fmt.Fprintf(writer, "%s\t%s\t%d\n", model.Provider, model.Name, len(model.ThinkingOptions)); err != nil {
			return fmt.Errorf("list models: write row: %w", err)
		}
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("list models: flush: %w", err)
	}
	return nil
}

func providerID(ctx context.Context, db *sql.DB, baseURL string) (string, error) {
	id := uuid.NewV7().String()
	_, err := db.ExecContext(ctx, `
		INSERT INTO model_providers (provider_id, provider_name, base_url, api_key_env)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (provider_name) DO UPDATE SET
			base_url = EXCLUDED.base_url,
			api_key_env = EXCLUDED.api_key_env`,
		id, providerName, baseURL, "PROVIDER_API_KEY_OPENCODE")
	if err != nil {
		return "", fmt.Errorf("refresh models: upsert provider: %w", err)
	}

	if err := db.QueryRowContext(ctx,
		`SELECT provider_id FROM model_providers WHERE provider_name = ?`, providerName).Scan(&id); err != nil {
		return "", fmt.Errorf("refresh models: query provider: %w", err)
	}
	return id, nil
}

func upsertModel(ctx context.Context, db *sql.DB, providerID, modelID string, options map[string]json.RawMessage) error {
	if strings.TrimSpace(modelID) == "" {
		return fmt.Errorf("model id is empty")
	}
	optionsJSON, err := json.Marshal(options)
	if err != nil {
		return fmt.Errorf("marshal thinking options: %w", err)
	}
	_, err = db.ExecContext(ctx, `
		INSERT INTO models (
			model_id, provider_id, model_name, context_window, has_vision, thinking_options
		) VALUES (?, ?, ?, 0, 0, ?)
		ON CONFLICT (provider_id, model_name) DO UPDATE SET
			thinking_options = EXCLUDED.thinking_options`,
		uuid.NewV7().String(), providerID, modelID, optionsJSON)
	return err
}

func thinkingOptions(modelID string) map[string]json.RawMessage {
	options := map[string]json.RawMessage{
		"low":    json.RawMessage(`{"reasoning_effort":"low"}`),
		"medium": json.RawMessage(`{"reasoning_effort":"medium"}`),
		"high":   json.RawMessage(`{"reasoning_effort":"high"}`),
	}
	lowerID := strings.ToLower(modelID)
	switch {
	case strings.Contains(lowerID, "deepseek-v4"):
		options["max"] = json.RawMessage(`{"reasoning_effort":"max"}`)
	case strings.Contains(lowerID, "glm-5.2") || strings.Contains(lowerID, "glm-5-2"):
		delete(options, "low")
		delete(options, "medium")
		options["max"] = json.RawMessage(`{"reasoning_effort":"max"}`)
	case lowerID == "minimax-m3":
		options = map[string]json.RawMessage{
			"disabled": json.RawMessage(`{"thinking":{"type":"disabled"}}`),
			"thinking": json.RawMessage(`{"thinking":{"type":"adaptive"}}`),
		}
	}
	return options
}
