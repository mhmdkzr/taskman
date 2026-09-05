package send

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools/telegram"
)

type fakeBot struct {
	send func(context.Context, *bot.SendMessageParams) (*models.Message, error)
}

func (f fakeBot) SendMessage(ctx context.Context, params *bot.SendMessageParams) (*models.Message, error) {
	return f.send(ctx, params)
}

func execTool(t *testing.T, tool goai.Tool, raw string) (string, error) {
	t.Helper()
	return tool.Execute(context.Background(), json.RawMessage(raw))
}

func TestToolSends(t *testing.T) {
	c := telegram.NewClient("token-123", 42)

	var gotToken string
	var gotParams *bot.SendMessageParams
	c.NewBot = func(token string) (telegram.Bot, error) {
		gotToken = token
		return fakeBot{
			send: func(ctx context.Context, params *bot.SendMessageParams) (*models.Message, error) {
				gotParams = params
				return &models.Message{ID: 99}, nil
			},
		}, nil
	}

	out, err := execTool(t, Tool(c), `{"message":" hello world "}`)
	if err != nil {
		t.Fatal(err)
	}
	if gotToken != "token-123" {
		t.Errorf("token = %q, want token-123", gotToken)
	}
	if gotParams == nil {
		t.Fatal("SendMessage was not called")
	}
	if gotParams.ChatID != int64(42) || gotParams.Text != "hello world" {
		t.Errorf("params = %+v, want chat_id=42 text=hello world", gotParams)
	}
	if out != `{"sent":true,"chat_id":42,"message_id":99}` {
		t.Errorf("out = %q", out)
	}
}

func TestToolNilResponse(t *testing.T) {
	c := telegram.NewClient("t", 7)
	c.NewBot = func(token string) (telegram.Bot, error) {
		return fakeBot{
			send: func(ctx context.Context, params *bot.SendMessageParams) (*models.Message, error) {
				return nil, nil
			},
		}, nil
	}

	out, err := execTool(t, Tool(c), `{"message":"hi"}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != `{"sent":true,"chat_id":7,"message_id":0}` {
		t.Errorf("out = %q", out)
	}
}

func TestToolFactoryError(t *testing.T) {
	c := telegram.NewClient("t", 7)
	c.NewBot = func(token string) (telegram.Bot, error) {
		return nil, context.DeadlineExceeded
	}

	if _, err := execTool(t, Tool(c), `{"message":"hi"}`); err == nil {
		t.Error("expected factory error")
	}
}

func TestToolSendError(t *testing.T) {
	c := telegram.NewClient("t", 7)
	c.NewBot = func(token string) (telegram.Bot, error) {
		return fakeBot{
			send: func(ctx context.Context, params *bot.SendMessageParams) (*models.Message, error) {
				return nil, context.Canceled
			},
		}, nil
	}

	if _, err := execTool(t, Tool(c), `{"message":"hi"}`); err == nil {
		t.Error("expected send error")
	}
}

func TestToolErrors(t *testing.T) {
	c := telegram.NewClient("", 0)

	if _, err := execTool(t, Tool(c), `{`); err == nil {
		t.Error("expected invalid json error")
	}
	if _, err := execTool(t, Tool(c), `{"message":""}`); err == nil {
		t.Error("expected empty message error")
	}
	if _, err := execTool(t, Tool(c), `{"message":"hi"}`); err == nil {
		t.Error("expected not-configured error")
	}
}
