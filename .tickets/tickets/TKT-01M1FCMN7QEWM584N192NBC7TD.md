---
schema: 1
id: TKT-01M1FCMN7QEWM584N192NBC7TD
title: Decide how a caller reads the parent hierarchy back
type: spike
status: done
status_reason: null
priority: normal
labels:
  - format
assignees: []
milestone: null
parent: null
dependencies: []
references:
  - ref: proposal:git-ticket
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-01T21:05:13Z
updated_at: 2026-09-02T15:16:42Z
created_by:
  id: human:sothr
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Deferred question 7 of plan section 15. The format records a hierarchy it cannot show.

Section 8 filters list on status, type, priority, label, assignee, and milestone. Parent is not among them. deps walks dependencies rather than parent, so on an epic it correctly reports that it depends on nothing, which is true and useless. show does not render children either. So nothing lists the children of an epic.

This is a hole rather than a decision. Parent is settable in 12.1 through both create and update, and section 11 validates it with parent_missing and parent_cycle. The format went to the trouble of validating a tree it gives no way to walk.

Found by filing this repository Phase 3 epic with four slices and three decisions under it, then having no way to ask what is under the epic.

A --parent filter on list is the obvious answer and probably the right one, because it costs one filter in an existing command. show rendering a children section is defensible and helps a reader more. deps --children is defensible too and keeps hierarchy queries in one place, though it overloads a command named for a different relation. The choice affects the JSON contract in 10.1, which is why it belongs in the plan before it belongs in code.

## Notes

**agent:terva/mieli** at 2026-09-01T22:04:12Z

This is the one cross-repo dependency named in docs/handoff-terva-phase-3.md. Terva slice 4, the kanban board, will probably want to show an epic's children, and the format cannot answer that: parent is settable and validated with parent_missing and parent_cycle, but list has no --parent filter and deps walks dependencies, so asking an epic what is under it truthfully reports that it depends on nothing. It blocks nothing before slice 4. If the board design needs it, it moves up. Left draft because the fix changes the JSON contract in 10.1 and belongs in the plan before it belongs in code.

**agent:terva/mieli** at 2026-09-02T15:16:37Z

Running the question before answering it shrank it. list --json already returned parent on every ticket, so a consumer could always rebuild the whole tree from one call and terva's board was never blocked. What was missing was a way to ask for one epic's children without pulling the store, and any way at all for a person at a terminal to ask.

**agent:terva/mieli** at 2026-09-02T15:16:37Z

That is what settled the surface. A filter is additive under 12.4, so it ships as a minor release and 10.1 does not move, which was verified by probe: the ticket-list envelope still carries exactly schemaVersion, kind, tickets. show rendering children would have put data derived from other files onto the ticket object, and list returns []*Ticket so every row would have to compute it. deps --children would have overloaded a command whose whole contract is that it walks dependencies.

**agent:terva/mieli** at 2026-09-02T15:16:37Z

Two things fell out that were not in the question. Ticket.Parent holds a full ULID, so an exact match on a typed prefix would have returned an empty list silently, which reads exactly like an epic with no children; --parent now goes through the same resolveID that create and update use, so a prefix works and an unknown ID is ticket_not_found. And the corpus had no valid hierarchy at all: parent-missing and parent-cycle are both deliberately broken, so testdata/stores/hierarchy was added and TestCheckStores now asserts a clean hierarchy checks clean.

## Summary

Answered in plan section 8 by a --parent filter on list. It matches direct children only, and --parent none selects the tickets with no parent, which the library spells as an empty string on Filter.Parent following the milestone precedent. The CLI resolves a parent through the same resolveID that create and update use, so a prefix works and an ID naming nothing is ticket_not_found rather than a silent empty list. deps still walks dependencies alone, but when its answer is empty and the ticket has children the human output names the count and points at list --parent; --json does not carry that. 10.1 is unchanged, so this ships as a minor release under 12.4.
