---
schema: 1
id: TKT-01M1FCJ8ZBT1P6JRS86B5DBRBX
title: Decide whether ticket_deps and ticket_files join the tool surface
type: spike
status: draft
status_reason: null
priority: low
labels:
  - integration
assignees: []
milestone: null
parent: TKT-01M1FCHH3QN04AYHZAP1M8DNQK
dependencies: []
references: []
claim: null
archive: null
created_at: 2026-09-01T21:03:55Z
updated_at: 2026-09-01T21:03:55Z
created_by:
  id: human:sothr
  name: ""
updated_by:
  id: human:sothr
  name: ""
extensions: {}
---

## Description

Both are read-only queries the CLI already has, and both were left out of the first ten rather than rejected.

git ticket files PATH lists the tickets referencing a path, which suits an agent well: it answers what work is already tracked against the file I am about to edit. deps walks the dependency graph.

Options: add them as their own local-read tools, fold deps into ticket_get as a field and files into ticket_list as a filter, or leave both to the CLI. Folding keeps the tool count down, which matters because every schema costs context.

Not urgent. Revisit after slice 2 has been used.
