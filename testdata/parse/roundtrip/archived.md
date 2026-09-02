---
schema: 1
id: TKT-01K3ZYSDJ0C9FYEK5D8R0BBT5Q
title: Drop the legacy config loader
type: chore
status: archived
status_reason: null
priority: normal
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references: []
claim: null
archive:
  archived_at: 2026-09-04T09:00:00Z
  from_status: done
  reason: shipped in v1.2
created_at: 2026-08-31T12:00:00Z
updated_at: 2026-09-04T09:00:00Z
created_by:
  id: human:sothr
  name: Drew Short
updated_by:
  id: human:sothr
  name: Drew Short
extensions: {}
---

## Description

An archived ticket that reached `archived` the ordinary way, through `done`.

`from_status: done` is what keeps this ticket satisfying its dependents after
the archive, per 6.3. Without it every archive would silently block whatever
depended on the work.

The status is authoritative and the directory follows it. Whether this file
sits in the right directory is a store-level question, so the fixture for
`archive_location_mismatch` lives under `stores/` instead.
