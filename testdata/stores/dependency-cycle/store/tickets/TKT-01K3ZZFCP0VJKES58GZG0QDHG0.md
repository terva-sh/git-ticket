---
schema: 1
id: TKT-01K3ZZFCP0VJKES58GZG0QDHG0
title: Cycle member A, waits on B
type: task
status: ready
priority: normal
labels: []
assignees: []
milestone: null
parent: null
dependencies:
  - TKT-01K3ZZH790E1HXA78V5PGPBVQ0
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

A waits on B, B waits on C, C waits on A. Every ticket in the cycle is `ready`
and none can ever become satisfiable.

This is the case that makes `ready` loop forever if dependency resolution walks
the graph without tracking visited nodes, which is why the corpus carries it.
