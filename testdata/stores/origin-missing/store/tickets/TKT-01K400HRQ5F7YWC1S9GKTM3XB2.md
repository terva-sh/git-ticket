---
schema: 2
id: TKT-01K400HRQ5F7YWC1S9GKTM3XB2
title: Records an origin that is not in this store
type: task
status: ready
status_reason: null
priority: normal
due_on: null
labels: []
assignees: []
milestone: null
parent: null
origin: TKT-01K400HRZZZZZZZZZZZZZZZZZZ
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

`origin` names a ticket that does not exist, which is `origin_missing`. It is
the same finding `parent_missing` is, and deliberately so: plan 5.6 makes
provenance a frontmatter field rather than a `ticket:` reference precisely
because `check` verifies that `parent` resolves and verifies no reference
target, and provenance wants the stronger guarantee.

There is no `origin_cycle` beside it. Nothing walks `origin` transitively, so a
cycle in it gates no work and breaks no query, and a finding naming a condition
with no consequence spends a reader's attention for nothing. A ticket naming
itself is refused at the point of writing, with `invalid_field`.
