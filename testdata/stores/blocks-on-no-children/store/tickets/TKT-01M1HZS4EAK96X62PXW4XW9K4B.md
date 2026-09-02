---
schema: 1
id: TKT-01M1HZS4EAK96X62PXW4XW9K4B
title: Epic gating on a decomposition nobody wrote
type: epic
status: ready
status_reason: null
priority: normal
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: children
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

This epic gates on its children and has none. No ticket in the store names it
as a parent.

It is not blocked. Blocking it would put a ticket in the blocked state with
nothing to name as the blocker, which section 8 refuses for a draft and for a
held ticket on the same grounds: it sends a reader looking for a dependency
that is not there.

Status is the guard that matters. A new ticket is a draft and never reaches
`ready`, so this state can only be reached by promoting an undecomposed epic by
hand. That is an authoring mistake, and a warning is the instrument for one.
