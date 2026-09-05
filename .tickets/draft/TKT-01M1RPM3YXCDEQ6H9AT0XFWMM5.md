---
schema: 1
id: TKT-01M1RPM3YXCDEQ6H9AT0XFWMM5
title: Copy a ticket's body from the TUI detail view
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels: []
assignees: []
milestone: null
parent: null
dependencies:
  - TKT-01M1RPM03QD2D41MX7Z00WQZ36
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-05T11:52:51Z
updated_at: 2026-09-05T11:52:51Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Filed 2026-09-05 from the same user ask as its dependency: copy the
open ticket's body out of the TUI without leaving it. Terminal
selection inside an alt-screen view is worse than in a scrollback: the
viewport wraps long lines, the selection picks up the frame, and a
body taller than the pane cannot be selected at all.

The shape: `y` in the detail view copies the ticket's Markdown body,
as stored below the frontmatter, to the system clipboard, and the
footer flashes a confirmation naming the ticket. `y` is unbound in
DetailView.HandleKey today, and yank is what a vim hand expects it to
mean.

The mechanism starts where the TUI already is. This TUI owns the
terminal, so OSC 52 written to it is the primary path: no external
binary, and it survives SSH, which the exec-a-tool path never does.
Windows Terminal honors OSC 52, which covers WSL2 without touching
clip.exe. When the terminal does not honor it, the fallback is the
probing helper the CLI ticket builds, which is what the dependency on
that ticket is for. If OSC 52 alone turns out to be enough, the
dependency can be unlinked and this ships independently.

Two conventions bind the implementation. Every list-state binding must
appear in the footer or the `?` help page, which
TestEveryListKeyIsHintedSomewhere enforces; the detail view carries
its own footer and help entries, so `y` goes into both. And the TUI
tests drive a vt10x terminal through renderApp, so the OSC 52 write
needs a seam the test can observe, the same way the resize and editor
paths have one.

Platform scope matches the CLI ticket: WSL2, Linux, macOS. Native
Windows is unsupported; the binding degrades to the footer message
naming the failure rather than a silent no-op, because a copy that
silently did nothing is the worst outcome a paste can discover.

### Trigger

None. Startable once its dependency lands; parked as a draft until the
user promotes it.

## Acceptance criteria

- [ ] y in the detail view copies the stored body to the clipboard and the footer confirms, naming the ticket
- [ ] y is named in the detail footer or the ? help page, per the every-binding-is-hinted convention
- [ ] OSC 52 is the primary path and the probing helper the fallback, with the write observable in a vt10x test
- [ ] an unsupported platform gets a footer failure message, never a silent no-op
