---
schema: 1
id: TKT-01K3ZZY1E0A2Y6DP5AWPQ90871
title: Archived status sitting in the live directory
type: task
status: archived
priority: normal
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
references: []
claim: null
archive:
  archived_at: 2026-09-12T09:00:00Z
  from_status: done
  reason: shipped in v1.1
created_at: 2026-08-31T12:00:00Z
updated_at: 2026-09-12T09:00:00Z
created_by:
  id: human:sothr
  name: Drew Short
updated_by:
  id: human:sothr
  name: Drew Short
extensions: {}
---

## Description

Somebody moved this file back to `tickets/` by hand, or a merge resurrected the
old path, and the status was never touched.

Section 6.3 makes the status authoritative, so this ticket is archived and the
directory is wrong. `check` reports `archive_location_mismatch` and a reader
must not decide the ticket is live just because of where the file sits.

Resolving it the other way would be worse. Trusting the directory would let a
`git mv` silently reopen finished work, and a path is not a field anyone
reviews.
