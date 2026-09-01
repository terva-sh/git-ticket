---
schema: 1
id: TKT-01K3ZYJ360Q7ESC30QAD2SMY0H
title: Cache the provider model list
type: chore
status: ready
priority: low
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
severity: high
---

## Description

Written by a reader one minor version ahead, which added a `severity` field.

A v1.0 reader must parse this file, warn on stderr, keep `severity` on write,
and report `unknown_field` from `check`. Dropping the field would corrupt the
ticket for the newer reader in the same repository, which is the mixed-speed
client problem 5.4 exists to survive.

The field sits after `extensions` because 5.3 renders unknown keys after the
known ones. A fixture with it anywhere else could not round-trip identically.
