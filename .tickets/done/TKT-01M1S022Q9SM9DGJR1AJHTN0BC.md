---
schema: 1
id: TKT-01M1S022Q9SM9DGJR1AJHTN0BC
title: Color the TUI list rows, palette decided live with the user
type: task
status: done
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
claim:
  actor: agent:terva/mieli
  branch: feat/tui-palette-switch
  worktree: /home/sothr/workspace/git.local.sothr.com/terva-sh/git-ticket
  commit: 2177c50d96138a6e3cf3433cef9b712415009f08
  claimed_at: 2026-09-05T15:55:34Z
  expires_at: null
archive: null
created_at: 2026-09-05T14:37:46Z
updated_at: 2026-09-05T15:57:56Z
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

- [x] a decide-and-test session with the user picks the driving field and the palette from switchable candidates in a real terminal
- [x] the selected row stays legible, NO_COLOR keeps a readable baseline, and no meaning rides on red-green alone
- [x] the decision and the rejected candidates are recorded on this ticket
- [x] the palettes live in one place, tui/view/palette.go, with priority the shipped default and p cycling the set at runtime, not escape codes in list.go

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

**agent:terva/mieli** at 2026-09-05T15:57:44Z

The decide-and-test session happened on 2026-09-05, and this note is
the record the fourth criterion asks for.

The user ran all five candidates against terva's own 66-ticket store,
a heavier working set than this repository's, and captured five
screenshots. What the terminal showed: status was dim-plus-accents on
a draft-heavy store, with the two review rows landing magenta;
priority made the high rows glow blue across the ready block with low
receding; type made epics, bugs, and spikes findable by hue but said
nothing about state; dim was the calm no-hue control; off was the flat
baseline. The cursor stayed legible in every shot.

The decision: priority wins as the default, because "what matters" is
the question the user asks the list most. The rejected-as-default
candidates are status, type, dim, and off, for the reasons above, but
the session also changed the shape of the ship: rather than deleting
the losers, the user asked for runtime switching, so color composes
with the filter and the sort the way a person actually mixes them.
All five palettes ship, p cycles them, the header names the active
one, and GIT_TICKET_UI_PALETTE selects the start. NO_COLOR pins the
colors off and locks the key, because an environment that asked for
no color should not be one keystroke from getting some.

That decision changed what the second criterion asks, so it is
reworded rather than shipped false. The original, verbatim: "the
chosen palette ships as Theme fields in tui/theme.go, not escape
codes in list.go". Its intent was one home for the color decisions
rather than escape codes scattered through list.go, and that intent
survives: the palettes live in tui/view/palette.go, one file, chosen
over tui/theme.go because a palette reads ticket fields and the tui
package deliberately does not import ticket. The reworded criterion
says what the session actually asked for. This note supersedes the
2026-09-05 scaffolding note's after-the-session paragraph, which
promised the winner would move into Theme fields and the file would
be deleted: the session decided otherwise, and the file stayed as the
feature.

## Summary

Shipped, decided live. The TUI list colors its rows: priority is the
default the 2026-09-05 session picked against terva's 66-ticket store,
with high blue, urgent bold yellow, low dim, and normal plain. The
session also reshaped the ship: rather than one winner as Theme
fields, all five palettes stayed as a feature, p cycles them at
runtime, the header names the active one, and color composes with the
filter and the sort. GIT_TICKET_UI_PALETTE selects the start, NO_COLOR
pins off and locks the key, the cursor row never takes palette color,
and no meaning rides on red-green alone. The decision, the rejected
candidates, and the reworded second criterion are all in the notes.
