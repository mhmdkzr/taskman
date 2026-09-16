package instructions

import (
	"fmt"
	"sort"

	"github.com/mhmdkzr/taskman/internal/task"
)

// commandNames maps each EventKind to the CLI invocation that reports it,
// with %s standing in for the task id. It is the one place instructions need
// to know each slice's command name - the rest of a command's flags aren't
// reconstructable from the task alone (a plan, check results, a commit
// target, ...), so placeholders stand in for those.
var commandNames = map[task.EventKind]string{
	task.EventSpecificationSubmitted: "specified --id %s --plan <plan text> [--agent-review] [--human-review]",
	task.EventSpecificationReviewAgentApproved: "specification review agent approved --id %s " +
		"[--comment <text>]",
	task.EventSpecificationReviewAgentRejected: "specification review agent rejected --id %s " +
		"--finding <location>=<detail> [...]",
	task.EventSpecificationReviewHumanApproved: "specification review human approved --id %s " +
		"[--comment <text>]",
	task.EventSpecificationReviewHumanRejected: "specification review human rejected --id %s --reason <text>",
	task.EventImplementationCompleted: "implemented --id %s --worktree <path> --branch <name> " +
		"[--unit] [--integration] [--end-to-end] [--linters] [--agent-review] [--human-review]",
	task.EventVerificationPassed: "verified --id %s --unit <ok|error> --integration <ok|error> " +
		"--end-to-end <ok|error> --linters <ok|error> [--output <text>]",
	task.EventVerificationFailed: "verified --id %s --unit <ok|error> --integration <ok|error> " +
		"--end-to-end <ok|error> --linters <ok|error> [--output <text>]",
	task.EventImplementationReviewAgentApproved: "implementation review agent approved --id %s " +
		"[--comment <text>]",
	task.EventImplementationReviewAgentRejected: "implementation review agent rejected --id %s " +
		"--finding <location>=<detail> [...]",
	task.EventCommitRecorded: "committed --id %s",
	task.EventImplementationReviewHumanApproved: "implementation review human approved --id %s " +
		"[--comment <text>]",
	task.EventImplementationReviewHumanRejected: "implementation review human rejected --id %s --reason <text>",
	task.EventMergeCompleted:                    "merged --id %s --target <branch>",
	task.EventUnblocked:                         "unblocked --id %s --reason <text> [--rounds <n>]",
}

// CommandsFor lists every command that would currently be accepted for t,
// derived from task.ValidEvents so it never drifts from the workflow's own
// guards. Distinct events that report through the same command (verified's
// pass/fail) collapse into one entry.
func CommandsFor(t task.Task) []string {
	seen := make(map[string]bool)
	commands := make([]string, 0, len(commandNames))
	for _, kind := range t.ValidEvents() {
		name, ok := commandNames[kind]
		if !ok {
			continue
		}
		command := fmt.Sprintf(name, t.ID)
		if seen[command] {
			continue
		}
		seen[command] = true
		commands = append(commands, command)
	}
	sort.Strings(commands)
	return commands
}
