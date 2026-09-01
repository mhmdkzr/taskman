package components

import "encoding/json"

// toolTitle returns the human display title for a tool name, e.g.
// "go_test" -> "Test", "edit" -> "Edit". Unknown tools keep their name.
func toolTitle(name string) string {
	switch name {
	case "edit":
		return "Edit"
	case "read":
		return "Read"
	case "glob":
		return "Glob"
	case "grep":
		return "Grep"
	case "go_build":
		return "Build"
	case "go_test":
		return "Test"
	case "task_create":
		return "Create Task"
	case "task_edit":
		return "Edit Task"
	case "task_get":
		return "Get Task"
	case "task_search":
		return "Search Tasks"
	case "spawn_subagent":
		return "Spawn Subagent"
	case "subagent_result":
		return "Subagent Result"
	case "telegram_send":
		return "Send Telegram"
	case "telegram_read":
		return "Read Telegram"
	case "datetime":
		return "Datetime"
	default:
		return name
	}
}

// editArgs is the parsed input of the edit tool.
type editArgs struct {
	Path       string `json:"path"`
	Old        string `json:"old"`
	New        string `json:"new"`
	ReplaceAll bool   `json:"replace_all"`
}

// readArgs is the parsed input of the read tool.
type readArgs struct {
	Path   string `json:"path"`
	Offset *int   `json:"offset"`
	Limit  *int   `json:"limit"`
}

// globArgs is the parsed input of the glob tool.
type globArgs struct {
	Pattern string  `json:"pattern"`
	Path    *string `json:"path"`
}

// grepArgs is the parsed input of the grep tool.
type grepArgs struct {
	Pattern string  `json:"pattern"`
	Path    *string `json:"path"`
	Include *string `json:"include"`
	Limit   *int    `json:"limit"`
}

// parseToolInput decodes a tool call's JSON input into out, best-effort: on
// empty or non-JSON input (tools like bash return plain text) out is left at
// its zero value. Unmarshal errors are discarded deliberately — the input is
// untrusted and the renderers fall back to showing it as-is.
func parseToolInput(input string, out any) {
	if input == "" {
		return
	}
	if err := json.Unmarshal([]byte(input), out); err != nil {
		return
	}
}

// parseEdit decodes an edit tool's input; on failure it returns the zero
// value, so the renderer shows whatever fields were present.
func parseEdit(input string) editArgs {
	var a editArgs
	parseToolInput(input, &a)
	return a
}

// parseRead decodes a read tool's input.
func parseRead(input string) readArgs {
	var a readArgs
	parseToolInput(input, &a)
	return a
}

// parseGlob decodes a glob tool's input.
func parseGlob(input string) globArgs {
	var a globArgs
	parseToolInput(input, &a)
	return a
}

// parseGrep decodes a grep tool's input.
func parseGrep(input string) grepArgs {
	var a grepArgs
	parseToolInput(input, &a)
	return a
}
