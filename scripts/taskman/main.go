// taskman is a small CLI for creating, listing, and querying the
// frontmatter-based task files under `<package>/.tasks/*.md` (see
// notes/templates/task.md in the core repo for the human-readable schema
// doc). It embeds the canonical template and uses text/template plus
// gopkg.in/yaml.v3 to generate and parse task files, so task metadata can be
// filtered without grepping free text.
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "new":
		cmdNew(os.Args[2:])
	case "list":
		cmdList(os.Args[2:])
	case "show":
		cmdShow(os.Args[2:])
	case "search":
		cmdSearch(os.Args[2:])
	case "done":
		cmdSetStatus(os.Args[2:], "done", "resolution")
	case "drop":
		cmdSetStatus(os.Args[2:], "dropped", "reason")
	case "validate":
		cmdValidate(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `taskman - manage frontmatter-based .tasks/*.md task files

Usage:
  taskman new      --package PATH --title TEXT --urgency U --importance I --type T
                    --source-kind K --source-ref TEXT
                    (--what TEXT | --what-file FILE)
                    (--why TEXT | --why-file FILE)
                    (--done-when TEXT | --done-when-file FILE)
                    [--how TEXT | --how-file FILE]
                    [--tags a,b] [--depends-on id1,id2] [--where file:line,...]
                    [--question TEXT]... [--slug SLUG] [--generated RFC3339] [--root PATH]

  taskman list      [--package PATH] [--urgency U] [--importance I] [--type T]
                     [--status S] [--tag NAME] [--depends-on ID] [--has-questions]
                     [--json] [--root PATH]

  taskman show      ID [--json] [--root PATH]

  taskman search    QUERY [--package PATH] [--urgency U] [--importance I]
                     [--type T] [--status S] [--tag NAME] [--has-questions]
                     [--json] [--root PATH]

  taskman done      ID (--resolution TEXT | --resolution-file FILE) [--root PATH]
  taskman drop      ID (--reason TEXT | --reason-file FILE) [--root PATH]

  taskman validate  [--package PATH] [--root PATH]

ID is "<package>/<slug>", e.g. internal/routes/validate-basepath-leading-slash.
urgency/importance are each low|medium|high (there is no single "severity" —
urgency is time pressure, importance is impact if never done).
--question is repeatable — one flag per question that needs the user's own
answer/judgment before the task is actionable (design decisions, ambiguous
scope, anything the executing agent can't resolve on its own).
--root defaults to the current directory.`)
}
