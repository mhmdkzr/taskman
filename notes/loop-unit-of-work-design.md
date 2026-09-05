# Loop — Unit of Work Design

A conceptual design for how loop represents work: what a unit of work *is*, what properties it carries, and how units relate to each other. This document covers concepts and reasoning only — no database schema.

## 1. Motivation

Loop needs to describe work at two different altitudes. An agent needs a concrete, execution-focused instruction it can act on directly — something small enough to attempt in one pass, review, and land as a commit. A human needs a bigger, outcome-focused frame to reason about *why* a batch of those instructions exists, what it's worth, and when it needs to land. Collapsing these into one flat "task" concept (as most PM tools default to) forces either the execution unit to carry meaning it doesn't have yet, or the outcome unit to pretend it's as concrete as a single step. Splitting them lets each carry only the properties that are actually knowable at that altitude.

## 2. Two kinds of work: Task and Goal

There are exactly two things in this design — no shared supertype above them:

- **Task** — execution-focused. Small enough for one agent pass. Where complexity, effort, and actual cost live.
- **Goal** — outcome-focused. Bigger than one pass. Where value, budget, and deadline live. A Goal exists to justify a cluster of Tasks; a Task exists to make progress on a Goal.

Tasks and goals can have dependencies in form of a DAG. They also have unique identifiers that allows them to be referenced.

## 3. Six aspects

### Value — flows top-down, lives on Goal only
A Goal exists *because* it's valuable; a Task borrows that value by being a step toward it. 

### Complexity — lives on Task, Fibonacci scale
How hard a piece of work is to *understand and get right*. This is only crisply knowable at the point of execution, which is why it belongs to Task rather than the Goal. A Goal-level complexity number would just be a coarse guess or a rollup of children — neither is the same kind of number as a real per-task estimate, so Goal doesn't carry this field at all; if a coarse advance sense of scale is useful, that's what a rolled-up view of children's complexity is for. We use fibonacci scale to represent complexity.

### Effort — lives on Task, distinct from complexity
How much *resource* (human time, model tokens) a Task is expected to consume once it's understood. Kept separate from complexity because they answer different questions and can diverge sharply: a task can be easy to understand but mechanically large (rename a variable across five hundred files — low complexity, high effort [TODO: find a better example]), or hard to understand but small to execute (a one-line fix to a subtle race condition — high complexity, low effort). Complexity mainly informs *routing* (how much autonomy to grant, how likely review will be needed); effort mainly informs *budgeting* (how many tokens/how much time to allot). Conflating them into one number, the way classic story points deliberately do for human teams, would lose the ability to make those two decisions independently — and for an agent pipeline specifically, the two decisions genuinely want different inputs.

### Cost vs. Budget
These are not the same field pointed in two directions; they're two different *kinds* of number that happen to share a topic.

- **Cost** is measured — it accumulates as work happens, only ever known in arrears, at the Task level (driven by actual token/resource consumption).
- **Budget** is declared — a ceiling set in advance, and it makes sense at the Goal level, because "how much am I willing to spend on this" is a decision about the outcome. A Task's budget can only work as a safeguard.

Cost rolls up from Tasks to give a Goal's running total; Budget is compared against that rolled-up total to see how much room is left. They're related but not conflated into one field.

### Risk
Risk means something different at each altitude.

- **Goal-level risk** is about the work itself being worth doing: requirements volatility, whether the outcome is even the right thing to build. [TODO: needs to change]
- **Task-level risk** is about execution safety: the likelihood or consequence of this specific change being wrong, and by extension how much human oversight it needs.

A Goal can be low-risk overall while containing one high-risk Task (e.g., a routine feature that happens to touch the multi-sig threshold logic once). Because these are different questions, risk is never summed or averaged from Task up to Goal — each altitude reports its own, on its own rubric. [TODO: needs to change]

### Deadlines
We care about time for **deadline**: a target date set on a Goal, constraining when the work under it needs to land. If required, we can track time on a task level, and have deadlines for it as well, but the important deadline is always at goal level, each task can drift as long as we reach the goal deadline.
