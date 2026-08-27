package auditlog

import (
	"testing"

	"github.com/mhmdkzr/app/pkg/natsembed"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{name: "empty subject", cfg: Config{Stream: "API_AUDIT"}},
		{name: "blank subject", cfg: Config{Subject: " ", Stream: "API_AUDIT"}},
		{name: "empty stream", cfg: Config{Subject: "api.audit"}},
		{name: "blank stream", cfg: Config{Subject: "api.audit", Stream: " "}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.validate(); err == nil {
				t.Fatal("validate() error = nil, want error")
			}
		})
	}
}

func TestCreateStream_UsesConfiguredSubjectAndStream(t *testing.T) {
	nc, js, err := natsembed.Connect()
	if err != nil {
		t.Fatalf("connect embedded NATS: %v", err)
	}
	defer nc.Close()

	subject, stream := uniqueAuditTestSubject(t)
	cfg := Config{Subject: subject, Stream: stream}
	if err := CreateStream(t.Context(), js, cfg); err != nil {
		t.Fatalf("CreateStream() error = %v", err)
	}

	info, err := js.Stream(t.Context(), stream)
	if err != nil {
		t.Fatalf("get configured stream: %v", err)
	}
	cached := info.CachedInfo()
	if cached.Config.Name != stream {
		t.Fatalf("stream name = %q, want %q", cached.Config.Name, stream)
	}
	if len(cached.Config.Subjects) != 1 || cached.Config.Subjects[0] != subject {
		t.Fatalf("stream subjects = %v, want [%q]", cached.Config.Subjects, subject)
	}
}
