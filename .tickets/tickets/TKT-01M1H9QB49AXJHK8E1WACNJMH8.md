---
schema: 1
id: TKT-01M1H9QB49AXJHK8E1WACNJMH8
title: Decide whether the binary should report its own version
type: spike
status: draft
status_reason: null
priority: low
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
references: []
claim: null
archive: null
created_at: 2026-09-02T14:52:44Z
updated_at: 2026-09-02T14:52:44Z
created_by:
  id: agent:terva/dev-loop
  name: ""
updated_by:
  id: agent:terva/dev-loop
  name: ""
extensions: {}
---

## Description

`just install` runs a plain `go install` with no ldflags, because there is nothing to read a stamped version back out of. A version surface would be a twenty-fifth command, so plan section 12.1 has to change before the code does. The question is whether a git-ticket build needs to identify itself at all: the schema version is already in every envelope, `schema` prints what the binary enforces, and a host embedding the cli package reports its own version instead. If the answer is yes, the justfile stamps main.version, main.commit, and main.date the way terva's does.
