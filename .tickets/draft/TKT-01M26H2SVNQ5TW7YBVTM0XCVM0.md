---
schema: 2
id: TKT-01M26H2SVNQ5TW7YBVTM0XCVM0
title: Show update metadata and relationships in the TUI detail header
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
  - ref: code:ticket-detail
    path: tui/view/detail.go
  - ref: plan:ui
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-10T20:45:23Z
updated_at: 2026-09-10T20:45:23Z
created_by:
  id: agent:terva/mieli
  name: Mieli
updated_by:
  id: agent:terva/mieli
  name: Mieli
extensions: {}
---

## Description

Expose enough metadata at the top of a ticket's TUI detail view to identify the ticket, see when and by whom it last changed, and understand what it is linked to without searching the body.

### Existing functionality

The header in tui/view/detail.go already shows the full ID, title, status, type, priority, and optional labels, assignees, milestone, due date, and status reason. Do not rebuild those fields. It does not show update metadata or ticket relationships in the header. The existing t picker navigates parent, children, dependencies, and dependents; retain that navigation rather than adding redundant keys.

### Proposed behavior

Add the last update time and actor. Show linked tickets with their relationship and title, retaining an identifying ticket reference. Keep the existing ID and metadata visible. The header must remain readable in narrow panes and leave usable space for the ticket body.

### Open questions

Settle which relationships belong in the header: parent, children, dependencies, dependents, origin, and arbitrary references need an explicit scope. Decide how to summarize large link sets and whether that summary points to the existing picker. Decide timestamp presentation, actor display, wrapping or collapsing behavior, and handling of missing targets. Do not imply that arbitrary reference targets resolve like ticket relationships.

### Trigger

The user requested a draft for richer metadata at the top of the ticket detail view. Promotion and layout decisions remain separate.

## Acceptance criteria

- [ ] The detail header retains the ticket ID and existing metadata while displaying the last update time and actor.
- [ ] The detail header shows the agreed relationship set with relationship labels, ticket references, and titles where targets resolve.
- [ ] The UI handles unresolved targets without crashing or hiding the rest of the ticket.
- [ ] The metadata remains readable in narrow panes and with many linked tickets, without making the body unusable.
- [ ] The existing t picker and linked-ticket back stack continue to work without redundant new navigation keys.
- [ ] The plan records the agreed relationship scope and presentation, and tests cover timestamps, missing targets, narrow panes, and large link sets.

## Implementation plan

1. Verify the current detail header and linked-ticket picker and settle the open scope and layout questions with the user.
2. Record the display behavior in the plan and reuse existing relationship lookup where suitable.
3. Add update metadata and relationship rendering while preserving body space and existing navigation.
4. Test with a fixed clock, narrow terminal dimensions, missing targets, and large relationship sets; review the layout in a real terminal.
