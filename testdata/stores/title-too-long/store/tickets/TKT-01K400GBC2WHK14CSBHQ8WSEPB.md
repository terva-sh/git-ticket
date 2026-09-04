---
schema: 1
id: TKT-01K400GBC2WHK14CSBHQ8WSEPB
title: Decide whether the readiness reason should also say which dependency is blocking, and whether that belongs in the JSON contract or only in the human rendering
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

The title is 158 characters, over the 120 of 5.1, so it is `title_too_long` and
an error.

This file is hand-written on purpose, because no write produces it. `create` and
`update` refuse a title this long with `invalid_field`, so the only way a store
holds one is somebody editing the file or a merge landing it. That is the same
shape as `invalid_priority`, which `check` exists to catch for the same reason.

Only one finding is raised. `title_long` is false once `title_too_long` is true,
so the error is not accompanied by a warning saying the same thing in weaker
words.
