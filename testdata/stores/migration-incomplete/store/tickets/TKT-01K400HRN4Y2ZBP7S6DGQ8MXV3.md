---
schema: 2
id: TKT-01K400HRN4Y2ZBP7S6DGQ8MXV3
title: Already converted before the migration stopped
type: task
status: ready
status_reason: null
priority: normal
due_on: null
labels: []
assignees: []
milestone: null
parent: null
origin: null
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

This ticket is at the level `config.yml` declares, so it produces no finding.
It is here so the fixture shows the check is selective rather than reporting
every ticket in a store whose config moved.

It carries `origin`, which is the key schema 2 adds, per plan 5.6. The ticket
beside it does not, because a field renders only at the level that introduced
it, and that is what makes the two files differ by more than a number.

The next `git ticket migrate` skips this one and rewrites only its neighbour,
which is the idempotence 12.5 requires: a run interrupted partway is finished
by running it again.
