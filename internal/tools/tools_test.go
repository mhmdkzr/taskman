package tools

import (
	"testing"

	"github.com/mhmdkzr/taskman/internal/codebase"
	"github.com/mhmdkzr/taskman/internal/tools/telegram"
)

func TestToolsTelegramEnabled(t *testing.T) {
	tg := telegram.NewClient("k", -100123456789)
	if got := len(Tools(codebase.Repository{}, tg)); got != 8 {
		t.Errorf("tools = %d, want 8 (read, edit, glob, grep, go_build, go_test, telegram_send, telegram_read)", got)
	}
}

func TestToolsTelegramDisabled(t *testing.T) {
	if got := len(Tools(codebase.Repository{}, nil)); got != 6 {
		t.Errorf("tools = %d, want 6 (read, edit, glob, grep, go_build, go_test)", got)
	}
}
