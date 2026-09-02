---
schema: 1
id: TKT-01M1F7Z2Z33HW6FW44TCQVWB7M
title: Decide whether to add sync helpers around ordinary Git commands
type: spike
status: done
status_reason: null
priority: low
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
updated_at: 2026-09-02T16:12:19Z
created_by:
  id: human:sothr
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Deferred question 3 of plan section 15. Publishing is the user's ordinary Git workflow and a helper that does it for them is out of scope for v1. The library runs no Git command that writes.

## Summary

Answered no in plan section 15, and Phase 4 no longer holds a slot for a remote helper. Two mechanisms had already decided it. Plan 7.4 enumerates the Git commands this code may run and TestGitCommandsAreReadOnly holds the source to that table, so adding fetch or push cannot happen quietly. The multi-agent friction that would justify a helper did arrive, and a process rule absorbed it rather than a tool, because work now lands on main through a pull request and pushing to main is forbidden. The friction raised Q9 instead, which is about merging two edits of one ticket and needs no network.
