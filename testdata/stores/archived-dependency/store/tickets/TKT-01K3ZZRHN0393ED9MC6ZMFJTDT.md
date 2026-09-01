---
schema: 1
id: TKT-01K3ZZRHN0393ED9MC6ZMFJTDT
title: Live ticket waiting on two archived ones
type: task
status: ready
priority: normal
labels: []
assignees: []
milestone: null
parent: null
dependencies:
  - TKT-01K3ZZTC80TFMW0F9BT4CD1QEB
  - TKT-01K3ZZW6V0YWD5KSGS7500YDAP
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

Both dependencies are archived, and only one of them counts.

The first was archived out of `done`, so 6.3 treats it as satisfied and it
produces nothing. The second was archived out of `ready`, so the work never
happened and this ticket is not actually unblocked. That second one is the
`dependency_archived_incomplete` warning.

This pair is the whole argument for recording `from_status`. Without it both
dependencies look identical from here, and the ordinary flow of finishing a
ticket and then archiving it would silently block every dependent.

The finding lands on this file rather than on the archived one, because this is
the ticket whose readiness is wrong and this is the file a person has to edit.
