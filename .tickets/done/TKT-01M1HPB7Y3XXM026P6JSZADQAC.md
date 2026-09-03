---
schema: 1
id: TKT-01M1HPB7Y3XXM026P6JSZADQAC
title: Put readiness and the reason for it on every ticket
type: task
status: done
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
updated_at: 2026-09-02T19:10:11Z
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

## Implementation plan

1. Add ticket.Readiness and Store.Readiness, computed from the whole store
2. Rebuild Ready on the same derivation so the query and the field cannot disagree
3. Count claimants per ID so an ambiguous dependency fails closed
4. Carry it on every ticket in the JSON contract, one Readiness call per command
5. Print blocked by and unresolved in human show
6. Move plan 8, 10.1, and 12.2 in the same commit

## Summary

Every ticket now carries a derived readiness: the verdict ready filters on, plus what stands in the way when the answer is no.

Ready was a query and nothing else, so a consumer drawing a board had to call ready, diff two ID sets, and then call deps per card to explain the difference. The field is computed from the whole store at read time and never stored, like revision and path in 7.1, so no ticket file changed and this is additive under 12.4.

Ready is now a filter over that same derivation rather than a second copy of the rule, which is what stops the query and the field drifting apart. A test asserts the two agree as sets.

Blocked is about dependencies alone. A draft and a held ticket are unready with nothing in the way but their own state, and calling those blocked would send a reader looking for a dependency that is not there.

The fail-closed rule is copied from Backlog.md and is the part worth having. A dependency that resolves to nothing lands in missingDependencies. So does one that two files claim, because readinessOf counts claimants rather than just indexing by ID, and no single ticket claiming an ID means nothing can be said to have met it. Both are states check already reports, as dependency_missing and duplicate_id.

Human show gained blocked by and unresolved lines, named apart because the repair differs: a blocker needs finishing, an unresolved dependency needs the ticket it names or an unlink.
