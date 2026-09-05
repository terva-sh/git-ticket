---
schema: 1
id: TKT-01M1QK6BWBDEF97GPTXC11M260
title: Show every TUI control in the footer or a ? help view
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
claim: null
archive: null
created_at: 2026-09-05T01:33:40Z
updated_at: 2026-09-05T01:36:10Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

A control the UI never prints might as well not exist for the user. The
list view binds `n` (new), `e` (edit), and `l` (open) and its footer
names none of them, so both write flows shipped in v0.7.0 are invisible
to anyone who has not read the source.

The footer is also over budget. The idle line runs about 95 characters,
so at 80 columns the Frame clips it and even the keys it does name fall
off the right edge. Adding `n` and `e` to it would make the clipping
worse, not better.

The fix is two halves. The footer shrinks to fit 80 columns and names
the everyday keys plus `? help`. A new `?` help view lists every
binding the list state accepts, including the filter token syntax
(`status:`, `label:`, `assignee:`, and bare title words), which today
is documented nowhere on screen.

A test holds the hints to reality, in the spirit of
TestInstructionsNameRealCommands: every key the App handles in the list
state must appear in the footer or the help view, so a binding added
later without a hint fails the suite instead of shipping invisible.

The form, the status picker, and the detail view already print
complete hints for their own keys and need no change.

## Acceptance criteria

- [x] The list footer fits 80 columns and names n, e, and ? help
- [x] A ? help view lists every list-state binding and the filter token syntax
- [x] A test fails when a handled key appears in neither the footer nor help

## Notes

**agent:terva/mieli** at 2026-09-05T01:36:09Z

Built and all three criteria are ticked.

The idle footer now reads `N open tickets · Enter open · n new · e edit
· / filter · ? help · q quit`, about 76 columns at a two-digit count, so
an 80-column Frame clips nothing. The old line ran about 95 and was
already losing its tail, so `s`, `c`, `u`, `r`, and `q` were invisible
at 80 columns even though the footer named them. Those keys moved to the
`?` page rather than off the screen. The filtered footer gained
`? help`.

`?` opens tui/view/help.go: every list-state binding including the
navigation keys tui.List handles, plus the filter token syntax
(`status:`, `label:`, `assignee:`, bare title words), which no screen
documented before. Esc, q, ?, or Enter closes it back to the list;
Ctrl+C quits, matching the detail view's ladder.

TestEveryListKeyIsHintedSomewhere is the durable half: an inventory
mirroring the switches in App.HandleKey, ListView.HandleKey, and
tui.List, each entry required to appear in the footer or on the help
page. The single-letter entries match the start of a help table row or
a footer phrase, so a letter inside prose does not satisfy them. The
form, picker, and detail footers were already complete and are
untouched.

## Summary

Every control the list state accepts is now printed somewhere the user
can see it. The footer fits 80 columns and names Enter, n, e, /, ?, and
q; the new `?` help page carries the full key table and the filter token
syntax. TestEveryListKeyIsHintedSomewhere holds the union complete, so a
binding added without a hint fails the suite instead of shipping
invisible. The form, status picker, and detail view already printed
their own keys and did not change.
