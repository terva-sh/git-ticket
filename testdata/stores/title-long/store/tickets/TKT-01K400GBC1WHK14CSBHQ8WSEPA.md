---
schema: 1
id: TKT-01K400GBC1WHK14CSBHQ8WSEPA
title: Decide whether the readiness reason should also say which dependency is blocking
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

The title is 80 characters, past the 72 of 5.1 and short of the 120 a write may
store, so it is `title_long` and a warning rather than an error.

A warning is what this threshold is for. A store that predates the rule keeps
working, and its long titles surface without anybody's build going red for a
sentence that was legal when it was written. `--strict` is how a repository that
wants the tighter reading gets it.

Nothing is ever rewritten. Only the author knows which half of a title was the
part worth keeping.
