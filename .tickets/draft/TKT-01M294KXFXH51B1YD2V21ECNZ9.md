---
schema: 2
id: TKT-01M294KXFXH51B1YD2V21ECNZ9
title: Decide whether import reconciles assignees like labels
type: spike
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - question
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references:
  - ref: plan:12.8
    path: docs/plan.md
  - ref: plan:4.1
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-11T21:05:15Z
updated_at: 2026-09-11T23:06:09Z
created_by:
  id: agent:terva/mieli
  name: Mieli
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

`import --adopt` carries assignees verbatim. Every other foreign vocabulary is
reconciled: a label or milestone this store does not declare is dropped and
named, and a reference path that resolves to nothing here loses the path.

An assignee is the same shape of thing. It names an actor of the sending store,
and carrying it asserts that somebody in this store is on the hook for work they
have not seen.

### Why it was left alone rather than fixed

`check` does not validate assignees at all, so there is no finding and no
`--strict` failure either way. Nothing forces the question, which is exactly why
it is easy to answer wrongly by reflex.

There is also no allowlist to reconcile against. `Config` has `KnownSeries`,
`KnownLabel` and `KnownMilestone` and no `KnownActor`, so the label rule has no
direct analogue. `actors` exists in `config.yml`, but it is the list a bare write
is attributed from rather than an allowlist that gates one, and reading it as a
gate would be a new meaning for an existing field.

The three rulings that shaped the rest of the reconciliation on 2026-09-11 all
went the same way: what the receiver never agreed to does not travel. Applied
here that would drop a foreign assignee and name it. That is not obviously right
either, because an assignee can also read as provenance, a record of who did the
work at the origin, which is the argument for carrying it.

Deliberately not invented in the commit that fixed the rest. The trigger is
somebody meeting a real adopted ticket assigned to a person who does not work
here.

## Acceptance criteria

- [ ] Decided whether an assignee is vocabulary the receiver must declare, or provenance that travels
- [ ] If it is vocabulary, what it is reconciled against, given that config.actors is an attribution list and not a gate
- [ ] 12.8's What travels section says which it is
