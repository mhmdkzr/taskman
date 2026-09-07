package config

import "fmt"

type ProviderConfig struct {
	BaseURL         string `env:"BASE_URL"`
	APIKeyOpenCode  string `env:"API_KEY_OPENCODE"`
	Model           string `env:"MODEL"`
	ReasoningEffort string `env:"REASONING_EFFORT"`
}

func (c ProviderConfig) validate() error {
	for name, value := range map[string]string{
		"BASE_URL":         c.BaseURL,
		"API_KEY_OPENCODE": c.APIKeyOpenCode,
		"MODEL":            c.Model,
		"REASONING_EFFORT": c.ReasoningEffort,
	} {
		if value == "" {
			return fmt.Errorf("%s must not be empty", name)
		}
	}
	return nil
}