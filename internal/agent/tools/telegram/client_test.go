package telegram

import (
	"testing"

	"github.com/mhmdkzr/loop/internal/app/config"
)

func TestNewClientFromConfig(t *testing.T) {
	c, err := NewClientFromConfig(config.TelegramConfig{APIKey: "k", ChannelID: "-100123456789"})
	if err != nil {
		t.Fatalf("NewClientFromConfig: %v", err)
	}
	if !c.Configured() {
		t.Error("not configured after NewClientFromConfig")
	}
}

func TestNewClientFromConfigDisabled(t *testing.T) {
	c, err := NewClientFromConfig(config.TelegramConfig{})
	if err != nil {
		t.Fatalf("NewClientFromConfig: %v", err)
	}
	if c.Configured() {
		t.Error("should be disabled with empty config")
	}
}

func TestNewClientFromConfigBadChannel(t *testing.T) {
	if _, err := NewClientFromConfig(config.TelegramConfig{APIKey: "k", ChannelID: "not-a-number"}); err == nil {
		t.Error("expected error for invalid channel id")
	}
}

func TestDefaultNewBot(t *testing.T) {
	c := NewClient("test-token", 0)
	b, err := c.NewBot("test-token")
	if err != nil {
		t.Fatal(err)
	}
	if b == nil {
		t.Error("expected a non-nil bot")
	}
}
