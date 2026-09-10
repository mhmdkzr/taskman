package automatedreview

import "testing"

func TestCommandListsVerdictSubcommands(t *testing.T) {
	cmd := Command()
	if cmd.Name != "automated-review" {
		t.Fatalf("Name = %q, want automated-review", cmd.Name)
	}
	want := []string{"approved", "rejected"}
	if len(cmd.Commands) != len(want) {
		t.Fatalf("Commands = %d, want %d", len(cmd.Commands), len(want))
	}
	for i, name := range want {
		if cmd.Commands[i].Name != name {
			t.Errorf("Commands[%d].Name = %q, want %q", i, cmd.Commands[i].Name, name)
		}
	}
}
