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
updated_at: 2026-09-16T20:45:44Z
created_by:
  id: human:sothr
  name: ""
updated_by:
  id: human:sothr
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
