package task

import "testing"

func TestCommandListsEveryTaskSubcommand(t *testing.T) {
	want := []string{
		"list", "get", "create", "update", "specify", "implement", "verify",
		"review", "commit", "escalate", "merge", "abandon", "next", "delete",
	}
	cmd := Command()
	if cmd.Name != "task" {
		t.Fatalf("Name = %q, want task", cmd.Name)
	}
	if len(cmd.Commands) != len(want) {
		t.Fatalf("Commands = %d, want %d", len(cmd.Commands), len(want))
	}
	for i, name := range want {
		if cmd.Commands[i].Name != name {
			t.Errorf("Commands[%d].Name = %q, want %q", i, cmd.Commands[i].Name, name)
		}
	}
}
