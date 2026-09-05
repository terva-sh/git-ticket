---
schema: 1
id: TKT-01M1F7Z30Q3PZFS1Q7B0F715Z9
title: Import from Backlog.md, and build a local view
type: task
status: draft
status_reason: null
priority: low
due_on: null
labels:
  - question
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references:
  - ref: proposal:git-ticket
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-01T19:43:32Z
updated_at: 2026-09-05T01:38:38Z
created_by:
  id: human:sothr
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

A deferred question in plan section 15, and the rest of Phase 4. Backlog.md is the source of the acceptance criteria and definition of done fields, so an importer is a real path in. Worth building when somebody has a backlog to move.

## Notes

**agent:terva/mieli** at 2026-09-02T16:56:19Z

Still parked, but plan section 15 now splits this into two triggers. Import waits for a real Backlog.md project somebody wants to move, with the ticket count as the evidence. The local view keeps the Phase 4 condition, that the file and agent contracts have held through one real project.

**agent:terva/mieli** at 2026-09-05T01:38:38Z

The local-view half is done, so this ticket is now the importer alone.

TKT-01M1QBSD (Build the Phase 4 TUI view for browsing and editing
tickets) shipped `git ticket ui` in v0.7.0: an alt-screen view over the
store with browsing, filtering, status transitions, claim and release,
and create and edit flows. That is the local view this ticket held a
trigger for, and the Phase 4 condition it waited on, the file and agent
contracts holding through one real project, was judged met when the
view was built. The carve-out and the gate decision are recorded in
plan section 13.

This supersedes the local-view half of the 2026-09-02 grooming note
above, which said the view keeps the Phase 4 condition. It kept it, and
then the condition was met.

What remains is the importer, and its trigger is unchanged: a real
Backlog.md project somebody wants to move, with the ticket count as the
evidence. Nothing here is blocked.
