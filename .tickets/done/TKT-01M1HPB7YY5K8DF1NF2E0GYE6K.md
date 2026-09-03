---
schema: 1
id: TKT-01M1HPB7YY5K8DF1NF2E0GYE6K
title: Derive a structured comments view the way checklists are derived
type: task
status: done
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
updated_at: 2026-09-02T19:19:24Z
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

## Implementation plan

1. Add ticket.Entry and ticket.Entries, a one-way reading of a stamped log
   section, next to Checklist in mutation.go.
2. Carry the reading as a top-level comments array in the JSON contract,
   parallel to checklists, with actor and at nullable.
3. Settle where an entry ends and record it in plan 10.1.

## Summary

Added ticket.Entries and a top-level comments array in the JSON.

The open question is settled as null rather than absent. A ticket is a file a
person edits, so prose typed in without a stamp is an ordinary case, and the
text somebody left is still a comment they left. It comes back with a null
actor and a null time.

Answering it raised the question it was hiding: where an entry ends. An entry
runs from its stamp to the next stamp, so a blank line inside one does not end
it and a comment can have several paragraphs. The cost is that prose appended
below a stamped entry joins it and reads as that author's. Splitting on the
blank line was the alternative and fixes nothing, because the fragment sits
under that stamp either way and would still carry that actor, while every
multi-paragraph comment would come apart. Plan 10.1 records both.

Notes use the same stamp and read back with the same call. Only comments is in
the contract: a note is written for the ticket, a comment is written to
somebody.
