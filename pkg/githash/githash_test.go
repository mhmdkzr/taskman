package githash

import (
	"runtime/debug"
	"testing"
)

func TestRevision(t *testing.T) {
	tests := []struct {
		name string
		info *debug.BuildInfo
		want string
	}{
		{
			name: "nil build info",
			info: nil,
			want: "",
		},
		{
			name: "no vcs settings",
			info: &debug.BuildInfo{
				Settings: []debug.BuildSetting{{Key: "go.version", Value: "go1.26"}},
			},
			want: "",
		},
		{
			name: "vcs revision present",
			info: &debug.BuildInfo{
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "7034b82c"},
					{Key: "vcs.modified", Value: "false"},
				},
			},
			want: "7034b82c",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := revision(tt.info); got != tt.want {
				t.Fatalf("revision() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetCommitHash_MatchesBuildInfo(t *testing.T) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		t.Skip("build info unavailable")
	}
	if want := revision(info); GetCommitHash() != want {
		t.Fatalf("GetCommitHash() = %q, want %q", GetCommitHash(), want)
	}
}
