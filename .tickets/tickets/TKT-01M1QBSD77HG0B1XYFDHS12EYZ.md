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
updated_at: 2026-09-05T00:00:37Z
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

- [ ] a list view shows open work, filterable by status, label, and assignee
- [ ] the detail view renders the ticket Markdown with its sections
- [ ] create and edit write through the library, and a stale revision re-presents instead of overwriting
- [ ] status transitions, claim, and release are reachable from the list
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
