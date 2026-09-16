---
schema: 2
id: TKT-01M2NZE9Z4G033QJDFG62N4Y9D
title: label_order's min cannot reach a store with a written dimension order
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - area/doctor
  - question
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references:
  - ref: origin-ticket:TKT-01M2NZ88
    path: .tickets/draft/
claim: null
archive: null
created_at: 2026-09-16T20:44:56Z
updated_at: 2026-09-16T21:55:43Z
created_by:
  id: human:sothr
  name: ""
updated_by:
  id: agent:claude/t3code
  name: ""
extensions: {}
---

## Description

`labelOrderMinimum`'s comment invites this report, so here are the numbers it
asked for:

> Measured when this default moved from 3 to 2: terva, which documents `area/`
> before `scope/` and carries two labels on nearly every ticket, went from 0
> soft findings to 48. Raising `min` is the narrow answer there; disabling the
> rule is the blunt one, and a store should not have to reach for the blunt one.

terva reached for the blunt one. `enabled: false`, in this change:
terva-sh/terva, `.tickets/config.yml`. Not because the narrow answer was
unpleasant, but because no value of `min` expresses what terva needs.

## Notes

**agent:claude/t3code** at 2026-09-16T21:10:30Z

v0.19.1 answers the noise half of this and leaves the `order:` proposal open.

The rule no longer thresholds on label count. It reads the dimension each ticket
leads with, finds the one this store leads with most often, and reports only the
tickets that disagree. Measured on the store this ticket describes, read from
`origin/sothr-main`: **120 findings become 4**, and the 4 are exactly the
`live-test` and `flake` tickets found by hand here. That is the part worth
noting, because it says the inferred convention and the written one agree.

Confirming the numbers independently: 121 open, 120 multi-label, `area/` leading
116 of them. This ticket counted 127 open and 120 multi-label against a slightly
different snapshot; the multi-label figure, which is the one the rule keys on,
matches exactly.

**What this does not do, and the ticket stays open for it.** There is no
`order:` parameter. The rule infers one dominant leading dimension rather than
reading a total order, so it cannot express `area/` before `init/` before
`scope/`, and it says nothing about the sequence after the first label. The
argument for inferring was that it needs no configuration and is therefore
already true of every store, including ones with no written convention; the
argument for declaring is that a store that has written the order down should
not have it guessed at, and that a total order catches a mis-sorted third label
that a leading-dimension rule cannot see. Those are not in conflict and the
second is still unbuilt.

Two observations from this ticket that the new rule does not reach:

- The 34 tickets carrying two `area/` labels, where the alphabet chose which
  leads. Both lead with `area/`, so the new rule is silent on all 34. This
  ticket already says that is a separate question, and it is filed separately.
- `visible` is still meaningless for a store with no board. It now only changes
  the wording of findings that are rare rather than universal, so the cost is
  smaller, but the parameter still cannot be set to anything useful here.

The `enabled: false` in terva's config and the by-hand fix to the five tickets
are both still unpushed as this note is written; `origin/sothr-main` is at
1977238d. Re-enabling after upgrading to v0.19.1 is worth trying, since the
thing it was switched off for is the thing that changed.

**agent:claude/t3code** at 2026-09-16T21:55:43Z

The `order:` proposal here now has a companion, TKT-01M2P3FAHHBGXEMRRYE9R5AW5B.

A second idea arrived independently: weight each label by one over the number of
labels sharing its dimension, so two `area/` labels are each half-identifying and
a lone `scope/` is whole. Measured, it finds the 34 two-area tickets this ticket
named, which is the set a total order cannot catch, because `area, area, scope`
does follow `area -> scope`.

It also cannot stand alone. On the four-label shape here the two areas weigh 0.5
and both `init/` and `scope/` weigh 1.0, so it ties and says nothing, which is
the 19 tickets where the initiative is the intended answer. And on ketju, whose
only non-area dimension is `effort:`, it says `effort:M` should lead over
`area:db` on 27 tickets, which is worse advice than silence.

So the two proposals compose: dilution says which dimension stopped identifying
the ticket, and the order declared here says what is eligible to take over. That
is the shape being taken forward.

## What the store actually looks like

127 open tickets, label counts: 7 with one, 48 with two, 51 with three, 21 with
four. At `min: 2` the rule reports 120 findings, which is every multi-label open
ticket. The 48 in the comment above has become 120 as the taxonomy filled in.

The dimension sequences present, counted:

```
  39  area -> scope
  36  area -> init -> scope
  19  area -> area -> init -> scope
  10  area -> area -> scope
   9  area -> init
   6  <bare>
   3  area -> area -> init
   2  <bare> -> area -> scope
   2  <bare> -> area -> area -> scope
   1  init
```

Labels sort alphabetically, and terva's prefixes were chosen so that the sort
*is* the dimension order: `area/`, `init/`, `scope/`, conditions last. That is
written down in terva's CONVENTIONS.md. So "is the leading label the one that
describes it best?" has a store-wide answer, given once, and 120 findings
restate the question the convention already closed.

## Why min cannot isolate the residue

The rule is not wrong that something here is unsettled. 34 of those open tickets
carry **two `area/` labels**, and which of the two leads was decided by the
alphabet, not by anyone:

- `area/permissions`, `area/tui` on "Phase 3: the launch trust dialog in the TUI"
- `area/ci`, `area/docs` on "A gate for comments that claim a symbol does not exist"
- `area/ext`, `area/permissions` on "Declare the web tools network-read and coordinate host egress"

In each the trailing label arguably names what the ticket is about. That is a
real finding and terva filed it as its own ticket rather than losing it.

But those 34 have two, three, and four labels — they are spread across every
bucket. `min` is a threshold on label count, so no setting of it separates the
34 that are a live question from the 120 that are not. `min: 3` leaves 72;
`min: 4` leaves 21 and drops 13 of the 34 it should have kept.

## And visible does not apply at all

`visible` is documented against what git-ticket-canvas renders. terva has no
board. Tickets are read through `git ticket list` and through terva's own
`ticket_get` tool, both of which print every label, so nothing is ever behind a
fold. The parameter cannot be set to a value that makes it meaningful here; it
only changes the wording of findings terva does not want.

## The shape that would work

The thing terva can state, and the rule cannot currently hear, is a **total
order on label prefixes**. Something like:

```yaml
doctor:
  rules:
    label_order:
      params:
        order: [area/, init/, scope/]
```

Read as: a ticket whose labels do not follow this prefix order is a finding;
a ticket that does follow it is silent, whatever its label count. That inverts
the rule from a question into a check, which is a real change in kind — but it
is the version a store with a written convention can actually use, and it would
have caught the five terva tickets where a bare `live-test`/`flake` label sorted
ahead of `area/` (found by hand while investigating this, now fixed).

It also leaves the genuine question intact: ordering *within* a prefix is not
something a prefix order can settle, so those 34 stay a human decision.

Filed as a question rather than a bug. The current rule is defensible and the
comment's reasoning is sound; this is the measurement it asked for, plus the
report that the narrow answer did not reach.
