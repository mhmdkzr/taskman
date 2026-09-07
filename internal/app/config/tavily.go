package config

import "fmt"

// TavilyConfig configures the websearch agent tool.
type TavilyConfig struct {
	APIKey string `env:"API_KEY"`
}

// validate reports whether the websearch tool config is well-formed. The API
// key is required: config loading fails when it is absent.
func (c TavilyConfig) validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("API_KEY must not be empty")
	}
	return nil
}