---
schema: 2
id: TKT-01M2P3FAHHBGXEMRRYE9R5AW5B
title: "Dilution and a declared order: neither picks the lead label alone"
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
origin: TKT-01M2NZE9Z4G033QJDFG62N4Y9D
dependencies: []
blocks_on: none
references:
  - ref: ticket:shipped
    path: .tickets/done/TKT-01M2P0FWJ1BEDGV3603X5YKR5G.md
claim: null
archive: null
created_at: 2026-09-16T21:55:24Z
updated_at: 2026-09-16T21:55:58Z
created_by:
  id: agent:claude/t3code
  name: ""
updated_by:
  id: agent:claude/t3code
  name: ""
extensions: {}
---

## Description

Two proposals are open for what should lead a ticket's labels, and measured against real stores each one fails exactly where the other succeeds. They are one design.

### The dilution half

Weight each label by `1 / (number of labels sharing its dimension)`. A ticket carrying `area/ci`, `area/docs`, `scope/structural` gives each area 0.5 and scope 1.0: needing two areas is itself evidence that no single area names the thing.

This detects something real. 34 of terva's open tickets carry two `area/` labels and 28 of ketju's carry two `area:` labels, and terva independently flagged the same 34 by hand. `label_order` as shipped in v0.19.1 is silent on all of them, because both labels lead with the store's dominant dimension.

### Why dilution alone does not work

**It cannot express the case that motivated it.** On `area/permissions`, `area/swarm`, `init/workspace-trust`, `scope/structural`, the two areas weigh 0.5 and *both* `init/` and `scope/` weigh 1.0. A tie, so no winner, so silence. That is 19 of terva's 34, and it is exactly the set where "the initiative should lead" is the intended answer. Nothing in the weights says an initiative identifies a ticket better than its blast radius does.

**Where it does pick a winner it often picks a bad one.** ketju has no initiative dimension. For 27 of its 28 diluted tickets the only fallback is `effort:`, so the weights say `effort:M` should lead over `area:db` and `area:identity`. That is worse than saying nothing: effort is a sizing estimate and names no subject. terva's own CONVENTIONS.md makes the same argument in refusing an effort dimension, "Effort is an estimate that rots and differs per person".

Measured totals: 13 findings on terva, missing the 19 that matter, and 27 on ketju, of which essentially all are wrong.

### Why a declared order alone does not work

TKT-01M2NZE9Z4G033QJDFG62N4Y9D proposes `order: [area/, init/, scope/]`, read as a total order a ticket's labels must follow. It cannot catch the 34 either, because `area, area, scope` does follow `area -> scope`. A conforming sequence with a diluted dimension is still conforming.

### Together

Dilution says which dimension stopped being identifying. The declared order says what is eligible to take over.

- terva declares `[area/, init/, scope/]`. `area/` is diluted, so the lead falls to `init/workspace-trust`. That is the intended answer for the 19.
- ketju declares `[area:]`. Nothing else is eligible to lead, so the bad advice is never generated and the rule stays silent on all 28.

The declared order also lets a store keep a dimension out of the running entirely, which is the part neither proposal has today and the reason the ketju case is currently unanswerable.

### Open questions

Whether this extends `label_order` or becomes a second rule. Adding it to `label_order` risks two contradictory findings on one ticket, since the v0.19.1 rule wants the dominant dimension to lead and this one wants a diluted dimension demoted.

Whether `order` should be inferred when undeclared, the way the dominant dimension already is, or whether declaring is required for any of this to fire.

What a store with no second eligible dimension should hear. Silence is defensible and so is a finding that names the dilution without prescribing a leader, since "this may want splitting or an initiative" is a real observation the tool can make honestly.

## Acceptance criteria

- [ ] A diluted dimension demotes its labels, and the lead falls to the next dimension the store declared eligible.
- [ ] A store that declares only one eligible dimension gets silence rather than a finding naming an ineligible one.
- [ ] The interaction with the v0.19.1 convention rule is settled so one ticket cannot draw two contradictory findings.
- [ ] Measured against terva, ketju, git-ticket-canvas and git-ticket before a default is chosen, with terva read from its ref.
- [ ] terva's 19 init-bearing tickets produce the initiative as the expected lead, and ketju's 28 produce nothing.
