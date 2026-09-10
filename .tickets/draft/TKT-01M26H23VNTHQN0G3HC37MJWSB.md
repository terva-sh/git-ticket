---
schema: 2
id: TKT-01M26H23VNTHQN0G3HC37MJWSB
title: Open the TUI on a ticket supplied to git ticket ui
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels: []
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references:
  - ref: code:ui-command
    path: cli/ui.go
  - ref: code:ticket-detail
    path: tui/view/detail.go
  - ref: plan:ui
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-10T20:45:00Z
updated_at: 2026-09-10T20:45:00Z
created_by:
  id: agent:terva/mieli
  name: Mieli
updated_by:
  id: agent:terva/mieli
  name: Mieli
extensions: {}
---

## Description

Let a human or model invoke git ticket ui <ID> to open a specific ticket directly, analogous to git ticket show <ID> but interactive.

### Existing functionality

cli/ui.go currently rejects every positional argument with "ui takes no arguments". It opens the store and delegates through Env.RunUI. The UI already has a detail stack for navigating linked tickets. This request adds an entry path, not a second detail renderer.

### Proposed behavior

Accept one optional ticket ID using the same unique-abbreviation resolution as show. Open the resolved ticket's detail view immediately. Esc returns to the list; invoking ui without an ID retains the current list entry. Missing or ambiguous references should fail before entering the terminal UI. Direct entry should work for done and archived tickets as well as open work.

### Implementation constraints

Keep terminal dependencies out of cli. UIParams and view.StoreParams must remain field-for-field compatible with their cast in cmd/git-ticket/main.go. Preserve the no-JSON rule for ui and the existing linked-ticket navigation stack.

### Trigger

The user requested a draft for direct ticket entry. Promotion remains a separate decision.

## Acceptance criteria

- [ ] git ticket ui <ID> opens the resolved ticket directly in the existing detail view.
- [ ] The ID accepts the same unique abbreviations as show; missing and ambiguous references fail before entering the TUI.
- [ ] Direct entry works for open, done, and archived tickets.
- [ ] Esc from the directly opened ticket returns to the list, and linked-ticket navigation retains its existing back-stack behavior.
- [ ] git ticket ui without an ID retains its existing behavior, and ui still refuses --json.
- [ ] The plan and command help document the optional ID, with CLI and TUI tests covering the new entry path.

## Implementation plan

1. Record the optional ID and navigation behavior in the plan before implementation.
2. Resolve the optional reference in cli and pass the initial ticket through the matching UI parameter structs.
3. Open the existing detail view on startup without replacing the list or linked-ticket stack.
4. Test argument handling, resolution errors, completed and archived sources, and Esc behavior; exercise the actual terminal entry path.
