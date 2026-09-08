package mcp

import (
	"context"
	"fmt"
	"uuid"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mhmdkzr/loop/internal/app"
	"github.com/mhmdkzr/loop/internal/task"
)

// addTaskTools registers read-only lookups over loop's tasks: definition,
// specification, planning levels, state, and labels. Not exposed here:
// creating, updating, or deleting a task, or which sessions it's linked to.
func addTaskTools(s *gomcp.Server, a app.App) {
	gomcp.AddTool(s, &gomcp.Tool{
		Name:        "task_list",
		Description: "List loop's tasks using optional state, label, planning, model, or commit filters.",
	}, taskListHandler(a))

	gomcp.AddTool(s, &gomcp.Tool{
		Name:        "task_get",
		Description: "Get a task's definition, specification, planning levels, state, and labels, by ID.",
	}, taskGetHandler(a))
}

// mcpTask mirrors task.Task with its ID as a plain string: task.Task.ID is a
// uuid.UUID, which jsonschema-go infers as a 16-byte array rather than the
// string it actually marshals to (uuid.UUID implements MarshalText, but
// schema inference looks at the underlying type, not that method) - every
// other tool package in this repo takes IDs as strings for the same reason.
type mcpTask struct {
	ID            string   `json:"id"`
	Definition    string   `json:"definition"`
	Specification string   `json:"specification"`
	State         string   `json:"state"`
	Labels        []string `json:"labels,omitempty"`
	Importance    int      `json:"importance"`
	Urgency       int      `json:"urgency"`
	Complexity    int      `json:"complexity"`
	Effort        int      `json:"effort"`
	Risk          int      `json:"risk"`
	Autonomy      int      `json:"autonomy"`
	Model         string   `json:"model"`
	CommitHash    string   `json:"commit_hash,omitempty"`
}

func newMCPTask(t task.Task) mcpTask {
	return mcpTask{
		ID: t.ID.String(), Definition: t.Definition, Specification: t.Specification,
		State: string(t.State), Labels: t.Labels,
		Importance: int(t.Importance), Urgency: int(t.Urgency), Complexity: int(t.Complexity),
		Effort: int(t.Effort), Risk: int(t.Risk), Autonomy: int(t.Autonomy),
		Model: t.Model, CommitHash: t.CommitHash,
	}
}

// taskListInput accepts the same filters as task.TaskFilter, but with IDs as
// strings (see mcpTask) rather than task.TaskFilter's []uuid.UUID.
type taskListInput struct {
	IDs          []string `json:"ids,omitempty"           jsonschema:"Task IDs to match."`
	State        []string `json:"state,omitempty"         jsonschema:"Task states to match (created, started, completed, cancelled, blocked, failed)."`
	Labels       []string `json:"labels,omitempty"        jsonschema:"Labels a task must have at least one of."`
	Importance   []int    `json:"importance,omitempty"    jsonschema:"Importance levels to match, 1-5."`
	Urgency      []int    `json:"urgency,omitempty"       jsonschema:"Urgency levels to match, 1-5."`
	Complexity   []int    `json:"complexity,omitempty"    jsonschema:"Complexity levels to match, 1-5."`
	Effort       []int    `json:"effort,omitempty"        jsonschema:"Effort levels to match, 1-5."`
	Risk         []int    `json:"risk,omitempty"          jsonschema:"Risk levels to match, 1-5."`
	Autonomy     []int    `json:"autonomy,omitempty"      jsonschema:"Autonomy levels to match, 1-5."`
	Model        []string `json:"model,omitempty"         jsonschema:"Models to match."`
	CommitHashes []string `json:"commit_hashes,omitempty" jsonschema:"Commit hashes to match."`
}

func (in taskListInput) toFilter() (task.TaskFilter, error) {
	ids := make([]uuid.UUID, len(in.IDs))
	for i, raw := range in.IDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			return task.TaskFilter{}, fmt.Errorf("parse ids[%d]: %w", i, err)
		}
		ids[i] = id
	}
	return task.TaskFilter{
		IDs: ids, State: toTaskStates(in.State), Labels: in.Labels,
		Importance: toLevels(in.Importance), Urgency: toLevels(in.Urgency),
		Complexity: toLevels(in.Complexity), Effort: toLevels(in.Effort),
		Risk: toLevels(in.Risk), Autonomy: toLevels(in.Autonomy),
		Model: in.Model, CommitHashes: in.CommitHashes,
	}, nil
}

func toTaskStates(values []string) []task.TaskState {
	states := make([]task.TaskState, len(values))
	for i, v := range values {
		states[i] = task.TaskState(v)
	}
	return states
}

func toLevels(values []int) []task.Level {
	levels := make([]task.Level, len(values))
	for i, v := range values {
		levels[i] = task.Level(v)
	}
	return levels
}

type taskListOutput struct {
	Tasks []mcpTask `json:"tasks"`
}

func taskListHandler(a app.App) gomcp.ToolHandlerFor[taskListInput, taskListOutput] {
	return func(ctx context.Context, _ *gomcp.CallToolRequest, in taskListInput) (*gomcp.CallToolResult, taskListOutput, error) {
		filter, err := in.toFilter()
		if err != nil {
			return nil, taskListOutput{}, err
		}
		tasks, err := task.ListTasks(ctx, a.Deps.Store.RO(), filter)
		if err != nil {
			return nil, taskListOutput{}, fmt.Errorf("list tasks: %w", err)
		}
		out := taskListOutput{Tasks: make([]mcpTask, len(tasks))}
		for i, t := range tasks {
			out.Tasks[i] = newMCPTask(t)
		}
		return nil, out, nil
	}
}

type taskGetInput struct {
	ID string `json:"id" jsonschema:"Task ID (see task_list)."`
}

type taskGetOutput struct {
	Task mcpTask `json:"task"`
}

func taskGetHandler(a app.App) gomcp.ToolHandlerFor[taskGetInput, taskGetOutput] {
	return func(ctx context.Context, _ *gomcp.CallToolRequest, in taskGetInput) (*gomcp.CallToolResult, taskGetOutput, error) {
		id, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, taskGetOutput{}, fmt.Errorf("parse id: %w", err)
		}
		t, err := task.GetTask(ctx, a.Deps.Store.RO(), id)
		if err != nil {
			return nil, taskGetOutput{}, fmt.Errorf("get task: %w", err)
		}
		return nil, taskGetOutput{Task: newMCPTask(t)}, nil
	}
}
