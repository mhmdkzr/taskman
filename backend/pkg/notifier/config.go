package notifier

import "fmt"

// Config holds the notifier configuration loaded from environment variables.
type Config struct {
	Enabled  bool           `env:"ENABLED"`
	Telegram TelegramConfig `envPrefix:"TELEGRAM_"`
}

type TelegramConfig struct {
	BotToken  string `env:"BOT_TOKEN"`
	ChannelID int64  `env:"CHANNEL_ID"`
}

// Validate returns an error if the configuration is invalid.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	return c.Telegram.Validate()
}

// Validate returns an error if the telegram configuration is invalid.
func (c TelegramConfig) Validate() error {
	if c.BotToken == "" {
		return fmt.Errorf("TELEGRAM_BOT_TOKEN must not be empty when notifier is enabled")
	}
	if c.ChannelID == 0 {
		return fmt.Errorf("TELEGRAM_CHANNEL_ID must not be zero when notifier is enabled")
	}
	return nil
}
