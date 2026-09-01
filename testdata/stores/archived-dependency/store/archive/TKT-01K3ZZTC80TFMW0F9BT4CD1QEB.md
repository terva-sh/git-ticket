---
schema: 1
id: TKT-01K3ZZTC80TFMW0F9BT4CD1QEB
title: Archived out of done, so it still counts
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
  archived_at: 2026-09-15T09:00:00Z
  from_status: done
  reason: shipped in v1.1
created_at: 2026-08-31T12:01:00Z
updated_at: 2026-09-15T09:00:00Z
created_by:
  id: human:sothr
  name: Drew Short
updated_by:
  id: human:sothr
  name: Drew Short
extensions: {}
---

## Description

The work finished and then the ticket was archived, which is the ordinary
order. `from_status: done` preserves that fact after the status became
`archived`, so this ticket keeps satisfying its dependents.
