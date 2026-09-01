// Command taskman is the human-facing CLI for the SQLite task backlog and
// the codebase → task → commit pipeline: add/list/show operate on
// internal/task directly, and run drives one task through internal/pipeline.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"uuid"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"

	"github.com/mhmdkzr/taskman/internal/codebase"
	"github.com/mhmdkzr/taskman/internal/config"
	"github.com/mhmdkzr/taskman/internal/pipeline"
	"github.com/mhmdkzr/taskman/internal/publisher"
	"github.com/mhmdkzr/taskman/internal/store"
	"github.com/mhmdkzr/taskman/internal/task"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "add":
		err = cmdAdd(os.Args[2:])
	case "list":
		err = cmdList(os.Args[2:])
	case "show":
		err = cmdShow(os.Args[2:])
	case "run":
		err = cmdRun(os.Args[2:])
	case "-h", "--help", "help":
		usage()
		return
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "taskman:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(
		os.Stderr,
		`taskman is the CLI for the task backlog and the codebase -> task -> commit pipeline.

Usage:
  taskman add [flags]     file a new task
  taskman list [-status s] list tasks, optionally filtered by status
  taskman show <id>       print one task in full, as JSON
  taskman run <id> [flags] run the pipeline for one task

Run "taskman <command> -h" for a command's flags.
`,
	)
}

// stringList is a repeatable flag.Value collecting every occurrence into a
// slice, e.g. -package a -package b -> ["a", "b"].
type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func cmdAdd(args []string) error {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	title := fs.String("title", "", "short title (required)")
	taskType := fs.String("type", "", "bug|docs|feature|refactor|test (required)")
	urgency := fs.String("urgency", "", "low|medium|high (required)")
	importance := fs.String("importance", "", "low|medium|high (required)")
	risk := fs.String("risk", "", "low|medium|high (required)")
	what := fs.String("what", "", "what the task is (required)")
	why := fs.String("why", "", "why it matters (required)")
	how := fs.String("how", "", "how to do it (required)")
	variant := fs.String("variant", "medium", "reasoning effort: low|medium|high|xhigh|max|default")
	var packages, completedWhen, where, invariants, tags, deps stringList
	fs.Var(&packages, "package", "package this task touches (repeatable)")
	fs.Var(
		&completedWhen,
		"completed-when",
		"a condition that must hold when done (repeatable, required)",
	)
	fs.Var(&where, "where", "file:line citation (repeatable)")
	fs.Var(&invariants, "invariant", "something that must stay true (repeatable)")
	fs.Var(&tags, "tag", "tag (repeatable)")
	fs.Var(&deps, "depends-on", "id of a task this one depends on (repeatable)")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	for name, v := range map[string]string{
		"-title": *title, "-type": *taskType, "-urgency": *urgency,
		"-importance": *importance, "-risk": *risk, "-what": *what, "-why": *why, "-how": *how,
	} {
		if v == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	if len(completedWhen) == 0 {
		return fmt.Errorf("at least one -completed-when is required")
	}

	db, err := openDB()
	if err != nil {
		return err
	}
	defer closeQuietly(db)

	t := task.Task{
		ID:            task.NewTaskID(),
		Title:         *title,
		TaskType:      *taskType,
		Urgency:       *urgency,
		Importance:    *importance,
		Risk:          *risk,
		What:          *what,
		Why:           *why,
		How:           *how,
		Where:         []string(where),
		Invariants:    []string(invariants),
		Packages:      []string(packages),
		CompletedWhen: []string(completedWhen),
		Tags:          []string(tags),
		Dependencies:  []string(deps),
		Variant:       *variant,
	}
	ctx := context.Background()
	if err := task.CreateTask(ctx, db, t); err != nil {
		return fmt.Errorf("create task: %w", err)
	}
	fmt.Println(t.ID)
	return nil
}

func cmdList(args []string) error {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	status := fs.String("status", "", "filter by status: created|started|completed|reviewed")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	db, err := openDB()
	if err != nil {
		return err
	}
	defer closeQuietly(db)

	ctx := context.Background()
	tasks, err := task.ListTasks(ctx, db, task.TaskStatus(*status))
	if err != nil {
		return fmt.Errorf("list tasks: %w", err)
	}
	if len(tasks) == 0 {
		fmt.Println("no tasks")
		return nil
	}
	for _, t := range tasks {
		fmt.Printf("%s  %-9s %-8s %s\n", t.ID, t.Status, t.TaskType, t.Title)
	}
	return nil
}

func cmdShow(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: taskman show <id>")
	}
	id, err := uuid.Parse(args[0])
	if err != nil {
		return fmt.Errorf("invalid task id %q: %w", args[0], err)
	}

	db, err := openDB()
	if err != nil {
		return err
	}
	defer closeQuietly(db)

	t, err := task.GetTask(context.Background(), db, id)
	if err != nil {
		return fmt.Errorf("get task: %w", err)
	}
	b, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	sourceDir := fs.String("source", ".", "git repository to run the task against")
	workDir := fs.String(
		"workdir",
		"",
		"parent directory for the task's isolated worktree (default: a fresh temp dir)",
	)
	maxFixupRounds := fs.Int(
		"max-fixup-rounds",
		0,
		"cap on lint/review fixup rounds per phase (default: pipeline's own default)",
	)
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: taskman run <id> [flags]")
	}
	id, err := uuid.Parse(fs.Arg(0))
	if err != nil {
		return fmt.Errorf("invalid task id %q: %w", fs.Arg(0), err)
	}

	agentCfg, err := loadAgentConfig()
	if err != nil {
		return fmt.Errorf("load agent config: %w", err)
	}
	if agentCfg.Provider.BaseURL == "" || agentCfg.Provider.APIKey == "" {
		return fmt.Errorf(
			"AGENT_PROVIDER_BASE_URL and AGENT_PROVIDER_API_KEY must be set to run a task",
		)
	}

	st, err := store.Open(agentCfg.DBPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer closeQuietly(st)
	if err := store.Migrate(st.RW()); err != nil {
		return fmt.Errorf("migrate store: %w", err)
	}

	t, err := task.GetTask(context.Background(), st.RW(), id)
	if err != nil {
		return fmt.Errorf("get task: %w", err)
	}

	source, err := filepath.Abs(*sourceDir)
	if err != nil {
		return fmt.Errorf("resolve source dir: %w", err)
	}
	work := *workDir
	if work == "" {
		work, err = os.MkdirTemp("", "taskman-run-")
		if err != nil {
			return fmt.Errorf("create work dir: %w", err)
		}
	}

	cfg := pipeline.Config{
		Store:     st,
		Publisher: publisher.Publisher{},
		Agent:     agentCfg,
		SourceDir: source,
		WorkDir:   work,
		Author: codebase.AuthorSignature{
			Name:  "taskman-pipeline",
			Email: "pipeline@taskman.local",
		},
		MaxFixupRounds: *maxFixupRounds,
	}

	fmt.Printf("running task %s against %s (worktree under %s)...\n", t.ID, source, work)
	result, err := pipeline.RunTask(context.Background(), cfg, *t)
	if err != nil {
		return fmt.Errorf("run task: %w", err)
	}

	fmt.Printf("commit:            %s\n", result.CommitHash)
	fmt.Printf("execution session: %s\n", result.ExecutionSessionID)
	fmt.Printf("review session:    %s\n", result.ReviewSessionID)
	return nil
}

// envConfig mirrors config.Config's Agent field so AgentConfig's env tags
// (env:"PROVIDER_" etc.) resolve under the AGENT_ prefix exactly as they do
// when the full application config loads, without requiring the NATS/Server
// config sections the CLI has no use for.
type envConfig struct {
	Agent config.AgentConfig `envPrefix:"AGENT_"`
}

// loadAgentConfig loads .env (unless SKIP_ENV_AUTO_LOAD is set, matching
// config.Config.Load) and parses just the agent configuration, normalized
// (DB path defaulted and ~-expanded) the same way config.Config.AgentOptions
// does. Unlike the full application config, provider credentials are not
// required here — commands that don't call an LLM (add/list/show) have no
// use for them; cmdRun checks for them explicitly since it does.
func loadAgentConfig() (config.AgentConfig, error) {
	if strings.ToLower(os.Getenv("SKIP_ENV_AUTO_LOAD")) != "true" {
		if err := godotenv.Load(); err != nil {
			fmt.Fprintln(os.Stderr, "taskman: no .env loaded:", err)
		}
	}
	var ec envConfig
	if err := env.Parse(&ec); err != nil {
		return config.AgentConfig{}, fmt.Errorf("parse environment config: %w", err)
	}
	a := ec.Agent
	if a.DBPath == "" {
		a.DBPath = config.DefaultDBPath
	}
	dbPath, err := config.ExpandHome(a.DBPath)
	if err != nil {
		return config.AgentConfig{}, fmt.Errorf("expand db path: %w", err)
	}
	a.DBPath = dbPath
	return a, nil
}

// openDB opens (and migrates) the task store's *sql.DB for add/list/show,
// which don't need the full agent config the run command loads.
func openDB() (*sql.DB, error) {
	agentCfg, err := loadAgentConfig()
	if err != nil {
		return nil, fmt.Errorf("load agent config: %w", err)
	}
	db, err := store.OpenDB(agentCfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}
	if err := store.Migrate(db); err != nil {
		return nil, fmt.Errorf("migrate store: %w", err)
	}
	return db, nil
}

// closeQuietly closes c (a *sql.DB or *store.Store, both io.Closers) and
// reports a failure to stderr rather than silently discarding it — the
// command's own result has already been reported by the time a deferred
// Close runs, so there's nothing left to return the error to.
func closeQuietly(c io.Closer) {
	if err := c.Close(); err != nil {
		fmt.Fprintln(os.Stderr, "taskman: close:", err)
	}
}
