---
schema: 1
id: TKT-01M1HPB7Y3XXM026P6JSZADQAC
title: Put readiness and the reason for it on every ticket
type: task
status: ready
status_reason: null
priority: high
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

git ticket ready is a query that answers which tickets are startable. Nothing says why a given ticket is not, so a consumer holding a list has to call ready, diff two ID sets, then call deps per card to explain the difference.

Backlog.md carries the same verdict as a field on every task, isReady, and adds a readiness object to task detail with isBlocked, blockingDependencies, and missingDependencies. See section B1 of docs/review-backlog-md.md.

This costs the format nothing. It is computed and never stored, like revision and path in plan 7.1, so it is additive under 12.4 and no ticket file changes.

Copy their fail-closed rule exactly. A dependency that resolves to no ticket, or to more than one, blocks rather than counting as satisfied. check already reports dependency_missing, so we agree about the finding; the verdict should agree too.

Open: whether the reason detail rides on the ticket kind alone or on every row of ticket-list. The verdict is cheap on both. The two ID lists on several hundred rows is the case to measure before deciding.
