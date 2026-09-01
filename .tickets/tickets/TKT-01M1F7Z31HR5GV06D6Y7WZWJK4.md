---
schema: 1
id: TKT-01M1F7Z31HR5GV06D6Y7WZWJK4
title: Set the compatibility policy for the module after the first stable schema
type: spike
status: draft
status_reason: null
priority: normal
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
references:
  - ref: proposal:git-ticket
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-01T19:43:32Z
updated_at: 2026-09-01T19:43:32Z
created_by:
  id: human:sothr
  name: ""
updated_by:
  id: human:sothr
  name: ""
extensions: {}
---

## Description

Deferred question 6 of plan section 15. Schema 1 is what the library reads and writes. Decide what a schema 2 means for a schema 1 reader, and what the Go module promises across it, before anything outside this repository depends on either.
