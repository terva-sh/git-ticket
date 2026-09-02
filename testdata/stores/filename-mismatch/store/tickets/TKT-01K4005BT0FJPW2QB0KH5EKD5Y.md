---
schema: 1
id: TKT-01K4003H70NE86QR9CMVCZ52PN
title: Filename and id field disagree
type: task
status: ready
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

The file is named `TKT-01K4005BT0FJPW2QB0KH5EKD5Y.md` and the `id` field says
`TKT-01K4003H70NE86QR9CMVCZ52PN`. Copying an existing ticket to start a new one
and editing only half of it produces exactly this.

Section 4 makes the filename the ID and nothing else, so ID-to-path resolution
is a string operation with no index. A file that breaks that rule makes every
lookup by ID either miss or require a full scan, which is why this is an error
rather than a warning.

Both IDs here are well formed, so nothing else fires. The `id` field is the one
that carries meaning, so the finding names it as the field rather than the path.
