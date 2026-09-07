package config

import "fmt"

// TelegramConfig configures the telegram_send and telegram_read agent tools.
type TelegramConfig struct {
	APIKey    string `env:"API_KEY"`
	ChannelID string `env:"CHANNEL_ID"`
}

// validate reports whether the telegram tool config is well-formed. The
// credentials are required: config loading fails when they are absent.
func (c TelegramConfig) validate() error {
	for name, value := range map[string]string{
		"API_KEY": c.APIKey, "CHANNEL_ID": c.ChannelID,
	} {
		if value == "" {
			return fmt.Errorf("%s must not be empty", name)
		}
	}
	return nil
}