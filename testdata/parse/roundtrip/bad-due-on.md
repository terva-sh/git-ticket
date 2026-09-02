---
schema: 1
id: TKT-01M1J5JQ792T0KQ8ZFP5B6EYCT
title: Ticket carrying an instant where a date belongs
type: task
status: ready
status_reason: null
priority: normal
due_on: "2026-10-14T00:00:00Z"
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

An RFC3339 instant is the way this field realistically goes wrong. Every other
time value in 5.1 is one, so a writer reaching for the shape it already knows
produces exactly this file.

The file parses and round-trips, and `check` reports `invalid_due_on`.

Nothing truncates it to `2026-10-14`. The author of this value can be seen
choosing midnight, and a reader cannot tell that apart from a writer that
expanded a date somebody typed, which is the whole reason 5.1 stores the date as
written.

`2026-1-4` fails the same way for a duller reason. The check demands the exact
`YYYY-MM-DD` shape rather than whatever a date parser tolerates, so a store does
not end up holding two spellings of one day.
