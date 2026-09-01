---
schema: 1
id: TKT-01M1FCMN7QEWM584N192NBC7TD
title: Decide how a caller reads the parent hierarchy back
type: spike
status: draft
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
updated_at: 2026-09-01T21:05:13Z
created_by:
  id: human:sothr
  name: ""
updated_by:
  id: human:sothr
  name: ""
extensions: {}
---

## Description

Deferred question 7 of plan section 15. The format records a hierarchy it cannot show.

Section 8 filters list on status, type, priority, label, assignee, and milestone. Parent is not among them. deps walks dependencies rather than parent, so on an epic it correctly reports that it depends on nothing, which is true and useless. show does not render children either. So nothing lists the children of an epic.

This is a hole rather than a decision. Parent is settable in 12.1 through both create and update, and section 11 validates it with parent_missing and parent_cycle. The format went to the trouble of validating a tree it gives no way to walk.

Found by filing this repository Phase 3 epic with four slices and three decisions under it, then having no way to ask what is under the epic.

A --parent filter on list is the obvious answer and probably the right one, because it costs one filter in an existing command. show rendering a children section is defensible and helps a reader more. deps --children is defensible too and keeps hierarchy queries in one place, though it overloads a command named for a different relation. The choice affects the JSON contract in 10.1, which is why it belongs in the plan before it belongs in code.
