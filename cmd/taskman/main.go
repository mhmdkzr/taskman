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
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"uuid"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/mhmdkzr/taskman/internal/codebase"
	"github.com/mhmdkzr/taskman/internal/config"
	"github.com/mhmdkzr/taskman/internal/pipeline"
	"github.com/mhmdkzr/taskman/internal/publisher"
	"github.com/mhmdkzr/taskman/internal/store"
	"github.com/mhmdkzr/taskman/internal/streams"
	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/web"
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
	case "web", "-web":
		err = cmdWeb(os.Args[2:])
	case "reset":
		err = cmdReset(os.Args[2:])
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
  taskman run <id> [<id> ...] [flags] run the pipeline for one or more tasks (concurrently)
  taskman reset <id>      return a stuck task to created so it can be re-run
  taskman web [-addr] [flags] serve the read-only realtime dashboard (blocking)

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

// parseFlagsAroundPositionals parses fs against args, tolerating any number
// of positional arguments (e.g. task ids) in any position relative to the
// flags — flag.FlagSet.Parse on its own stops at the first non-flag
// argument, so "run <id> -source x" would silently leave -source
// unparsed and defaulted instead of erroring, since flag.Parse treats
// everything from the id onward as positional. This re-parses the
// remainder after each positional argument it finds, so flags before and
// after the ids both take effect. It returns the positional arguments
// found (nil if none) or an error if a flag fails to parse.
func parseFlagsAroundPositionals(fs *flag.FlagSet, args []string) ([]string, error) {
	var positionals []string
	rest := args
	for {
		if err := fs.Parse(rest); err != nil {
			return nil, fmt.Errorf("parse flags: %w", err)
		}
		if fs.NArg() == 0 {
			return positionals, nil
		}
		positionals = append(positionals, fs.Arg(0))
		rest = fs.Args()[1:]
	}
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

// cmdReset returns a task stuck mid-lifecycle (e.g. a `run` that crashed
// between phases) back to TaskStatusCreated, so `taskman run <id>` can be
// retried. See task.ResetTask for exactly what it clears.
func cmdReset(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: taskman reset <id>")
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

	if err := task.ResetTask(context.Background(), db, id); err != nil {
		return fmt.Errorf("reset task: %w", err)
	}
	fmt.Printf("task %s reset to %s\n", id, task.TaskStatusCreated)
	return nil
}

func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	sourceDir := fs.String("source", ".", "git repository to run the task against")
	workDir := fs.String(
		"workdir",
		"",
		"parent directory for each task's isolated worktree (default: a fresh temp dir)",
	)
	maxFixupRounds := fs.Int(
		"max-fixup-rounds",
		0,
		"cap on lint/review fixup rounds per phase (default: pipeline's own default)",
	)
	rawIDs, err := parseFlagsAroundPositionals(fs, args)
	if err != nil {
		return err
	}
	if len(rawIDs) == 0 {
		return fmt.Errorf("usage: taskman run <id> [<id> ...] [flags]")
	}
	ids := make([]uuid.UUID, 0, len(rawIDs))
	for _, rawID := range rawIDs {
		id, err := uuid.Parse(rawID)
		if err != nil {
			return fmt.Errorf("invalid task id %q: %w", rawID, err)
		}
		ids = append(ids, id)
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

	tasks := make([]task.Task, 0, len(ids))
	for _, id := range ids {
		t, err := task.GetTask(context.Background(), st.RW(), id)
		if err != nil {
			return fmt.Errorf("get task %s: %w", id, err)
		}
		tasks = append(tasks, *t)
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

	pub, closeNATS := connectPublisher()
	defer closeNATS()

	cfg := pipeline.Config{
		Store:     st,
		Publisher: pub,
		Agent:     agentCfg,
		SourceDir: source,
		WorkDir:   work,
		Author: codebase.AuthorSignature{
			Name:  "taskman-pipeline",
			Email: "pipeline@taskman.local",
		},
		MaxFixupRounds: *maxFixupRounds,
	}

	fmt.Printf("running %d task(s) against %s (worktrees under %s)...\n", len(tasks), source, work)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type outcome struct {
		id     uuid.UUID
		result *pipeline.Result
		err    error
	}
	results := make(chan outcome, len(tasks))
	for _, t := range tasks {
		go func(t task.Task) {
			res, err := pipeline.RunTask(ctx, cfg, t)
			results <- outcome{id: t.ID, result: res, err: err}
		}(t)
	}

	var failures int
	for range tasks {
		o := <-results
		if o.err != nil {
			failures++
			fmt.Fprintf(os.Stderr, "task %s failed: %v\n", o.id, o.err)
			continue
		}
		fmt.Printf("task %s:\n", o.id)
		fmt.Printf("  commit:            %s\n", o.result.CommitHash)
		fmt.Printf("  execution session: %s\n", o.result.ExecutionSessionID)
		fmt.Printf("  review session:    %s\n", o.result.ReviewSessionID)
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d task(s) failed", failures, len(tasks))
	}
	return nil
}

// cmdWeb serves the read-only realtime dashboard (see internal/web) and
// blocks until interrupted. Commands posted to it (e.g. run tasks) are
// executed through the pipeline, so it also needs a provider and a source
// repository, like `run`.
func cmdWeb(args []string) error {
	fs := flag.NewFlagSet("web", flag.ExitOnError)
	addr := fs.String("addr", "127.0.0.1:8080", "address to listen on")
	sourceDir := fs.String("source", ".", "git repository to run tasks against")
	workDir := fs.String(
		"workdir",
		"",
		"parent directory for each task's isolated worktree (default: a fresh temp dir)",
	)
	maxFixupRounds := fs.Int(
		"max-fixup-rounds",
		0,
		"cap on lint/review fixup rounds per phase (default: pipeline's own default)",
	)
	if err := fs.Parse(args); err != nil {
		return err
	}

	agentCfg, err := loadAgentConfig()
	if err != nil {
		return fmt.Errorf("load agent config: %w", err)
	}
	if agentCfg.Provider.BaseURL == "" || agentCfg.Provider.APIKey == "" {
		return fmt.Errorf(
			"AGENT_PROVIDER_BASE_URL and AGENT_PROVIDER_API_KEY must be set to run tasks from the web UI",
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

	source, err := filepath.Abs(*sourceDir)
	if err != nil {
		return fmt.Errorf("resolve source dir: %w", err)
	}
	work := *workDir
	if work == "" {
		work, err = os.MkdirTemp("", "taskman-web-")
		if err != nil {
			return fmt.Errorf("create work dir: %w", err)
		}
	}

	pub, closeNATS := connectPublisher()
	defer closeNATS()

	pipeCfg := pipeline.Config{
		Store:     st,
		Publisher: pub,
		Agent:     agentCfg,
		SourceDir: source,
		WorkDir:   work,
		Author: codebase.AuthorSignature{
			Name:  "taskman-pipeline",
			Email: "pipeline@taskman.local",
		},
		MaxFixupRounds: *maxFixupRounds,
	}

	srv, err := web.New(web.Config{
		Addr:     *addr,
		Store:    st,
		Pub:      pub,
		Pipeline: pipeCfg,
	})
	if err != nil {
		return fmt.Errorf("web: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return srv.Run(ctx)
}

// defaultNATSURL matches the URL docker compose exposes NATS on locally (see
// .env.example); NATS_URL overrides it.
const defaultNATSURL = "nats://127.0.0.1:4222"

// connectPublisher best-effort connects to NATS and ensures the agent/
// scheduler event streams exist, returning a live Publisher on success. NATS
// is optional for `run`: on any failure to connect, init JetStream, or
// create streams, it reports why to stderr and returns a zero-value
// Publisher instead — every Publish call through that is a no-op (see
// Session.publish's best-effort error handling), so the task still runs,
// just without live event visibility. The returned func closes the
// connection, if one was made; always defer it.
func connectPublisher() (publisher.Publisher, func()) {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = defaultNATSURL
	}

	nc, err := nats.Connect(natsURL, nats.Timeout(2*time.Second))
	if err != nil {
		fmt.Fprintf(os.Stderr, "taskman: NATS not reachable at %s, continuing without live events: %v\n", natsURL, err)
		return publisher.Publisher{}, func() {}
	}

	js, err := jetstream.New(nc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "taskman: jetstream init failed, continuing without live events: %v\n", err)
		nc.Close()
		return publisher.Publisher{}, func() {}
	}
	if err := streams.CreateStreams(context.Background(), js); err != nil {
		fmt.Fprintf(os.Stderr, "taskman: create streams failed, continuing without live events: %v\n", err)
		nc.Close()
		return publisher.Publisher{}, func() {}
	}

	fmt.Printf("connected to NATS at %s (publishing agent.>/scheduler.> events)\n", natsURL)
	return publisher.NewPublisher(nc), nc.Close
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
