---
schema: 2
id: IDEA-01K400HRP8Q3XVT2M6BJ5NDZW4
title: Filed under a series this store never declared
type: task
status: ready
status_reason: null
priority: normal
due_on: null
labels: []
assignees: []
milestone: null
parent: null
origin: null
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

The ID is well formed. `IDEA` is two to eight uppercase characters starting
with a letter, so it passes the grammar of plan 5.6 and the file parses.

What is wrong is narrower: `config.yml` declares an empty `series` list, which
means `[TKT]` and not "anything goes", so nothing in this store declares
`IDEA`. That is `unknown_series`, and the repair is a line in `config.yml`
rather than a change to this file. The ID is immutable, per 5.6, so there is no
repair that edits it.

An ID that broke the grammar would never reach this check. It would fail to
parse and be reported as `parse_error`, which is why the two are separate
codes: one is a broken file and the other is a missing declaration.
