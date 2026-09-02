---
schema: 1
id: TKT-01M1F7Z2Y5H1ZJAHRGF3XE6F91
title: Decide whether custom statuses are worth the cost
type: spike
status: draft
status_reason: null
priority: low
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
references:
  - ref: proposal:git-ticket
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-01T19:43:32Z
updated_at: 2026-09-02T16:56:19Z
created_by:
  id: human:sothr
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Deferred question 2 of plan section 15. The set in 6.1 is fixed and the transition table in 6.2 is built on it. Decide after a real workflow asks for a status that is not there, not before.

## Notes

**agent:terva/mieli** at 2026-09-02T16:56:19Z

Still parked, but the trigger in plan section 15 is now precise. It fires only when a workflow needs a state the seven statuses cannot express and that a label plus status_reason cannot express either. The second half is the real test, because labels are open and status_reason already answers why a ticket is blocked.
