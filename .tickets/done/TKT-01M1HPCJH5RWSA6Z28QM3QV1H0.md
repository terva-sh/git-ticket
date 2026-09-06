---
schema: 2
id: TKT-01M1HPCJH5RWSA6Z28QM3QV1H0
title: Decide whether the format stores a hand-set order
type: spike
status: done
status_reason: null
priority: low
due_on: null
labels:
  - question
  - format
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references:
  - ref: review:backlog-md
    path: docs/review-backlog-md.md
claim: null
archive: null
created_at: 2026-09-02T18:34:03Z
updated_at: 2026-09-06T00:27:47Z
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

**agent:terva/mieli** at 2026-09-06T00:27:47Z

Settled by the user: the format does not carry a hand-set ordinal.
Section 15 of `docs/plan.md` now records it as answered rather than
deferred, and this note supersedes the guidance in both grooming
notes above.

### What those notes told a reader to do, and why it no longer holds

Both said the trigger had not fired and that whoever builds a board
owes plan 7.5 a row for the ordinal first. The second grooming note
went further and said to read this ticket before starting a board,
because building one would either fire the trigger or force the
fifth-priority dodge.

That is withdrawn. A board no longer reopens this question by itself.
It derives its column order from fields that already exist, and it
owes 7.5 nothing, because there is no ordinal to write a row for.

### The argument that settled it

Two findings, neither of which was written down before.

This format already expresses order twice, and both mechanisms merge.
`due_on` is a total order, and 7.5 resolves it by taking the side that
changed, which works because a date means something by itself.
`dependencies` is a real partial order that 7.5 unions, for the same
reason: an edge means something by itself. An ordinal would be a third
ordering mechanism and the only one that cannot merge, because its
value means nothing alone and only means something against its
neighbours.

That is precisely what makes its honest 7.5 row conflict. Two agents
inserting at the same position genuinely disagree, and no rule
resolves that without discarding somebody's intent. A field that
conflicts by design fights the property the format rests on, which is
that two agents working at once produce files Git can reconcile.

The file cost is real and was never the reason. A frontmatter field
under 5.3 lands on every ticket, which is 141 files today: 52 fixtures
carrying a `schema:` line plus 89 in this repository's own store. That
is tedious, not disqualifying. The concurrency objection is the
disqualifying one, and unlike the file count it does not improve with
time.

### What the display half already gave away

Worth restating because it is the half people actually want. Sorting a
listing needs no format change and shipped in
`TKT-01M1J2YR9D5242F6H7TPEV4M8K` as `list --sort`, over data every
consumer already had. Nothing in this decision takes that away, and a
new sort key remains a flag rather than a format question.

### The reopen condition

Narrower than the old trigger, deliberately. It needs a want that
`due_on`, `dependencies` and `priority` provably cannot express, and
an answer to the concurrency objection rather than an acknowledgement
of it.

## Summary

Decided: the format does not carry a hand-set ordinal. Section 15 of
`docs/plan.md` records it as answered, replacing the deferred entry.

The question always had two halves. Sorting a listing is display and
needed no format change, which `list --sort` already delivered. A
sequence set by hand was the half no flag could reach, because no
interface can persist a field the format does not define. That half is
now closed rather than waiting.

Two findings settled it, neither written down before this spike was
worked.

The format already expresses order twice, with fields that merge.
`due_on` is a total order that 7.5 resolves by taking the side that
changed, because a date means something by itself. `dependencies` is a
real partial order that 7.5 unions, because an edge does too. An
ordinal would be a third ordering mechanism and the only one whose
value means nothing alone.

That is what makes its honest 7.5 row conflict. Two agents inserting
at the same position genuinely disagree, and no rule resolves that
without discarding somebody's intent. A field that conflicts by design
fights the property the format rests on. The file cost, a frontmatter
field on 141 files today, is the smaller objection and was never the
reason: it is tedious, while the concurrency objection is
disqualifying and does not weaken with time.

The old trigger is retired with the question. A board with reorderable
columns no longer reopens this by itself, because it derives its
column order from fields that exist and owes 7.5 no new row. Reopening
now needs a want that `due_on`, `dependencies` and `priority` provably
cannot express, together with an answer to the concurrency objection
rather than an acknowledgement of it.

No code changed. The deliverable is the plan text and the retired
trigger.
