---
schema: 1
id: TKT-01M1HPB7YY5K8DF1NF2E0GYE6K
title: Derive a structured comments view the way checklists are derived
type: task
status: ready
status_reason: null
priority: normal
labels:
  - format
assignees: []
milestone: null
parent: null
dependencies: []
references:
  - ref: review:backlog-md
    path: docs/review-backlog-md.md
claim: null
archive: null
created_at: 2026-09-02T18:33:19Z
updated_at: 2026-09-02T18:34:29Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

body.comments is one string. comment ID TEXT stamps an entry into it with the actor and the time, and a consumer that wants to render a thread has to parse that stamp format back out of prose. That is a second parser for a shape we already wrote.

Plan 10.1 settles the same question for checkboxes and lands the other way: body is the document, checklists is a one-way view of it derived by reading the lines, and the two cannot disagree because the derivation runs in one direction. Comments deserve the same treatment.

Backlog.md carries { index, body, createdAt, author } per comment. See section B2 of docs/review-backlog-md.md.

Scope: a comments array beside checklists in the ticket JSON, derived from body.comments, carrying index, text, actor, and time. Additive under 12.4.

Open: whether an entry a person hand-wrote without a stamp appears with a null actor and time, or does not appear at all. Absent is tidier and loses a comment somebody wrote, which argues for null. The format is meant to be hand-edited, so this case is ordinary rather than exotic.
