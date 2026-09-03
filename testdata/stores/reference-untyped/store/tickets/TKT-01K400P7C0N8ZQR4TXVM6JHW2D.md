---
schema: 1
id: TKT-01K400P7C0N8ZQR4TXVM6JHW2D
title: One reference carries a namespace and one does not
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
references:
  - ref: jira:PROJ-1234
    path: null
  - ref: PROJ-1234
    path: null
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

Two references naming the same work item in the same tracker, one finding.
`jira:PROJ-1234` is the typed stable identifier 5.1 describes and says nothing.
`PROJ-1234` is the recorded warning: it has no namespace, so `git ticket refs
jira:PROJ-1234` cannot reach it and `git ticket refs jira:` does not list it.

Both refs carry a null path, so nothing here can raise
`reference_path_unresolved` and the fixture stays about one code.

The pair matters more than the untyped ref alone. A check that warned on every
reference would pass a fixture holding only the bad one, so the good one is
what proves it discriminates.
