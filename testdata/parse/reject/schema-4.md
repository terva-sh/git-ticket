---
schema: 4
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
meaning, so a reader one level behind cannot know whether `status: ready` still
means what it used to. Guessing would be worse than stopping, so the reader
refuses with `schema_unsupported` and names the version it would need.

The ID is recoverable here, unlike the parse failures, so the finding carries
it.

This fixture is always one level above what the reader supports, so its number
moves with `SchemaVersion`. It was `schema-2.md` until schema 2 shipped, per
plan 5.6, and `schema-3.md` until schema 3 shipped, per 12.5. Nothing else about
it changes: the point is the policy refusal, not any particular number.

One level above rather than a number far away, deliberately. The boundary is
what the gate gets wrong, so a fixture sitting at `SchemaVersion + 1` fails an
off-by-one where a `schema: 99` would sail past it and still look green.
