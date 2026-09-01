---
schema: 1
id: TKT-01K3ZZW6V0YWD5KSGS7500YDAP
title: Archived out of ready, so it satisfies nothing
type: task
status: archived
status_reason: null
priority: normal
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
references: []
claim: null
archive:
  archived_at: 2026-09-18T09:00:00Z
  from_status: ready
  reason: descoped, nobody ever started it
created_at: 2026-08-31T12:02:00Z
updated_at: 2026-09-18T09:00:00Z
created_by:
  id: human:sothr
  name: Drew Short
updated_by:
  id: human:sothr
  name: Drew Short
extensions: {}
---

## Description

Archived without ever being done. The work does not exist, so anything
depending on this ticket is still blocked no matter what the archive says.

Archiving is not a way to close a dependency. That is the distinction
`from_status` exists to record, and it is why the finding is a warning rather
than an error: descoping work is legitimate, but a dependent that silently
looks ready is not.
