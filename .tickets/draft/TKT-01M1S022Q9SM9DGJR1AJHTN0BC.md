---
schema: 1
id: TKT-01M1S022Q9SM9DGJR1AJHTN0BC
title: Color the TUI list rows, palette decided live with the user
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-05T14:37:46Z
updated_at: 2026-09-05T14:37:46Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Filed 2026-09-05 from a user ask: the TUI list renders every row in
the same weight, so telling a blocked urgent bug from a low draft
chore means reading five columns of text. Color would let the eye do
it.

This ticket deliberately decides no colors. The user asked for that
restraint explicitly: the palette, and even which field drives the
color, are to be decided in a live session with the user, trying
candidates in a real terminal, because color choices made in prose
look right and read wrong. The work here is to prepare that session
and then implement its outcome.

What the ticket does record is where the mechanism lives and what the
session must weigh. The home is tui/theme.go: Theme currently carries
Accent and Muted, and row coloring extends Theme rather than scatters
escape codes through list.go, so a palette stays one place and a
future light-terminal or high-contrast variant swaps cleanly. The
candidates to try are at least: status-driven color (blocked red,
done dim), priority-driven (urgent stands out, low recedes),
type-driven (epics differ from tasks), and dim-only variants that use
weight instead of hue. Each candidate gets tried against this store's
own working set, which mixes all of them.

Constraints the session should hold whatever it picks: the selected
row's reverse-video must stay legible over any row color; the Frame
clips styled lines, so WrapANSILineKeepStyle territory applies if
color ever reaches wrapped text; NO_COLOR and dumb terminals need the
uncolored fallback to remain the readable baseline, not an
afterthought; and a palette that collapses for the common forms of
color blindness fails the point, so red-green alone cannot carry
meaning that nothing else carries.

### Plan

Prepare a small set of switchable candidate palettes behind a flag or
environment variable, sit down with the user in their terminal, try
them against the real store, pick one, then ship it as Theme fields
with the decision recorded here.

### Trigger

None. Startable once the user schedules the decide-and-test session,
which is the first step of the work rather than a precondition for it.

## Acceptance criteria

- [ ] a decide-and-test session with the user picks the driving field and the palette from switchable candidates in a real terminal
- [ ] the chosen palette ships as Theme fields in tui/theme.go, not escape codes in list.go
- [ ] the selected row stays legible, NO_COLOR keeps a readable baseline, and no meaning rides on red-green alone
- [ ] the decision and the rejected candidates are recorded on this ticket
