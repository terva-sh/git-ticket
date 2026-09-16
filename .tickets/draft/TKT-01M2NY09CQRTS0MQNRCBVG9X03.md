---
schema: 2
id: TKT-01M2NY09CQRTS0MQNRCBVG9X03
title: A doctor rule for a ticket filed with no acceptance criteria
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - area/doctor
assignees: []
milestone: null
parent: null
origin: TKT-01M2NJDAVHTXKEPJ0ZCAJP3QY0
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-16T20:19:48Z
updated_at: 2026-09-16T20:20:03Z
created_by:
  id: agent:claude/t3code
  name: ""
updated_by:
  id: agent:claude/t3code
  name: ""
extensions: {}
---

## Description

`doctor` ships two rules and both are about labels. The gap named twice during the v0.19.0 work, and never filed until now, is a ticket that carries no acceptance criteria.

### Why this one

It has a real trigger rather than a hypothetical one. terva filed three tickets on 2026-09-16 (TKT-01M2NT7QS, TKT-01M2NT7VM, TKT-01M2NT7VN) and none of them carried acceptance criteria. Two of the three also carried a factual error about this tool's behaviour, and criteria are where such a claim would have been written as something checkable rather than as prose nobody tests. A criterion is the cheapest place a wrong assumption gets caught.

It is also the finding `check` will never make. Criteria are optional in the format, per plan 6, so a ticket without them is valid and always will be. That is exactly the shape doctor exists for: untidy rather than invalid.

### Which level

Probably hard, but that is the question to settle rather than assume. Hard means the store's own configuration already settled it, and no config key today says "this store expects criteria". Either the rule is soft, phrased as a question the way `label_order` is, or the level is earned by a config key a store sets deliberately.

Worth weighing: a draft is where a ticket is filed before anybody has thought it through, so firing on drafts may be reporting the system working as designed. Scoping to non-draft open tickets is the obvious narrowing, and `status` is the natural parameter.

### Measure before choosing a default

The v0.19.0 `label_order` threshold is the precedent, including how it went wrong. Its default moved on a measurement of 48 findings against terva that was taken from a stale working tree; the real figure was 120, and v0.19.1 replaced the threshold entirely. Measure from a sibling store's ref, never from whatever its working tree is checked out at. Do the same here: count what this rule would report against git-ticket, terva, terva-ext-web and git-ticket-canvas before picking a default, and record the numbers.

## Acceptance criteria

- [ ] The rule reports an open ticket that carries no acceptance criteria.
- [ ] Its level is hard or soft with the reason recorded, and if hard, what configuration earns that level is named.
- [ ] Whether drafts are in scope is decided and configurable, with the default recorded.
- [ ] The finding count this rule would produce is measured against git-ticket, terva, terva-ext-web and git-ticket-canvas before a default is chosen, and the numbers are in the ticket.
- [ ] The rule ID is registered in the shared namespace and published by schema, with the collision test still passing.
