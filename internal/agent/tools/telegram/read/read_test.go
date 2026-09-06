package read

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools/telegram"
)

func execTool(t *testing.T, tool goai.Tool, raw string) (string, error) {
	t.Helper()
	return tool.Execute(context.Background(), json.RawMessage(raw))
}

// newClient returns a Client pointed at a fake Telegram server that answers
// fetch calls with the given handler and records any ack offset in acked.
func newClient(t *testing.T, fetchBody string, acked *string) *telegram.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if offset := r.URL.Query().Get("offset"); offset != "" {
			if acked != nil {
				*acked = offset
			}
			fmt.Fprint(w, `{"ok":true,"result":[]}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fetchBody)
	}))
	t.Cleanup(srv.Close)

	c := telegram.NewClient("token-123", 42)
	c.BaseURL = srv.URL
	return c
}

func TestTool(t *testing.T) {
	var acked string
	c := newClient(t, `{"ok":true,"result":[
		{"update_id":10,"message":{"chat":{"id":99},"from":{"first_name":"Other"},"date":1609459200,"text":"ignore me"}},
		{"update_id":11,"channel_post":{"chat":{"id":42},"from":{"first_name":"Alice","last_name":"Smith"},"date":1609459200,"text":"hello"}},
		{"update_id":12,"message":{"chat":{"id":42},"from":{"username":"bob"},"date":1609462800,"text":"hi there"}}
	]}`, &acked)

	out, err := execTool(t, Tool(c), `{}`)
	if err != nil {
		t.Fatal(err)
	}
	var got Output
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Messages) != 2 || got.Messages[0].Sender != "Alice Smith" || got.Messages[1].Text != "hi there" {
		t.Errorf("messages = %+v", got.Messages)
	}
	if acked != "13" {
		t.Errorf("acked = %q, want 13", acked)
	}
}

func TestToolLimit(t *testing.T) {
	var acked string
	c := newClient(t, `{"ok":true,"result":[
		{"update_id":1,"message":{"chat":{"id":42},"from":{"first_name":"A"},"date":1609459200,"text":"one"}},
		{"update_id":2,"message":{"chat":{"id":42},"from":{"first_name":"B"},"date":1609462800,"text":"two"}},
		{"update_id":3,"message":{"chat":{"id":42},"from":{"first_name":"C"},"date":1609466400,"text":"three"}}
	]}`, &acked)

	out, err := execTool(t, Tool(c), `{"limit":2}`)
	if err != nil {
		t.Fatal(err)
	}
	var got Output
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Messages) != 2 || got.Messages[0].Sender != "B" || got.Messages[1].Text != "three" {
		t.Errorf("messages = %+v", got.Messages)
	}
}

func TestToolNoMessages(t *testing.T) {
	var acked string
	c := newClient(t, `{"ok":true,"result":[
		{"update_id":5,"message":{"chat":{"id":99},"from":{"first_name":"A"},"date":1609459200,"text":"other chat"}}
	]}`, &acked)

	out, err := execTool(t, Tool(c), `{}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != `{"messages":[]}` {
		t.Errorf("out = %q, want empty messages", out)
	}
	if acked != "6" {
		t.Errorf("acked = %q, want 6", acked)
	}
}

func TestToolEmptyNoAck(t *testing.T) {
	var acked string
	c := newClient(t, `{"ok":true,"result":[]}`, &acked)

	out, err := execTool(t, Tool(c), `{}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != `{"messages":[]}` {
		t.Errorf("out = %q, want empty messages", out)
	}
	if acked != "" {
		t.Errorf("acked = %q, want no ack", acked)
	}
}

func TestToolMediaMessage(t *testing.T) {
	var acked string
	c := newClient(t, `{"ok":true,"result":[
		{"update_id":1,"channel_post":{"chat":{"id":42},"from":{"first_name":"Alice"},"date":1609459200,"text":""}}
	]}`, &acked)

	out, err := execTool(t, Tool(c), `{}`)
	if err != nil {
		t.Fatal(err)
	}
	var got Output
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Messages) != 1 || got.Messages[0].Text != "[media message]" {
		t.Errorf("messages = %+v", got.Messages)
	}
}

func TestToolErrors(t *testing.T) {
	c := telegram.NewClient("", 0)

	if _, err := execTool(t, Tool(c), `{`); err == nil {
		t.Error("expected invalid json error")
	}
	if _, err := execTool(t, Tool(c), `{}`); err == nil {
		t.Error("expected not-configured error")
	}

	c = telegram.NewClient("k", 1)
	for _, args := range []string{`{"limit":0}`, `{"limit":101}`} {
		if _, err := execTool(t, Tool(c), args); err == nil {
			t.Errorf("expected limit error for %s", args)
		}
	}
}

func TestToolFetchError(t *testing.T) {
	c := newClient(t, "", nil)

	if _, err := execTool(t, Tool(c), `{}`); err == nil {
		t.Error("expected fetch error")
	}
}

func TestToolAckError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("offset") != "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		fmt.Fprint(
			w,
			`{"ok":true,"result":[{"update_id":7,"message":{"chat":{"id":42},"from":{"first_name":"A"},"date":1609459200,"text":"hi"}}]}`,
		)
	}))
	defer srv.Close()

	c := telegram.NewClient("k", 42)
	c.BaseURL = srv.URL

	if _, err := execTool(t, Tool(c), `{}`); err == nil {
		t.Error("expected ack error")
	}
}

func TestFetchUpdates(t *testing.T) {
	var gotLimit, gotAllowed string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		gotLimit = q.Get("limit")
		gotAllowed = q.Get("allowed_updates")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,"result":[{"update_id":1,"message":{"chat":{"id":5},"text":"x"}}]}`)
	}))
	defer srv.Close()

	updates, err := fetchUpdates(context.Background(), srv.URL, &http.Client{}, "tok")
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

func TestFetchUpdatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "boom")
	}))
	defer srv.Close()

	_, err := fetchUpdates(context.Background(), srv.URL, &http.Client{}, "tok")
	if err == nil || !strings.Contains(err.Error(), "getUpdates status 500") {
		t.Fatalf("err = %v, want status 500 error", err)
	}
}

func TestFetchUpdatesDecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true,`)
	}))
	defer srv.Close()

	if _, err := fetchUpdates(context.Background(), srv.URL, &http.Client{}, "tok"); err == nil {
		t.Error("expected decode error")
	}
}

func TestFetchUpdatesNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	if _, err := fetchUpdates(context.Background(), url, &http.Client{}, "tok"); err == nil {
		t.Error("expected network error")
	}
}

func TestAckUpdates(t *testing.T) {
	var gotOffset string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotOffset = r.URL.Query().Get("offset")
		fmt.Fprint(w, `{"ok":true,"result":[]}`)
	}))
	defer srv.Close()

	if err := ackUpdates(context.Background(), srv.URL, &http.Client{}, "tok", 9); err != nil {
		t.Fatal(err)
	}
	if gotOffset != "10" {
		t.Errorf("offset = %q, want 10", gotOffset)
	}
}

func TestAckUpdatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	if err := ackUpdates(context.Background(), srv.URL, &http.Client{}, "tok", 9); err == nil {
		t.Error("expected ack status error")
	}
}

func TestAckUpdatesNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	if err := ackUpdates(context.Background(), url, &http.Client{}, "tok", 9); err == nil {
		t.Error("expected ack network error")
	}
}

func TestSenderName(t *testing.T) {
	cases := []struct {
		u    *user
		want string
	}{
		{nil, "unknown"},
		{&user{}, "unknown"},
		{&user{FirstName: " A "}, "A"},
		{&user{FirstName: "Alice", LastName: "Smith"}, "Alice Smith"},
		{&user{Username: "bob"}, "bob"},
	}
	for _, c := range cases {
		if got := senderName(c.u); got != c.want {
			t.Errorf("senderName(%+v) = %q, want %q", c.u, got, c.want)
		}
	}
}
