package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/mhmdkzr/taskman/internal/config"
	"github.com/zendev-sh/goai"
)

type fakeTelegramClient struct {
	send func(context.Context, *bot.SendMessageParams) (*models.Message, error)
}

func (f fakeTelegramClient) SendMessage(ctx context.Context, params *bot.SendMessageParams) (*models.Message, error) {
	return f.send(ctx, params)
}

func execSend(t *testing.T, tool goai.Tool, raw string) (string, error) {
	t.Helper()
	return tool.Execute(context.Background(), json.RawMessage(raw))
}

func execRead(t *testing.T, tool goai.Tool, raw string) (string, error) {
	t.Helper()
	return tool.Execute(context.Background(), json.RawMessage(raw))
}

func TestNewClientFromConfig(t *testing.T) {
	c, err := NewClientFromConfig(config.Telegram{APIKey: "k", ChannelID: "-100123456789"})
	if err != nil {
		t.Fatalf("NewClientFromConfig: %v", err)
	}
	if !c.Configured() {
		t.Error("not configured after NewClientFromConfig")
	}
}

func TestNewClientFromConfigDisabled(t *testing.T) {
	c, err := NewClientFromConfig(config.Telegram{})
	if err != nil {
		t.Fatalf("NewClientFromConfig: %v", err)
	}
	if c.Configured() {
		t.Error("should be disabled with empty config")
	}
}

func TestNewClientFromConfigBadChannel(t *testing.T) {
	if _, err := NewClientFromConfig(config.Telegram{APIKey: "k", ChannelID: "not-a-number"}); err == nil {
		t.Error("expected error for invalid channel id")
	}
}

func TestSendToolSends(t *testing.T) {
	c := NewClient("token-123", 42)

	var gotToken string
	var gotParams *bot.SendMessageParams
	c.newBot = func(token string) (telegramClient, error) {
		gotToken = token
		return fakeTelegramClient{
			send: func(ctx context.Context, params *bot.SendMessageParams) (*models.Message, error) {
				gotParams = params
				return &models.Message{ID: 99}, nil
			},
		}, nil
	}

	out, err := execSend(t, SendTool(c), `{"message":" hello world "}`)
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
	if out != `{"chat_id":42,"message_id":99,"sent":true}` {
		t.Errorf("out = %q", out)
	}
}

func TestSendToolNilResponse(t *testing.T) {
	c := NewClient("t", 7)
	c.newBot = func(token string) (telegramClient, error) {
		return fakeTelegramClient{
			send: func(ctx context.Context, params *bot.SendMessageParams) (*models.Message, error) {
				return nil, nil
			},
		}, nil
	}

	out, err := execSend(t, SendTool(c), `{"message":"hi"}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != `{"chat_id":7,"message_id":0,"sent":true}` {
		t.Errorf("out = %q", out)
	}
}

func TestSendToolFactoryError(t *testing.T) {
	c := NewClient("t", 7)
	c.newBot = func(token string) (telegramClient, error) {
		return nil, context.DeadlineExceeded
	}

	if _, err := execSend(t, SendTool(c), `{"message":"hi"}`); err == nil {
		t.Error("expected factory error")
	}
}

func TestSendToolSendError(t *testing.T) {
	c := NewClient("t", 7)
	c.newBot = func(token string) (telegramClient, error) {
		return fakeTelegramClient{
			send: func(ctx context.Context, params *bot.SendMessageParams) (*models.Message, error) {
				return nil, context.Canceled
			},
		}, nil
	}

	if _, err := execSend(t, SendTool(c), `{"message":"hi"}`); err == nil {
		t.Error("expected send error")
	}
}

func TestSendToolErrors(t *testing.T) {
	c := NewClient("", 0)

	if _, err := execSend(t, SendTool(c), `{`); err == nil {
		t.Error("expected invalid json error")
	}
	if _, err := execSend(t, SendTool(c), `{"message":""}`); err == nil {
		t.Error("expected empty message error")
	}
	if _, err := execSend(t, SendTool(c), `{"message":"hi"}`); err == nil {
		t.Error("expected not-configured error")
	}
}

func TestDefaultTelegramBotFactory(t *testing.T) {
	b, err := defaultTelegramBotFactory("test-token")
	if err != nil {
		t.Fatal(err)
	}
	if b == nil {
		t.Error("expected a non-nil bot")
	}
}

// newReadClient returns a Client with stubbed fetch/ack plumbing.
func newReadClient(key string, id int64, fetch func(string, *http.Client, string) ([]telegramUpdate, error), ack func(string, *http.Client, string, int64) error) *Client {
	c := NewClient(key, id)
	c.fetch = fetch
	c.ack = ack
	return c
}

func msg(id int64, text string, date int64, first, last, username string) *telegramMessage {
	return &telegramMessage{
		Chat: &telegramChat{ID: id},
		From: &telegramUser{FirstName: first, LastName: last, Username: username},
		Date: date,
		Text: text,
	}
}

func TestReadTool(t *testing.T) {
	c := newReadClient("token-123", 42,
		func(token string, client *http.Client, gotToken string) ([]telegramUpdate, error) {
			if gotToken != "token-123" {
				t.Errorf("token = %q, want token-123", gotToken)
			}
			return []telegramUpdate{
				{UpdateID: 10, Message: msg(99, "ignore me", 1609459200, "Other", "", "")},
				{UpdateID: 11, ChannelPost: msg(42, "hello", 1609459200, "Alice", "Smith", "")},
				{UpdateID: 12, Message: msg(42, "hi there", 1609462800, "", "", "bob")},
			}, nil
		},
		func(baseURL string, client *http.Client, token string, maxID int64) error {
			if maxID != 12 {
				t.Errorf("acked = %d, want 12", maxID)
			}
			return nil
		},
	)

	out, err := execRead(t, ReadTool(c), `{}`)
	if err != nil {
		t.Fatal(err)
	}
	want := "Alice Smith (2021-01-01 00:00): hello\nbob (2021-01-01 01:00): hi there"
	if out != want {
		t.Errorf("out = %q, want %q", out, want)
	}
}

func TestReadToolLimit(t *testing.T) {
	c := newReadClient("k", 1,
		func(string, *http.Client, string) ([]telegramUpdate, error) {
			return []telegramUpdate{
				{UpdateID: 1, Message: msg(1, "one", 1609459200, "A", "", "")},
				{UpdateID: 2, Message: msg(1, "two", 1609462800, "B", "", "")},
				{UpdateID: 3, Message: msg(1, "three", 1609466400, "C", "", "")},
			}, nil
		},
		func(string, *http.Client, string, int64) error { return nil },
	)

	out, err := execRead(t, ReadTool(c), `{"limit":2}`)
	if err != nil {
		t.Fatal(err)
	}
	want := "B (2021-01-01 01:00): two\nC (2021-01-01 02:00): three"
	if out != want {
		t.Errorf("out = %q, want %q", out, want)
	}
}

func TestReadToolNoMessages(t *testing.T) {
	var ackCalled bool
	c := newReadClient("k", 1,
		func(string, *http.Client, string) ([]telegramUpdate, error) {
			return []telegramUpdate{{UpdateID: 5, Message: msg(2, "other chat", 1609459200, "A", "", "")}}, nil
		},
		func(string, *http.Client, string, int64) error {
			ackCalled = true
			return nil
		},
	)

	out, err := execRead(t, ReadTool(c), `{}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "no new messages" {
		t.Errorf("out = %q, want no new messages", out)
	}
	if !ackCalled {
		t.Error("expected updates to be acknowledged even with no matching messages")
	}
}

func TestReadToolEmptyNoAck(t *testing.T) {
	var ackCalled bool
	c := newReadClient("k", 1,
		func(string, *http.Client, string) ([]telegramUpdate, error) { return nil, nil },
		func(string, *http.Client, string, int64) error {
			ackCalled = true
			return nil
		},
	)

	out, err := execRead(t, ReadTool(c), `{}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "no new messages" {
		t.Errorf("out = %q, want no new messages", out)
	}
	if ackCalled {
		t.Error("expected no ack when there are no updates")
	}
}

func TestReadToolMediaMessage(t *testing.T) {
	c := newReadClient("k", 1,
		func(string, *http.Client, string) ([]telegramUpdate, error) {
			return []telegramUpdate{{UpdateID: 1, ChannelPost: msg(1, "", 1609459200, "Alice", "", "")}}, nil
		},
		func(string, *http.Client, string, int64) error { return nil },
	)

	out, err := execRead(t, ReadTool(c), `{}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "Alice (2021-01-01 00:00): [media message]" {
		t.Errorf("out = %q", out)
	}
}

func TestReadToolErrors(t *testing.T) {
	c := NewClient("", 0)

	if _, err := execRead(t, ReadTool(c), `{`); err == nil {
		t.Error("expected invalid json error")
	}
	if _, err := execRead(t, ReadTool(c), `{}`); err == nil {
		t.Error("expected not-configured error")
	}

	c = NewClient("k", 1)
	c.fetch = func(string, *http.Client, string) ([]telegramUpdate, error) { return nil, nil }
	c.ack = func(string, *http.Client, string, int64) error { return nil }
	for _, args := range []string{`{"limit":0}`, `{"limit":101}`} {
		if _, err := execRead(t, ReadTool(c), args); err == nil {
			t.Errorf("expected limit error for %s", args)
		}
	}
}

func TestReadToolFetchError(t *testing.T) {
	c := newReadClient("k", 1,
		func(string, *http.Client, string) ([]telegramUpdate, error) { return nil, context.DeadlineExceeded },
		func(string, *http.Client, string, int64) error { return nil },
	)
	if _, err := execRead(t, ReadTool(c), `{}`); err == nil {
		t.Error("expected fetch error")
	}
}

func TestReadToolAckError(t *testing.T) {
	c := newReadClient("k", 1,
		func(string, *http.Client, string) ([]telegramUpdate, error) {
			return []telegramUpdate{{UpdateID: 7, Message: msg(1, "hi", 1609459200, "A", "", "")}}, nil
		},
		func(string, *http.Client, string, int64) error { return context.Canceled },
	)
	if _, err := execRead(t, ReadTool(c), `{}`); err == nil {
		t.Error("expected ack error")
	}
}

func TestFetchTelegramUpdates(t *testing.T) {
	var gotLimit, gotAllowed string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		gotLimit = q.Get("limit")
		gotAllowed = q.Get("allowed_updates")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"result":[{"update_id":1,"message":{"chat":{"id":5},"text":"x"}}]}`)
	}))
	defer srv.Close()

	updates, err := fetchTelegramUpdates(srv.URL, &http.Client{}, "tok")
	if err != nil {
		t.Fatal(err)
	}
	if len(updates) != 1 || updates[0].UpdateID != 1 {
		t.Fatalf("updates = %+v", updates)
	}
	if gotLimit != "100" {
		t.Errorf("limit param = %q, want 100", gotLimit)
	}
	if !strings.Contains(gotAllowed, "message") || !strings.Contains(gotAllowed, "channel_post") {
		t.Errorf("allowed_updates param = %q", gotAllowed)
	}
}

func TestFetchTelegramUpdatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "boom")
	}))
	defer srv.Close()

	_, err := fetchTelegramUpdates(srv.URL, &http.Client{}, "tok")
	if err == nil || !strings.Contains(err.Error(), "getUpdates status 500") {
		t.Fatalf("err = %v, want status 500 error", err)
	}
}

func TestFetchTelegramUpdatesDecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,`)
	}))
	defer srv.Close()

	if _, err := fetchTelegramUpdates(srv.URL, &http.Client{}, "tok"); err == nil {
		t.Error("expected decode error")
	}
}

func TestFetchTelegramUpdatesNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	if _, err := fetchTelegramUpdates(url, &http.Client{}, "tok"); err == nil {
		t.Error("expected network error")
	}
}

func TestAckTelegramUpdates(t *testing.T) {
	var gotOffset string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotOffset = r.URL.Query().Get("offset")
		fmt.Fprint(w, `{"ok":true,"result":[]}`)
	}))
	defer srv.Close()

	if err := ackTelegramUpdates(srv.URL, &http.Client{}, "tok", 9); err != nil {
		t.Fatal(err)
	}
	if gotOffset != "10" {
		t.Errorf("offset = %q, want 10", gotOffset)
	}
}

func TestAckTelegramUpdatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	if err := ackTelegramUpdates(srv.URL, &http.Client{}, "tok", 9); err == nil {
		t.Error("expected ack status error")
	}
}

func TestAckTelegramUpdatesNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	if err := ackTelegramUpdates(url, &http.Client{}, "tok", 9); err == nil {
		t.Error("expected ack network error")
	}
}

func TestSenderName(t *testing.T) {
	cases := []struct {
		u    *telegramUser
		want string
	}{
		{nil, "unknown"},
		{&telegramUser{}, "unknown"},
		{&telegramUser{FirstName: " A "}, "A"},
		{&telegramUser{FirstName: "Alice", LastName: "Smith"}, "Alice Smith"},
		{&telegramUser{Username: "bob"}, "bob"},
	}
	for _, c := range cases {
		if got := senderName(c.u); got != c.want {
			t.Errorf("senderName(%+v) = %q, want %q", c.u, got, c.want)
		}
	}
}
