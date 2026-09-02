---
schema: 1
id: TKT-01M1HVMQQQE3K6VZG7793RXVXN
title: Add optional draft and done directories so tickets/ holds the working set
type: task
status: draft
status_reason: null
priority: normal
labels:
  - format
  - question
assignees: []
milestone: null
parent: null
dependencies: []
references:
  - ref: plan:store-layout
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-02T20:05:53Z
updated_at: 2026-09-02T20:05:53Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

`ls .tickets/tickets/` currently answers a question nobody asks. As of filing it holds 38 files: 27 done, 10 draft, 1 in-progress. So 37 of 38 are not active work, and a human or agent trying to see what is open without running the tool has to read every file to find the one that matters.

Proposal: optional `draft/` and `done/` directories beside `tickets/` and `archive/`, so `tickets/` holds the working set alone. Origin and terminal tickets sit outside it.

Structurally this is not a new kind of thing. `archive/` already is a status-derived directory: 6.3 rules that the status wins when the status and the directory disagree, `archive_location_mismatch` reports the drift, and `check --fix` now repairs it by moving the file. The current placement rule is already a status-to-directory function with one hardcoded entry. This generalizes it.

### Invariant, not opportunistic

The layout should be opt-in, but once opted in it is an invariant like `archive/` is, not a soft preference the tool applies when it happens to touch a file.

If placement were opportunistic, a done ticket left in `tickets/` could not be a finding, or opting in would raise 27 errors at once. Placement would become advisory, and advisory placement rots: done tickets end up spread across two directories indefinitely, which is worse for the legibility goal than not having `done/` at all, because now you check two places and trust neither.

The migration objection that makes opportunistic attractive is already answered. `check --fix` exists, and relocating on opt-in is exactly the move it makes, so the upgrade is one rename-only commit that reviews cleanly. Opportunistic placement is also worse for history: it sprinkles unrelated renames through every commit that happens to touch a ticket, where a single deliberate migration commit is one reviewable diff.

### Decision 1: the shape of the config key

Option A, a general status-to-directory table:

```yaml
directories:
  draft: draft
  done: done
  archived: archive
```

Anything unlisted lands in `tickets/`. A store with no `directories` key gets the current default, so existing stores and every current fixture are untouched. This makes `archive/` a case of the rule rather than a hardcode, which removes a special case instead of adding one.

Option B, a narrow opt-in list of extra status directories, leaving `archive/` hardcoded. Smaller blast radius, but keeps the special case and reads as two mechanisms doing one job.

Either way, only placement becomes table-driven. Archived-ness stays semantically special, because `Archived()` also drives dependency satisfaction per 6.3, `unarchive`, and the `archive` frontmatter block. Placement and meaning are separable and should be separated.

### Decision 2: the name of the finding

`archive_location_mismatch` becomes a wrong name once placement generalizes past the archive.

Option A, widen the existing code's condition and keep the name. Least disruptive to consumers and to the corpus, but the name misleads, and a misleading name rots.

Option B, rename to `location_mismatch`. Honest, but it is a breaking change to the JSON contract under 12.4 and every sidecar naming the old code has to follow, per `TestCorpusCoversEveryPlanCode`.

Option C, add `location_mismatch` alongside and retire the old code at a schema bump. Two codes for one condition in the meantime, which is its own kind of wrong.

### Explicitly out of scope

Directories for `ready`, `in-progress`, `blocked`, or `review`. Those are the working set, they churn, and a directory for each would turn every status transition into a rename. The plan should say so rather than leaving it as an obvious next step for somebody.

### The baseline this has to beat

Archiving the 27 done tickets costs nothing and needs no format change. `tickets/` would drop to 11.

It does not help the 10 drafts, and it conflates work finished last week with work retired, which `unarchive` restoring to `ready` shows was never the intent of the archive. The distinction looks real, but the comparison belongs on the record so the change is auditable rather than assumed.

### Costs to weigh

A status change that crosses a directory boundary is a rename. Two agents moving one ticket to different statuses collide as a rename conflict rather than a content conflict on `status:`. The collision already exists, this makes it uglier to resolve, and it matters because several agents work this repository at once in separate worktrees.

Every store fixture gains a layout dimension, though only for stores that opt in, so the cost is one or two new fixtures plus their mismatch findings.

`tickets/` comes to mean "the working set" and the name gets less accurate. Not worth a rename, worth a sentence in section 4.

## Acceptance criteria

- [ ] Both decisions in the description are settled and recorded in docs/plan.md before any code lands.
- [ ] The layout is opt-in: a store with no config key behaves exactly as it does today, and every existing fixture is untouched.
- [ ] check --fix migrates a store into the new layout in one pass, and check reports the drift until it does.
- [ ] Directories for ready, in-progress, blocked, and review are refused, with the reason recorded in the plan.
- [ ] A corpus fixture covers a store using the new layout, with a sidecar for its mismatch findings.
