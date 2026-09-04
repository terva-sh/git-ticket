---
schema: 1
id: TKT-01M1QBSD77HG0B1XYFDHS12EYZ
title: Build the Phase 4 TUI view for browsing and editing tickets
type: task
status: draft
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
claim: null
archive: null
created_at: 2026-09-04T23:24:15Z
updated_at: 2026-09-04T23:24:15Z
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
- [ ] tests drive a fake terminal and assert the emulated cell grid
