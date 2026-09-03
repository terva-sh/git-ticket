---
schema: 1
id: TKT-01M1F7Z30Q3PZFS1Q7B0F715Z9
title: Import from Backlog.md, and build a local view
type: task
status: draft
status_reason: null
priority: low
labels:
  - question
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
updated_at: 2026-09-02T18:06:04Z
created_by:
  id: human:sothr
  name: ""
updated_by:
  id: agent:terva/dev-loop
  name: ""
extensions: {}
---

## Description

A deferred question in plan section 15, and the rest of Phase 4. Backlog.md is the source of the acceptance criteria and definition of done fields, so an importer is a real path in. Worth building when somebody has a backlog to move.

## Notes

**agent:terva/mieli** at 2026-09-02T16:56:19Z

Still parked, but plan section 15 now splits this into two triggers. Import waits for a real Backlog.md project somebody wants to move, with the ticket count as the evidence. The local view keeps the Phase 4 condition, that the file and agent contracts have held through one real project.
