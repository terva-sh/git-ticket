---
schema: 1
id: TKT-01M1RPM3YXCDEQ6H9AT0XFWMM5
title: Copy a ticket's body from the TUI detail view
type: task
status: done
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
claim:
  actor: agent:terva/mieli
  branch: feat/tui-yank
  worktree: /home/sothr/workspace/git.local.sothr.com/terva-sh/git-ticket
  commit: 4198358be19fb28758f65aac71135d1d83f3ca37
  claimed_at: 2026-09-05T14:31:32Z
  expires_at: null
archive: null
created_at: 2026-09-05T11:52:51Z
updated_at: 2026-09-05T14:35:41Z
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

- [x] y in the detail view copies the stored body to the clipboard and the footer confirms, naming the ticket
- [x] y is named in the detail footer or the ? help page, per the every-binding-is-hinted convention
- [ ] OSC 52 is the primary path and the probing helper the fallback, with the write observable in a vt10x test
- [x] an unsupported platform gets a footer failure message, never a silent no-op

## Notes

**agent:terva/mieli** at 2026-09-05T14:35:41Z

Built. Three of the four criteria are ticked, and the third is left
unticked deliberately, because the work deviated from it and a
criterion is not reworded to match what shipped.

The third criterion asked for OSC 52 as the primary path with the
probing helper as the fallback. The build inverted that order, for a
reason the draft did not see: an OSC 52 write cannot report failure,
so a footer confirming "copied via OSC 52" while a working wl-copy sat
on PATH would lie exactly when the terminal ignored the sequence. The
tools can report success and failure; OSC 52 cannot; so the honest
order is tools first, OSC 52 last, which is also the order `copy`
already ships. Plan 12.7's new TUI paragraph records the reasoning.
The criterion also asked for the write to be observable in a vt10x
test, and the observation instead happens at the Actions.Copy and
StoreParams.Clipboard seams, driven through the App, because the write
itself is cli's writeClipboard and the cli suite already holds it.

The mechanics: y is handled in the App, which holds the actions the
detail view deliberately does not. Actions gains Copy, StoreParams and
UIParams gain a Clipboard field, last in both structs because the
composition-root cast needs identical field order, and cli/ui.go binds
it to writeClipboard, so the y binding and the copy command cannot
disagree about how a clipboard is reached. storeCopy reads the file
and feeds RawBody bytes, which TestStoreCopyFeedsTheStoredBody holds
against a real store, byte for byte. The footer flash clears on the
next key, and the detail footer hint dropped its arrow glyphs to fit
"y copy" in a 60-column pane, which a width test caught immediately.

An unwired host gets "copying is not wired in this host" and a failed
copy flashes the error, so the fourth criterion holds on both paths.

## Summary

Shipped. y in the TUI detail view puts the ticket's stored body on the
system clipboard through the same probing helper as `git ticket copy`,
wired in as a Clipboard field on StoreParams and UIParams, so the two
surfaces cannot disagree about how a clipboard is reached. The footer
flashes the ticket, the byte count, and the path taken, clears on the
next key, and says so loudly when the copy fails or the host wired no
clipboard. The third criterion is unticked on purpose: the build
inverted the OSC 52-first order it asked for, because OSC 52 cannot
report failure and a confirming footer must not lie. Plan 12.7 records
the reasoning.
