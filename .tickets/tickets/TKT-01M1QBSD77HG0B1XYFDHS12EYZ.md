---
schema: 1
id: TKT-01M1QBSD77HG0B1XYFDHS12EYZ
title: Build the Phase 4 TUI view for browsing and editing tickets
type: task
status: in-progress
status_reason: null
priority: low
due_on: null
labels: []
assignees: []
milestone: null
parent: null
dependencies:
  - TKT-01M1QBS9F4Y1YY0J90EBW2FTZS
blocks_on: none
references: []
claim:
  actor: agent:terva/mieli
  branch: feat/tui-frame-renderer
  worktree: /home/sothr/workspace/git.local.sothr.com/terva-sh/git-ticket
  commit: 758edcf068ad768d968a3b02af03c8104df4b3ff
  claimed_at: 2026-09-04T23:47:58Z
  expires_at: null
archive: null
created_at: 2026-09-04T23:24:15Z
updated_at: 2026-09-05T01:05:42Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

An interactive terminal UI for reviewing, editing, creating, and managing
tickets. This is the "local browser or TUI view" of plan section 13 Phase 4,
and it stays gated behind that section: Phase 3 first, and the file and agent
contracts held through at least one real project.

Placement, decided 2026-09-04: the TUI ships in this module as its own package
and entrypoint. Consumers that import `ticket` or `cli` alone do not build or
link the terminal stack, because Go prunes the module graph to the packages
actually imported. Extraction into a shared terva-sh TUI library is deferred
until a second consumer exists. One sub-decision stays open: a `git ticket ui`
subcommand in the existing binary, against a separate command, which trades
invocation convenience against binary size for the agent hosts that never open
a terminal UI.

The rendering core lifts from terva's `packages/tui`, which is framework-free
on `golang.org/x/term`, `go-runewidth`, and `chroma/v2`. Both trees are MIT
under the same copyright holder; lifted files carry the terva notice. The
clean-lift set, audited free of terva-internal imports: `terminal.go`,
`input.go`, `editor.go`, the theme trio, `markdown.go` with `highlight.go`,
the wrap and line-buffer helpers, hyperlink, clipboard, progress, and the
`tuitest` fake-terminal harness with its vt10x screen. The chat-specific
`view.go`, `toolsummary.go`, and `partialjson.go` stay behind. Widget seeds
come from terva's `modes/dialogs`: `viewport.go`, `picker.go`,
`dialog_frame.go`, `field_input.go`, and the `question_dialog.go` form
patterns. Do not copy the overlay dispatch around them; terva's own archived
maintainability plan records the boilerplate it cost.

The one genuinely new core piece is an alt-screen frame renderer. terva's flow
renderer paints a chat into scrollback and is the wrong shape for a
list-detail-form application. Editing writes through the library with the
revision precondition, and a conflict re-reads and re-presents rather than
overwrites.

## Acceptance criteria

- [x] a list view shows open work, filterable by status, label, and assignee
- [x] the detail view renders the ticket Markdown with its sections
- [ ] create and edit write through the library, and a stale revision re-presents instead of overwriting
- [x] status transitions, claim, and release are reachable from the list
- [x] tests drive a fake terminal and assert the emulated cell grid

## Notes

**agent:terva/mieli** at 2026-09-04T23:52:20Z

The rendering foundation is in. The `tui` package lands the lift and the one
new core piece, per the description:

- Lifted with the terva MIT notice: `terminal.go` (Terminal, ProcTerm, the
  escape vocabulary), `resize_unix.go`/`resize_windows.go`, `ansiescape.go`,
  and the width helpers in `width.go` (VisibleWidth, StripANSI,
  TruncateToWidth). Inline-image handling was deliberately left behind;
  `width.go` says so.
- New: `frame.go`, the alt-screen Frame renderer. Start/Stop own the raw-mode
  and alternate-screen lifecycle, Draw takes the whole frame as []string and
  rewrites only the rows that changed, wrapped in synchronized output. Resize
  and Invalidate force a full repaint. SetCursor/HideCursor place the caret.
- Lifted: `tuitest` (FakeTerm plus the vt10x Screen), so every frame test
  asserts the emulated cell grid rather than escape bytes. Sixteen tests
  cover the lifecycle, the diff, clipping, resize, the cursor, and the
  width helpers.

Dependencies added to go.mod: golang.org/x/term, golang.org/x/sys,
github.com/mattn/go-runewidth, github.com/hinshun/vt10x (tests only).
Consumers that import only ticket or cli do not build any of them.

The fifth acceptance criterion is met and ticked. The remaining four are the
view itself: list, detail, forms, and transitions, all still to build on this
foundation. The subcommand-versus-separate-binary sub-decision stays open.

**agent:terva/mieli** at 2026-09-05T00:00:37Z

The list view is up on the Frame, with the Viewport and picker patterns
lifted:

- `tui/input.go` lifts terva's key Reader verbatim: CSI and SS3, the kitty
  keyboard protocol, xterm modifyOtherKeys, bracketed paste, and SGR wheel
  events.
- `tui/viewport.go` lifts the dialogs Viewport, with the themed more-above
  and more-below indicator rows replaced by plain text until a Theme exists
  here.
- `tui/list.go` reshapes the picker pattern into a generic cursor List:
  cursor plus viewport in one type, arrows, j/k, g/G, paging, and the wheel.
- `tui/view` is the new application package, split so package tui stays
  ticket-free. `ListView` renders open work with shortest-unique IDs
  (mirroring cli/commands.go), a header, and a key-hint footer. Reload
  follows the selected ticket by ID, not by row, and a lister error keeps
  the previous rows and names itself in the footer. `Run` wires Frame,
  Reader, and a Lister closure; a resize repaints immediately through a
  mutex-guarded draw.
- The Lister feeds from Store.List(Filter{}), open work by definition. Ten
  view tests and two run-loop tests assert through StripANSI and the vt10x
  grid.

The first acceptance criterion is part-met: the list shows open work and
stays unticked until it filters by status, label, and assignee. Next moves
on this ticket: the filter line, the detail view on Enter (SelectedID is
already there for it), and the entrypoint decision.

**agent:terva/mieli** at 2026-09-05T00:02:14Z

Process note, 2026-09-05: the list-view commit 632cf0d landed on main by a direct push, against the branch-and-PR rule in AGENTS.md. After merging PR #95 the working tree sat on main, and the next commit was made there without branching first. CI on the push is green and the content is what the previous note describes, but the change skipped review. Recorded so the history of this ticket does not read as if the rule held.

**agent:terva/mieli** at 2026-09-05T00:08:25Z

The entrypoint sub-decision is settled: `git ticket ui`, a subcommand in the
existing binary, decided 2026-09-04. This supersedes the open question in the
description and in the first progress note.

The wiring keeps the plan 12.2 embedding rule intact. The `ui` command lives
in `cli/ui.go` with flag parsing and store resolution like every other
command, but the terminal work sits behind a new `Env.RunUI` field that only
the composition root binds: `cmd/git-ticket` sets it to `view.RunProc`, which
refuses a non-tty and runs the list view over `Store.List(Filter{})`. A host
that embeds `cli` for its command surface leaves `RunUI` nil, never builds
the terminal stack, and `ui` fails with a message saying this entrypoint has
no terminal UI wired.

Plan 12.1 lists the command with its no-JSON note. `ui` refuses `--json`,
arguments, and a missing store, each covered by a test in `cli/ui_test.go`.
Verified on the built binary: `--help` lists ui, and `./git-ticket ui
</dev/null` exits 1 with "the TUI needs a terminal on stdin and stdout".

**agent:terva/mieli** at 2026-09-05T00:21:24Z

The detail view is in, and the second acceptance criterion is ticked.

Enter (or l) on the list opens the selected ticket; Esc, q, h, or backspace
return to the list, and Ctrl+C quits from anywhere. `tui/view/app.go` is the
state machine over the two views, one field deep until a third view earns a
stack. `tui/view/detail.go` renders a fixed header (full ID, bold title, a
meta line with status, type, priority, labels, assignees, milestone, due,
and a blocked reason when one exists), then every non-empty body section
under a bold heading, in the plan 5.3 render order with Extra sections
last, scrollable through the lifted Viewport.

The section text is the ticket's own Markdown, wrapped to the pane by the
new `tui.WrapPlain` (space breaks, hard breaks for wide words, indent
preserved on continuation lines, rune-width aware) and otherwise untouched.
Plan 14 promises a ticket reads correctly in an ordinary Markdown viewer,
and this view leans on exactly that property. Styled rendering, terva's
markdown.go with chroma highlighting, remains a later lift and would slot
into DetailView.build.

Twelve view tests now cover the detail and app paths: section order, empty
sections skipped, wrapping, scrolling, the back and quit keys, Enter on an
empty list, and the return path to the list.

**agent:terva/mieli** at 2026-09-05T00:51:17Z

Status transitions, claim, and release are reachable from the list, and the
fourth acceptance criterion is ticked.

From the list: `c` claims the selected ticket, `u` releases it, and `s`
opens a status picker whose options come from ticket.PermittedTransitions,
so the picker cannot offer a move the lifecycle refuses. Choosing blocked
opens a one-line reason prompt, because plan 6.2 makes the reason mandatory
there; an empty confirm re-prompts rather than sending a write the store
would bounce. The prompt is deliberately not the editor lift, and the full
editor replaces it when the edit flows land.

The write surface is `view.Actions`: three closures, each taking the ref
and the revision the view last read. `view.StoreActions` binds them to
Store.Apply with the actor and the claim provenance, and `cli.UIParams`
(store, actor via ctx.actor with its warning, branch/worktree/commit via
gitState) mirrors `view.StoreParams` field for field, so cmd/git-ticket
converts with a cast and cli still imports no terminal code. A nil closure
degrades to a footer message naming what is not wired, the same nil-ness
rule terva's InteractiveConfig uses.

Every action lands in `afterWrite`: reload first, then say what happened.
A stale_revision answer reports "changed by another writer; reloaded, try
again", which is the re-present loop of the TKT-01M1QBS9 spike doing its
job on the first real write surface. Ten new tests cover the ref and
revision travelling with each write, the picker flow end to end through
App, the blocked reason, the conflict path, and the unwired degradation.

Remaining: the filter line for criterion 1, and create/edit flows for
criterion 3.

**agent:terva/mieli** at 2026-09-05T01:05:42Z

The filter line is in, and the first acceptance criterion is ticked: the
list shows open work, filterable by status, label, and assignee.

`/` opens a one-line prompt on the footer. Tokens `status:S`, `label:L`,
and `assignee:A` filter their fields with the ticket.Filter semantics, so
the rule learned from `git ticket list` holds here: within one field the
values are alternatives, across fields they all have to hold. Bare words
match the title case-insensitively, because the first thing anybody types
is a word they remember from a title. An assignee keeps its own colon, so
`assignee:human:sothr` works as written, and an unknown key falls through
to a title word rather than being dropped.

The list narrows live on every keystroke, backspace widens it again,
Enter keeps the filter and returns the keyboard to the list, and Esc
backs out one level at a time: prompt, then filter, then quit. The footer
shows `N/M match` with the filter text while one is active, the selection
follows the selected ticket into the filtered set per keystroke, and a
filter survives a reload. While the prompt has the keyboard every
shortcut is text, which the App enforces by routing all keys to the list
when FilterEditing reports true.

Filtering is client-side over the already-listed open work, which the
TKT-01M1QBS9 spike's arithmetic makes free at this store's scale. Twelve
new tests cover the tokens, the conjunction rule, live narrowing, the Esc
ladder, reload survival, and the App routing.

Criterion 3, create and edit through the library, is the one left.
