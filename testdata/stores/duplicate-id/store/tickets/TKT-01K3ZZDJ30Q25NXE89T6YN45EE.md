---
schema: 1
id: TKT-01K3ZZDJ30Q25NXE89T6YN45EE
title: Same ID in two places, live copy
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

An archive that copied instead of moving leaves the same ID in `tickets/` and
in `archive/`.

Both files are named `<id>.md` and each status agrees with the directory it
sits in, so `filename_id_mismatch` and `archive_location_mismatch` stay quiet.
`duplicate_id` is the only finding, which is the point: the fixture isolates
one code.
