---
schema: 1
id: TKT-01K400AVK0P8DHFD7CRR6DQYKV
title: Ticket carrying a priority outside the set
type: task
status: ready
status_reason: null
priority: p0
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-08-31T12:00:00Z
updated_at: 2026-08-31T12:00:00Z
created_by:
  id: human:sothr
  name: Drew Short
updated_by:
  id: human:sothr
  name: Drew Short
extensions: {}
---

## Description

`p0` is not one of the four priorities in 5.1. It comes from the numeric scale
most trackers use, so this is what an import or a habit produces.

The file parses and round-trips, and `check` reports `invalid_priority`.
Silently mapping `p0` to `urgent` would be a guess about someone else's scale,
where `p0` sometimes means the highest priority and sometimes means the one
above it.
