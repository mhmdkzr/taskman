package review

import "testing"

func TestCommandListsEveryReviewSubcommand(t *testing.T) {
	want := []string{"recorded", "approved", "rejected"}
	cmd := Command()
	if cmd.Name != "review" {
		t.Fatalf("Name = %q, want review", cmd.Name)
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
