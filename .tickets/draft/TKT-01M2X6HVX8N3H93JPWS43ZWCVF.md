---
schema: 2
id: TKT-01M2X6HVX8N3H93JPWS43ZWCVF
title: Boards lists layout files that Load refuses
type: bug
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - area/format
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-19T16:03:54Z
updated_at: 2026-09-19T16:03:54Z
created_by:
  id: agent:claude/t3code-a6d0ff31
  name: ""
updated_by:
  id: agent:claude/t3code-a6d0ff31
  name: ""
extensions: {}
---

## Description

layout.Store.Boards returns every non-directory entry under .tickets/canvas ending in .yml with the suffix stripped, without applying boardNameOK, while Store.path applies it and Load refuses anything else. A file such as "with space.yml" is listed as a board and then fails to open with "invalid board name". The canvas's board picker reads Boards, so a stray file becomes a board a person can select and cannot open. Found by Terva review 32 on git-ticket PR 208 while the package was being moved in unchanged; the fix was kept out of that PR to preserve byte-identical behaviour with the canvas copy. Fix: filter names through boardNameOK in Boards, and add a test with a malformed filename beside a valid board. Decide whether to log or silently skip the file; the canvas lists an unopenable store with its reason rather than hiding it, and the same rule probably applies here.
