---
schema: 1
id: TKT-01K400HRM0V8QDX5N2WTFB3CJ7
title: Left behind when a migration stopped partway
type: task
status: ready
status_reason: null
priority: normal
due_on: null
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

`config.yml` declares schema 2 and this ticket declares 1, which is the state
plan 12.5 leaves behind when a migration is interrupted by a crash or a full
disk.

It is a warning rather than an error because the store is correct for a reader
that understands both levels. Nothing here is malformed and every query still
returns this ticket. A half-finished job should still not be invisible, which
is what the finding is for.

`check --fix` does not repair it. The repair is `git ticket migrate`, which
rewrites every ticket in the store under the lock, and 12.5 makes a store move
only through a migration a person runs. `check --fix` is what CI runs, so a
`--fix` that migrated would take that decision on a job's behalf.

The order is what makes this state reachable at all: 12.5 writes `config.yml`
before any ticket, so an interrupted run leaves the declaration ahead of the
files rather than behind them. A store in the other shape would have tickets a
reader refuses inside a config it accepts, which is the failure that ordering
exists to prevent.
