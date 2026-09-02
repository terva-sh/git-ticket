---
schema: 1
id: TKT-01M1HZS4EQFS2102Y7TV75JKJ1
title: Epic waiting on a child that waits on the epic
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

This epic gates on its children, so it waits on
`TKT-01M1HZS4F4SZREHSRP8SZZ71SJ`. That child depends on this epic, so it waits
here. Neither can ever start.

The cycle alternates edge kinds, which is why it needs a code of its own. The
dependency graph holds one edge, child to epic, and is acyclic. The parent
graph holds one edge, child to epic, and is acyclic. Only the blocking graph,
which is dependencies plus the child edges of an epic with `blocks_on:
children`, closes the loop.
