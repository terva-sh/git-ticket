---
schema: 1
id: TKT-01K3ZZH790E1HXA78V5PGPBVQ0
title: Cycle member B, waits on C
type: task
status: ready
status_reason: null
priority: normal
labels: []
assignees: []
milestone: null
parent: null
dependencies:
  - TKT-01K3ZZK1W0XWVS6RYX1JVVWSK3
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-08-31T12:01:00Z
updated_at: 2026-08-31T12:01:00Z
created_by:
  id: human:sothr
  name: Drew Short
updated_by:
  id: human:sothr
  name: Drew Short
extensions: {}
---

## Description

The middle of the cycle. Nothing here is malformed on its own, which is the
point: no single file can exhibit `dependency_cycle`, so this case has to be a
store.
