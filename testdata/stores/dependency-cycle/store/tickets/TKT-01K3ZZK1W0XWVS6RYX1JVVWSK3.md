---
schema: 1
id: TKT-01K3ZZK1W0XWVS6RYX1JVVWSK3
title: Cycle member C, waits on A
type: task
status: ready
status_reason: null
priority: normal
labels: []
assignees: []
milestone: null
parent: null
dependencies:
  - TKT-01K3ZZFCP0VJKES58GZG0QDHG0
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-08-31T12:02:00Z
updated_at: 2026-08-31T12:02:00Z
created_by:
  id: human:sothr
  name: Drew Short
updated_by:
  id: human:sothr
  name: Drew Short
extensions: {}
---

## Description

The edge that closes the cycle back to A.

Every dependency here names a ticket that exists, so `dependency_missing` stays
quiet and `dependency_cycle` is the only finding.
