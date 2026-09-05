---
schema: 1
id: TKT-01M1HPCJH5RWSA6Z28QM3QV1H0
title: Decide whether the format stores a hand-set order
type: spike
status: draft
status_reason: null
priority: low
due_on: null
labels:
  - question
  - format
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references:
  - ref: review:backlog-md
    path: docs/review-backlog-md.md
claim: null
archive: null
created_at: 2026-09-02T18:34:03Z
updated_at: 2026-09-05T11:38:32Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

A deferred question raised by reading Backlog.md. See section B6 of docs/review-backlog-md.md.

Store.List sorts by ID and nothing else, which plan 5.5 makes chronological because a ULID sorts by creation time. priority is a filter and never an order. So the only sequence the format can express is the order tickets were filed in.

Two different wants hide in that. Sorting a list by priority is display: every consumer already has priority on every row, and if human output ever wants it that is a --sort flag with no format change behind it. A hand-set sequence is not display, because no UI can persist a field the format does not have. Backlog.md stores an ordinal per task and maintains it in src/core/reorder.ts, which is what makes its drag-and-drop stick.

The trigger: terva ships a board with reorderable columns, or somebody asks for an ordering within one priority level and a fifth priority is not the answer.

The cost, if the trigger is met, is a frontmatter field on every ticket under plan 5.3, which is every fixture. Worse, an ordinal is the first field in the format whose value is meaningless on its own and only means something relative to its neighbours, so two agents inserting concurrently produce a merge Git cannot resolve sensibly. That interacts with the merge driver question, and whoever settles this should read that one first.

## Notes

**agent:terva/mieli** at 2026-09-03T06:01:18Z

Groomed. Two of the facts in this ticket changed today.

The display half is settled and shipped. `list --sort priority` and `ready` ranking by priority landed in TKT-01M1J2YR, exactly as this ticket predicted: a flag, no format change behind it. So the question here is now only the hand-set ordinal, which is the half no flag can reach.

The merge-driver question this defers to is answered. Plan 7.5 is the design and TKT-01M1JPW1 built it, so read 7.5 rather than the spike. That changes the cost paragraph above. An ordinal is still the first field whose value means nothing on its own, but concurrent inserts now have somewhere to be resolved: a 7.5 field rule, per field, rather than a line merge. Whoever settles this owes 7.5 a row for the ordinal, and the honest answer for that row may well be conflict, because two agents inserting at the same position genuinely disagree.

The trigger has not fired. No board, and nobody has asked for an order inside one priority level.

**agent:terva/mieli** at 2026-09-05T11:38:32Z

Groomed. The trigger has not fired, but it is the closest to firing of
any draft in the store. The trigger names a board with reorderable
columns, and the TUI shipped in v0.7.0 as `git ticket ui`, a filterable
list over the open working set, not a board. A board view is the
natural next TUI slice, and building one either fires this or forces
the fifth-priority dodge the trigger rules out.

The order of operations from the last note stands and is now the
actionable part: whoever builds the board owes plan 7.5 a row for the
ordinal first, and the honest answer for that row may well be conflict,
because two agents inserting at the same position genuinely disagree.
Read this ticket before starting a board, not after.
