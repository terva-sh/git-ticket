---
schema: 1
id: TKT-01M1S022Q9SM9DGJR1AJHTN0BC
title: Color the TUI list rows, palette decided live with the user
type: task
status: blocked
status_reason: awaiting the decide-and-test session with the user at their terminal
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
updated_at: 2026-09-05T15:31:49Z
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

## Notes

**agent:terva/mieli** at 2026-09-05T15:31:49Z

The session scaffolding is built and the decide-and-test session is
now the only thing between this ticket and its palette. Everything
here is per the plan section of the description: switchable
candidates, no decision made.

To run the session, launch the TUI once per candidate against this
store and look at the same working set each time:

    GIT_TICKET_UI_PALETTE=status   git ticket ui
    GIT_TICKET_UI_PALETTE=priority git ticket ui
    GIT_TICKET_UI_PALETTE=type     git ticket ui
    GIT_TICKET_UI_PALETTE=dim      git ticket ui
    git ticket ui                  # the off baseline

What each candidate says. status: in-progress cyan, review magenta,
blocked yellow, draft and done dim, ready plain. priority: urgent bold
yellow, high blue, low dim, normal plain. type: epic magenta, bug
yellow, spike cyan, chore dim, task plain. dim is the control
candidate, weight and no hue at all: in-progress bold, draft and done
dim. Every candidate colors the exceptional and leaves the ordinary
alone, because a palette that colors everything makes color mean
nothing, and TestEveryPaletteLeavesTheOrdinaryAlone holds that.

The constraints are enforced already, not promised: the selected row
takes no palette color, so the cursor stays legible over every
candidate; NO_COLOR wins over any selection, presence alone, per
no-color.org; an unknown name is off rather than an error; and the
hues in use are cyan, magenta, yellow, and blue plus weight, so no
meaning rides on red-green alone. Six tests in palette_test.go pin all
of it.

After the session: the winner's mapping moves into Theme fields in
tui/theme.go, the losers and the reasoning land here as the fourth
criterion asks, and palette.go plus the environment variable are
deleted. The file's own header says the same, so a reader finding it
without this ticket still knows it is scaffolding.

Parked blocked until the session happens, and the claim is released,
because a claim naming a merged branch tells the other agents a
falsehood. Anyone picking this up starts at the command list above,
with the user at their own terminal.

**agent:terva/mieli** at 2026-09-05T15:31:49Z

in-progress to blocked: awaiting the decide-and-test session with the user at their terminal
