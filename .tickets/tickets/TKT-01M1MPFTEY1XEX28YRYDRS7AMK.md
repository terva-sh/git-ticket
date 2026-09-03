---
schema: 1
id: TKT-01M1MPFTEY1XEX28YRYDRS7AMK
title: Add section 15 entries for the two questions from the Backlog.md review
type: task
status: ready
status_reason: null
priority: normal
due_on: null
labels:
  - policy
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references:
  - ref: plan:deferred-questions
    path: docs/plan.md
  - ref: review:backlog-md
    path: docs/review-backlog-md.md
claim: null
archive: null
created_at: 2026-09-03T22:33:32Z
updated_at: 2026-09-03T23:34:02Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

AGENTS.md makes plan section 15 the home for a deferred question: "When the plan does not answer a question, add it to section 15 instead of inventing an answer." Two parked questions never got an entry there.

TKT-01M1HPCJH5RWSA6Z28QM3QV1H0, whether the format stores a hand-set order, and TKT-01M1HPCJJ1FFHG7HXC8QG1JRAG, whether a query reads tickets from other branches. Both arrived through docs/review-backlog-md.md, its sections B6 and D, rather than through section 15, so the plan never grew a paragraph for either. Every other ticket carrying the question label has one.

The gap is not cosmetic. TKT-01M1HJKBYGENBTJ7F9S71BN3Q1 settled that section 15 stays hand-maintained prose rather than a query against the store, on the argument that the entries carry reasoning longer than the ticket descriptions they name. A question with no entry has nowhere for that reasoning to live, and a reader working from the plan alone cannot see that either question is open.

It surfaced while cutting v0.5.0. The README claimed that seven questions sit in plan section 15, filed as tickets, which is false for these two. PR #62 replaced that line with a pointer to the question label query, which is correct, but it routed around the gap rather than closing it.

The work is a paragraph in section 15 for each, in the form the other entries use: the subject in bold, the ULID, and the trigger the ticket already records. Both were groomed on 2026-09-03 and their notes carry current triggers, so the material exists and is not invented here. Append in ULID order, which is the convention the end of the TKT-01M1HJKB entry records.

Worth checking while there: whether any other ticket carrying the question label is missing an entry, and whether the reverse holds, an entry naming a ticket that no longer exists.

## Acceptance criteria

- [ ] docs/plan.md section 15 carries an entry for TKT-01M1HPCJH5RWSA6Z28QM3QV1H0
- [ ] docs/plan.md section 15 carries an entry for TKT-01M1HPCJJ1FFHG7HXC8QG1JRAG
- [ ] Every open ticket carrying the question label has a section 15 entry, and every section 15 entry names a ticket that exists
