package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"
)

// listFlags are shared by list and search.
type listFlags struct {
	root         string
	pkg          string
	urgency      string
	importance   string
	typ          string
	status       string
	tag          string
	dependsOn    string
	hasQuestions bool
	asJSON       bool
}

func addListFlags(fs *flag.FlagSet) *listFlags {
	lf := &listFlags{}
	fs.StringVar(&lf.root, "root", ".", "repo root that .tasks trees are relative to")
	fs.StringVar(&lf.pkg, "package", "", "restrict to this package (and its subpackages)")
	fs.StringVar(&lf.urgency, "urgency", "", "filter: low|medium|high")
	fs.StringVar(&lf.importance, "importance", "", "filter: low|medium|high")
	fs.StringVar(&lf.typ, "type", "", "filter: bug-fix|test-gap|doc-drift|convention|feature|operational")
	fs.StringVar(&lf.status, "status", "", "filter: open|done|dropped")
	fs.StringVar(&lf.tag, "tag", "", "filter: must have this tag")
	fs.StringVar(&lf.dependsOn, "depends-on", "", "filter: tasks that depend on this id")
	fs.BoolVar(&lf.hasQuestions, "has-questions", false, "filter: tasks with at least one open question for the user")
	fs.BoolVar(&lf.asJSON, "json", false, "output JSON instead of a table")
	return lf
}

func (lf *listFlags) matches(t task) bool {
	if lf.urgency != "" && !strings.EqualFold(t.Urgency, lf.urgency) {
		return false
	}
	if lf.importance != "" && !strings.EqualFold(t.Importance, lf.importance) {
		return false
	}
	if lf.typ != "" && !strings.EqualFold(t.Type, lf.typ) {
		return false
	}
	if lf.status != "" && !strings.EqualFold(t.Status, lf.status) {
		return false
	}
	if lf.tag != "" && !slices.Contains(t.Tags, lf.tag) {
		return false
	}
	if lf.dependsOn != "" && !slices.Contains(t.DependsOn, lf.dependsOn) {
		return false
	}
	if lf.hasQuestions && len(t.Questions) == 0 {
		return false
	}
	return true
}

func printTasks(tasks []task, asJSON bool) {
	if asJSON {
		b, _ := json.MarshalIndent(tasks, "", "  ")
		fmt.Println(string(b))
		return
	}
	for _, t := range tasks {
		marker := " "
		if len(t.Questions) > 0 {
			marker = "?"
		}
		fmt.Printf("%s %-60s %-6s %-6s %-8s %-11s %s\n", marker, t.ID, t.Urgency, t.Importance, t.Type, t.Status, t.Title)
	}
}

func printWarnings(warnings []string) {
	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, "taskman: warning:", w)
	}
}

func cmdList(args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	lf := addListFlags(fs)
	fs.Parse(args)

	tasks, warnings, err := walkTasks(lf.root, lf.pkg)
	if err != nil {
		die("list: %v", err)
	}
	printWarnings(warnings)
	out := tasks[:0]
	for _, t := range tasks {
		if lf.matches(t) {
			out = append(out, t)
		}
	}
	printTasks(out, lf.asJSON)
}

func cmdShow(args []string) {
	fs := flag.NewFlagSet("show", flag.ExitOnError)
	root := fs.String("root", ".", "repo root that .tasks trees are relative to")
	asJSON := fs.Bool("json", false, "output JSON instead of raw markdown")
	fs.Parse(args)
	if fs.NArg() != 1 {
		die("show: expected exactly one task id")
	}
	t, err := findTask(*root, fs.Arg(0))
	if err != nil {
		die("show: %v", err)
	}
	if *asJSON {
		b, _ := json.MarshalIndent(t, "", "  ")
		fmt.Println(string(b))
		return
	}
	b, err := os.ReadFile(t.Path)
	if err != nil {
		die("show: %v", err)
	}
	fmt.Print(string(b))
}

func cmdSearch(args []string) {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	lf := addListFlags(fs)
	fs.Parse(args)
	if fs.NArg() != 1 {
		die("search: expected exactly one query argument")
	}
	query := strings.ToLower(fs.Arg(0))

	tasks, warnings, err := walkTasks(lf.root, lf.pkg)
	if err != nil {
		die("search: %v", err)
	}
	printWarnings(warnings)
	out := tasks[:0]
	for _, t := range tasks {
		if !lf.matches(t) {
			continue
		}
		haystack := strings.ToLower(strings.Join([]string{
			t.Title, t.What, t.How, t.Why, t.DoneWhen, t.Resolution,
			strings.Join(t.Tags, " "), strings.Join(t.Questions, "\n"),
		}, "\n"))
		if strings.Contains(haystack, query) {
			out = append(out, t)
		}
	}
	printTasks(out, lf.asJSON)
}
