---
name: taskman
description: Create, query, and resolve tasks in a package's `.tasks/` directory via the taskman CLI. Use whenever the user asks to add/file/track a task, or to list/search/resolve existing ones — never hand-write a `.tasks/*.md` file directly.
---

# taskman

CLI for the frontmatter-based task files under `<package>/.tasks/`. Full
schema and examples: `scripts/taskman/README.md`. Build if `taskman` isn't
already on hand:

```bash
cd scripts/taskman && GOWORK=off go build -o taskman .
```

## When the user says "add a task for ..."

Don't transcribe their words literally. Understand what they actually have
in mind — the real problem, its scope, why it matters, what "done" looks
like — then call `taskman new` with fields you chose, not fields you copied.
A short, title-cased `--title` (3-7 words), the right `--type`, and an
honest `--urgency`/`--importance` (not both defaulted to `medium`) matter
more than speed. If something about the task is genuinely undecided and
only the user can decide it (a design choice, ambiguous scope, a judgment
call), capture that with `--question` instead of guessing.

## Commands

```
taskman new      --package PATH --title TEXT --urgency U --importance I --type T
                  --source-kind K --source-ref TEXT
                  (--what TEXT | --what-file FILE)
                  (--why TEXT | --why-file FILE)
                  (--done-when TEXT | --done-when-file FILE)
                  [--how TEXT] [--tags a,b] [--depends-on id1,id2]
                  [--where file:line,...] [--question TEXT]...

taskman list      [--package PATH] [--urgency U] [--importance I] [--type T]
                   [--status S] [--tag NAME] [--depends-on ID] [--has-questions] [--json]

taskman show      ID [--json]
taskman search    QUERY [same filters as list]
taskman done      ID (--resolution TEXT | --resolution-file FILE)
taskman drop      ID (--reason TEXT | --reason-file FILE)
taskman validate  [--package PATH]
```

`--root` defaults to `.` — run from the repo root or pass it explicitly.
Flags go before the positional arg (`show`/`search`/`done`/`drop`).
`--question` is repeatable — one flag per open question.

## Field cheat sheet

- `urgency`/`importance` — each `low`/`medium`/`high`, independent axes
  (no single "severity"). Urgency = time pressure, importance = impact if
  never done.
- `type` — `bug-fix` / `test-gap` / `doc-drift` / `convention` / `feature` /
  `operational`.
- `source-kind` — `review` (traced to a `REVIEW.md`), `session` (this
  conversation's own analysis/design work), or `user-request`.
- `depends_on` — other tasks' full `id` (`<package>/<slug>`); only for a
  genuine blocking prerequisite, not "related." Never create a two-way
  dependency (cycle) — `taskman validate` rejects it.
- `questions` — things only the user can answer before the task is
  actionable. `taskman validate` flags a `done`/`dropped` task that still
  has unanswered ones.

## Rules

- Never hand-write or hand-edit a `.tasks/*.md` file — every mutation goes
  through this CLI.
- Run `taskman validate --package <pkg>` after creating/resolving tasks in
  that package.
- An empty or missing `.tasks/` directory means no tasks — no marker file.
