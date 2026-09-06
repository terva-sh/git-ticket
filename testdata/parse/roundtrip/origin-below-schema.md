---
schema: 1
id: TKT-01K3ZYS9X0T6BFN4Q2WMD7HC5R
title: Cache the provider model list
type: chore
status: ready
status_reason: null
priority: low
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
origin: TKT-01K3ZYRB40N8QVD2M5T7XCFH3J
---

## Description

A schema 1 ticket carrying a key that schema 2 defines, per plan 5.6.

`origin` is unknown at this level, so it takes the 5.4 path rather than the
5.1 one: the reader parses the rest, keeps the key, and `check` reports
`unknown_field`. It renders after `extensions` because 5.3 puts unknown keys
after the known ones, which is the same position `unknown-field.md` relies on.

The alternative would have been to parse `origin` into its struct field at
every level and let the renderer decline to emit it below schema 2. That reads
as the smaller change and it silently deletes this line on the next write. A
field a level does not define has to be unknown to it, or the round trip is not
a round trip.

Compare `origin.md`, which is the same key at the level that defines it.
