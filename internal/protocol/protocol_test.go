package protocol

import (
	"strings"
	"testing"
)

func TestRunRequestValidate(t *testing.T) {
	cases := []struct {
		name    string
		req     RunRequest
		wantErr string
	}{
		{name: "prompt only", req: RunRequest{Prompt: "hello"}, wantErr: ""},
		{name: "missing prompt", req: RunRequest{}, wantErr: "prompt is required"},
		{name: "blank prompt", req: RunRequest{Prompt: "   "}, wantErr: "prompt is required"},
		{name: "continue session", req: RunRequest{Prompt: "hi", SessionID: "s1"}, wantErr: ""},
		{name: "fork", req: RunRequest{Prompt: "hi", ForkFrom: "s1", ForkTurn: 2}, wantErr: ""},
		{name: "fork and session", req: RunRequest{Prompt: "hi", SessionID: "s1", ForkFrom: "s2", ForkTurn: 1}, wantErr: "mutually exclusive"},
		{name: "fork without turn", req: RunRequest{Prompt: "hi", ForkFrom: "s1"}, wantErr: "turn is required with fork"},
		{name: "fork with zero turn", req: RunRequest{Prompt: "hi", ForkFrom: "s1", ForkTurn: 0}, wantErr: "turn is required with fork"},
		{name: "fork with negative turn", req: RunRequest{Prompt: "hi", ForkFrom: "s1", ForkTurn: -1}, wantErr: "turn is required with fork"},
		{name: "turn without fork", req: RunRequest{Prompt: "hi", ForkTurn: 2}, wantErr: "turn requires fork"},
		{name: "turn with session", req: RunRequest{Prompt: "hi", SessionID: "s1", ForkTurn: 2}, wantErr: "turn requires fork"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validate() = nil, want error containing %q", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("Validate() error = %q, want containing %q", err, tc.wantErr)
			}
		})
	}
}
