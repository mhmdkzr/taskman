# taskman

A small CLI for creating, listing, and querying the `.tasks/*.md` task files
that live under `<package>/.tasks/` throughout this repo. Each task is a
Markdown file with YAML frontmatter (metadata) plus a short prose body, so
tasks can be filtered by field (package, urgency, importance, type, status,
tag, dependency) instead of grepping free text.

There is no combined index file and no `NONE.md` marker — an empty or
missing `.tasks/` directory just means there are no tasks for that package.

## Build

```bash
cd scripts/taskman
GOWORK=off go build -o taskman .
```

(`GOWORK=off` because this is a standalone module — the repo's `go.work`
doesn't list it as a workspace member.)

## Task schema

Frontmatter fields:

| Field        | Values                                                            |
|--------------|-------------------------------------------------------------------|
| `id`         | `<package>/<slug>` — matches the file's own path, globally unique |
| `package`    | package path relative to repo root, e.g. `internal/routes`        |
| `title`      | short (a few words) — a title, not a sentence                    |
| `status`     | `open` \| `done` \| `dropped`                                     |
| `urgency`    | `low` \| `medium` \| `high` — time pressure                       |
| `importance` | `low` \| `medium` \| `high` — impact if never done                |
| `type`       | `bug-fix` \| `test-gap` \| `doc-drift` \| `convention` \| `feature` \| `operational` |
| `tags`       | freeform list, for cross-cutting queries                          |
| `depends_on` | list of other tasks' `id` — this task waits on those              |
| `questions`  | list of open questions only the user can answer — see below       |
| `where`      | list of `file:line` citations                                     |
| `source.kind`| `review` \| `session` \| `user-request`                           |
| `source.ref` | free-text pointer (e.g. `REVIEW.md (reviewed ...)`, a design doc) |
| `generated`  | RFC3339 timestamp                                                 |
| `resolved`   | `true` \| `false`                                                 |
| `resolved_at`| RFC3339 timestamp — omitted entirely while `resolved: false`       |

There is deliberately no single `severity` field — it conflated two
independent axes, so `urgency` and `importance` replace it.

Body sections: `## What` (required), `## How` (optional implementation
direction), `## Why` (required), `## Done when` (required), `## Questions`
(only when `questions` is non-empty — rendered from the frontmatter list,
not a separate source of truth), and `## Resolution` (only once
`status != open`).

### Questions

`--question` (repeatable) captures things only the user can answer before
a task is actionable — a design decision, ambiguous scope, a judgment
call — as opposed to something the executing agent could resolve on its
own by reading code. Each occurrence adds one entry to the `questions`
list:

```bash
taskman new --package internal/auth --title "Pick refresh-token strategy" \
  --urgency medium --importance high --type feature \
  --source-kind user-request --source-ref "user request" \
  --question "JWT or opaque tokens?" \
  --question "Support refresh tokens at launch, or defer?" \
  --what "Decide the auth token format before implementation starts." \
  --why "Blocks the auth slice." --done-when "A decision is documented."
```

`taskman list`/`search --has-questions` finds tasks with at least one open
question (also marked with a leading `?` in the table view). `taskman
validate` flags a `done`/`dropped` task that still has unanswered
`questions` — there's no `taskman answer`/edit command yet, so clearing them
currently means recreating the task once they're resolved.

## Usage

```
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
```

`--root` defaults to `.` and is the directory `.tasks` trees are relative
to — run `taskman` from the repo root, or pass `--root` explicitly.
`--package` on `list`/`search`/`validate` matches the package itself and
any subpackage (`--package internal/network` also covers
`internal/network/ton/jetton_wallet`).

Flags with a long/multi-line value (`--what`, `--how`, `--why`,
`--done-when`, `--resolution`, `--reason`) all have a `-file` sibling that
reads the text from a file instead, for anything too long to comfortably
pass as a shell argument.

Flags must come before the positional argument (`show`/`search`/`done`/
`drop` take one) — this is normal Go `flag` package behavior, e.g.:

```bash
taskman show --root . --json internal/routes/validate-basepath-leading-slash
```

### Examples

Create a task:

```bash
taskman new --package internal/routes \
  --title "Validate BasePath leading slash" \
  --urgency medium --importance high --type bug-fix \
  --tags http,config \
  --source-kind review --source-ref "REVIEW.md (reviewed 2026-08-25)" \
  --where "routes.go:28-32" \
  --what "Server.BasePath is not validated before being used to build route patterns." \
  --why "A slash-less BasePath makes joinBasePath produce a pattern stdlib parses as a host pattern, so every route silently 404s." \
  --done-when "A test registers a route with a slash-less BasePath and asserts either normalization or a loud failure."
```

List everything open and important, across the whole repo:

```bash
taskman list --status open --importance high
```

Resolve a task once the fix lands:

```bash
taskman done internal/routes/validate-basepath-leading-slash \
  --resolution "Handle now panics on a slash-less BasePath instead of silently mis-registering routes."
```

Or drop one that's deliberately not being done:

```bash
taskman drop internal/network/ton/jetton_wallet/backfill-jetton-wallet-rows-for-pre-existing-wallets \
  --reason "Deploy order changed; tracked as a separate ops runbook instead."
```

`done`/`drop` never delete the file — they set `status`, flip `resolved` to
`true`, stamp `resolved_at`, and append a `## Resolution` section explaining
what happened. The file
stays as a record either way; query past decisions with `list --status
done` / `list --status dropped`.

### Cross-package dependencies

`depends_on` entries are full `id`s (`<package>/<slug>`), so a task in one
package can depend on a task in another. `new` only warns (doesn't block)
if a dependency doesn't exist yet, since tasks in the same generation batch
may be written concurrently by different agents/processes. Run `taskman
validate` once a batch is done to catch anything still dangling.

## Validate

```bash
taskman validate [--package PATH]
```

Walks every `.tasks/` tree under `--root` (or just one package subtree) and
checks:

- every file's frontmatter parses and every enum value is one of the
  allowed values
- a task's `id`/`package` match where the file actually lives
- no two tasks claim the same `id`
- `status`/`resolved`/`resolved_at`/`## Resolution` are internally consistent
  (`open` ⇒ `resolved: false`, no `resolved_at`, no resolution section;
  `done`/`dropped` ⇒ `resolved: true`, `resolved_at` set, resolution present)
- every `depends_on` entry resolves to an existing task
- the `depends_on` graph has no cycles
- a `done`/`dropped` task doesn't still carry unanswered `questions`

Exits `0` with a summary line if clean, `1` (printing each problem) otherwise.
