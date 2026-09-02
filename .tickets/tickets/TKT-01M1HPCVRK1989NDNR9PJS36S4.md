---
schema: 1
id: TKT-01M1HPCVRK1989NDNR9PJS36S4
title: Decide whether a ticket or a milestone carries a due date
type: spike
status: draft
status_reason: null
priority: low
labels:
  - question
  - format
assignees: []
milestone: null
parent: null
dependencies:
  - TKT-01M1HPB7ZT0FREYR04ZSHTMW3F
references:
  - ref: review:backlog-md
    path: docs/review-backlog-md.md
claim: null
archive: null
created_at: 2026-09-02T18:34:12Z
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

A deferred question raised by reading Backlog.md. See section B5 of docs/review-backlog-md.md.

The format has no date beyond created_at and updated_at. Backlog.md carries dueDate on both tasks and milestones, stored UTC at minute precision, rejecting date-only values, with --clear-due-date to remove one.

The trigger is a store where somebody has to answer what is late and is reduced to reading it out of prose in a description.

The real question is not whether to add a field but where to put it. A date on every ticket is the obvious answer and probably the wrong one: most tickets in an agent ledger have no deadline, and a null on every row buys nothing. A date on a milestone buys the same answer once, and list --milestone M already groups the tickets under it.

The cost, if the trigger is met, is set by plan 5.3: an absent scalar renders as null rather than being omitted, so a new frontmatter field means editing every fixture that carries frontmatter. status_reason cost exactly that, and the AGENTS.md note records that it was all 32 of them. Putting the date on a milestone avoids that entirely, because a milestone has no fixtures yet.

This waits on how the milestone allowlist lands, because a milestone with a date is a milestone with a file, which is the expensive half that ticket parks.
