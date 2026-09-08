# Task State Machine (Mermaid)

Companion to `task-fsm.html` (the hand-drawn SVG version) - same machine, rendered via Mermaid
instead, kept side by side to compare rather than to replace either. Source of truth for the
design itself is `design.md` §6.

```mermaid
stateDiagram-v2
    [*] --> Definition : task create
    Definition --> Specification : task specify
    Specification --> Implementation : task implement
    Implementation --> VerifyReview : diff ready

    state "Verification & review" as VerifyReview {
        state "Build check\n(task verify)" as BuildCheck
        state "Fix\n(agent, dispatched by loop)" as Fix
        state "Automated review\n(task review record)" as AutoReview
        state "Human review\n(task review approve/reject)" as HumanReview

        [*] --> BuildCheck
        BuildCheck --> Fix : fail
        Fix --> BuildCheck : re-check
        BuildCheck --> AutoReview : pass
        AutoReview --> Fix : rejected · attempt 1
        AutoReview --> HumanReview : approved
        HumanReview --> Fix : reject → fix
    }

    VerifyReview --> Merge : approve
    VerifyReview --> Blocked : rejected · attempt 2
    Fix --> Blocked : escalate
    Implementation --> Failed : abandon (any time)
    Blocked --> Failed : abandon
    Blocked --> VerifyReview : resume
    Merge --> Done : merged
    Done --> [*]
    Failed --> [*]
```

Two edges (`Fix --> Blocked`, `Blocked --> VerifyReview`) cross into/out of the nested composite
state (`Verification & review`) from outside it - Mermaid support for that pattern varies by
renderer, so this is also a test of whether it renders sensibly at all here.
