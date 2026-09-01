---
schema: 2
id: TKT-01K3ZYYXB0QP32SG8GF99N5VXW
title: Ticket written by a future major version
type: task
status: ready
priority: normal
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
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

Everything here parses. The refusal is a policy decision, not a syntax one.

A major bump is the only place 5.4 allows a field to be removed or given a new
meaning, so a v1 reader cannot know whether `status: ready` still means what it
used to. Guessing would be worse than stopping, so the reader refuses with
`schema_unsupported` and names the version it would need.

The ID is recoverable here, unlike the parse failures, so the finding carries
it.
