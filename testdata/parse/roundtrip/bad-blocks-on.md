---
schema: 1
id: TKT-01M1HZGA5WG085NMZ00X2C6K7P
title: Epic carrying a blocks_on outside the set
type: epic
status: ready
status_reason: null
priority: normal
due_on: null
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: childrn
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

`childrn` is a typo for `children`, which is the way this field realistically
goes wrong. The value is one keystroke from correct and reads as correct.

The file parses and round-trips, and `check` reports `invalid_blocks_on`.

This is the one enum whose invalid value fails quietly rather than loudly.
A bad `status` stops a transition and a bad `type` shows up in every listing,
but a `blocks_on` nothing recognises simply is not `children`, so the epic stops
gating on its own decomposition and every child reads as optional. Nothing else
in the store would say so.
