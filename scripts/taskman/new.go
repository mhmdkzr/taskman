package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

func cmdNew(args []string) {
	fs := flag.NewFlagSet("new", flag.ExitOnError)
	root := fs.String("root", ".", "repo root that .tasks trees are relative to")
	pkg := fs.String("package", "", "package path, e.g. internal/routes (required)")
	title := fs.String("title", "", "task title (required)")
	slug := fs.String("slug", "", "override the title-derived slug")
	urgency := fs.String("urgency", "", "low|medium|high — time pressure (required)")
	importance := fs.String("importance", "", "low|medium|high — impact if never done (required)")
	typ := fs.String("type", "", "bug-fix|test-gap|doc-drift|convention|feature|operational (required)")
	tags := fs.String("tags", "", "comma-separated tags")
	dependsOn := fs.String("depends-on", "", "comma-separated task ids this task depends on")
	var questions repeatedFlag
	fs.Var(&questions, "question", "a question that needs the user's answer before this task is actionable (repeatable)")
	where := fs.String("where", "", "comma-separated file:line citations")
	sourceKind := fs.String("source-kind", "", "review|session|user-request (required)")
	sourceRef := fs.String("source-ref", "", "free-text source pointer (required)")
	generated := fs.String("generated", "", "RFC3339 timestamp, defaults to now")
	what := fs.String("what", "", "what this task is (required, or --what-file)")
	whatFile := fs.String("what-file", "", "read --what text from file")
	how := fs.String("how", "", "optional implementation direction")
	howFile := fs.String("how-file", "", "read --how text from file")
	why := fs.String("why", "", "why this matters (required, or --why-file)")
	whyFile := fs.String("why-file", "", "read --why text from file")
	doneWhen := fs.String("done-when", "", "checkable completion condition (required, or --done-when-file)")
	doneWhenFile := fs.String("done-when-file", "", "read --done-when text from file")
	fs.Parse(args)

	if *pkg == "" || *title == "" || *urgency == "" || *importance == "" || *typ == "" || *sourceKind == "" || *sourceRef == "" {
		die("new: --package, --title, --urgency, --importance, --type, --source-kind, and --source-ref are required")
	}
	urg := strings.ToLower(*urgency)
	if err := validateEnum("urgency", urg, validUrgency); err != nil {
		die("new: %v", err)
	}
	imp := strings.ToLower(*importance)
	if err := validateEnum("importance", imp, validImportance); err != nil {
		die("new: %v", err)
	}
	ty := strings.ToLower(*typ)
	if err := validateEnum("type", ty, validTypes); err != nil {
		die("new: %v", err)
	}
	sk := strings.ToLower(*sourceKind)
	if err := validateEnum("source.kind", sk, validSourceKinds); err != nil {
		die("new: %v", err)
	}

	whatText, err := textOrFile(*what, *whatFile)
	if err != nil {
		die("new: --what-file: %v", err)
	}
	howText, err := textOrFile(*how, *howFile)
	if err != nil {
		die("new: --how-file: %v", err)
	}
	whyText, err := textOrFile(*why, *whyFile)
	if err != nil {
		die("new: --why-file: %v", err)
	}
	doneWhenText, err := textOrFile(*doneWhen, *doneWhenFile)
	if err != nil {
		die("new: --done-when-file: %v", err)
	}
	if strings.TrimSpace(whatText) == "" || strings.TrimSpace(whyText) == "" || strings.TrimSpace(doneWhenText) == "" {
		die("new: --what, --why, and --done-when (or their -file variants) are required")
	}

	s := *slug
	if s == "" {
		s = slugify(*title)
	}
	gen := *generated
	if gen == "" {
		gen = time.Now().Format(time.RFC3339)
	}

	path := taskPath(*root, *pkg, s)
	if _, err := os.Stat(path); err == nil {
		die("new: %s already exists — pass --slug to disambiguate", path)
	}

	// Best-effort dependency check: warn, don't block, since tasks in the
	// same generation batch may be written concurrently by other agents.
	depIDs := splitCSV(*dependsOn)
	if len(depIDs) > 0 {
		existing, _, _ := walkTasks(*root, "")
		known := map[string]bool{}
		for _, t := range existing {
			known[t.ID] = true
		}
		for _, d := range depIDs {
			if !known[d] {
				fmt.Fprintf(os.Stderr, "taskman: new: warning: depends_on %q not found yet (run `taskman validate` once all tasks are written)\n", d)
			}
		}
	}

	fm := frontmatter{
		ID:         *pkg + "/" + s,
		Package:    *pkg,
		Title:      *title,
		Status:     "open",
		Urgency:    urg,
		Importance: imp,
		Type:       ty,
		Tags:       splitCSV(*tags),
		DependsOn:  depIDs,
		Questions:  []string(questions),
		Where:      splitCSV(*where),
		Source:     source{Kind: sk, Ref: *sourceRef},
		Generated:  gen,
		Resolved:   false,
	}
	out, err := renderTask(fm, whatText, howText, whyText, doneWhenText)
	if err != nil {
		die("new: %v", err)
	}
	if err := os.MkdirAll(tasksDir(*root, *pkg), 0o755); err != nil {
		die("new: %v", err)
	}
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		die("new: %v", err)
	}
	fmt.Println(fm.ID)
}
