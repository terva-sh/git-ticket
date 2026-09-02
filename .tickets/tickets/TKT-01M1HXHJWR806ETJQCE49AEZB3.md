---
schema: 1
id: TKT-01M1HXHJWR806ETJQCE49AEZB3
title: Let an epic block on its children without enumerating them
type: task
status: draft
status_reason: null
priority: normal
labels:
  - format
  - question
assignees: []
milestone: null
parent: null
dependencies: []
references:
  - ref: plan:epic-blocking
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-02T20:39:07Z
updated_at: 2026-09-02T20:39:30Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

An epic that gates on its children can only say so today by listing them in `dependencies`. That works, and 6.3 already makes the epic go ready exactly when the listed children are done. It makes the epic file a hot spot: adding a child edits the epic's dependency list, so two agents decomposing the same epic collide on one line of one file. AGENTS.md names that collision as the reason this repository uses pull requests at all.

Proposal: let an epic state the rule once instead of maintaining a set.

```yaml
blocks_on: none | listed | children
```

`none` is today's behaviour and the default, so every existing ticket and every fixture is unchanged. `listed` is today's `dependencies` list. `children` derives the blocking set from the direct children at read time and enumerates nothing.

### What this deliberately does not change

Membership and blocking stay separate, which is the part of the current design worth keeping. `parent` says what an epic contains. `dependencies` says what gates it. A child can belong to an epic without blocking it, and `blocks_on: listed` keeps that behaviour exactly.

Many-to-many needs no new field. `dependencies` is a list on the epic and any number of epics can name the same child, so `deps --dependents CHILD` already answers what a ticket is part of. Storing membership on the child would add a field to buy a query that exists.

Milestones stay out of it. A milestone string is a shipping event with no acceptance criteria that nobody claims. An epic is a deliverable with a definition of done that somebody could claim. Nothing links the string `v1.2` to a ticket titled `v1.2`, because linking them by name would turn a string into a reference and reintroduce the registry problem the advisory allowlist in 5.1 exists to paper over.

### Derived, not stored

The blocking set for `blocks_on: children` is computed at read time and never written to a file, like `readiness`, `revision`, and `path`. `readinessOf` already receives the whole store, so this is a second pass over tickets already in memory rather than a new read. At the scale section 8 targets, hundreds to a few thousand, that is not a new complexity class. The plan should say so rather than leave a reader to assume it is free per ticket.

### Decision 1: the shape

Option A, `blocks_on: none | listed | children` as above. One enum on the epic, no new edge kind, no hot file.

Option B, no field at all. The epic keeps enumerating in `dependencies` and the store accepts the hot file. Zero format change, and the merge pain is real but bounded by how often an epic gains a child.

### Decision 2: an epic with blocks_on children and no children

Option A, it is unblocked and therefore ready. Consistent with an empty dependency list, and wrong in practice: `ready` then offers an agent a ticket titled `v1.2`, and an agent claims it and starts working on a release.

Option B, it is blocked. Strict, and defensible, because an epic that gates on children it does not have is not startable.

Option A plus a `check` warning is the middle and is probably right. The state is nearly always a decomposition somebody has not written yet, which is a store observation rather than a readiness verdict.

### Decision 3: where children appear in readiness

`Readiness` carries `Ready`, `Blocked`, `Blocking`, and `Missing`, and its doc comment says blocked is about dependencies alone. That comment stops being true.

Option A, children go into `Blocking` beside dependencies. Less for a consumer to learn, and the distinction between the two edge kinds is lost.

Option B, a separate field. Keeps "blocked means dependencies" true and lets a board draw the two differently, at the cost of one more field in the JSON contract under 12.4.

### deps stays pure

Section 8 keeps `deps` to `dependencies` and nothing else, and when the answer is empty on an epic the human output points at `list --parent`. Once children actually block, `deps` reports nothing about a ticket that is genuinely blocked, so that pointer stops being a courtesy and becomes a gap.

The fix belongs in `readiness`, which already carries what stands in the way, and not in `deps`. Mixing two edge kinds into one dependency walk is what section 8 declined on purpose.

### A cycle can now alternate edge kinds

`parent_cycle` and dependency cycles are checked separately, which is correct while the two edges point in opposite directions. An epic blocking on its children, plus a child depending on that epic, is a real cycle that neither check sees, because each edge kind alone is acyclic.

That finding has to exist before `blocks_on: children` ships. It is the one item here that is a correctness problem rather than a design preference.

### Not a directory

Epics stay in `tickets/` with everything else. The store partitions on status or not at all, per the decision recorded on TKT-01M1HVMQ.

## Acceptance criteria

- [ ] All three decisions in the description are settled and recorded in docs/plan.md before any code lands.
- [ ] blocks_on defaults to none, so every existing ticket and every fixture is unchanged.
- [ ] The blocking set for blocks_on children is derived at read time and never stored.
- [ ] check reports a cycle that alternates parent and dependency edges, which neither existing cycle check sees.
- [ ] deps still walks dependencies alone, and the plan records why children are not in it.
- [ ] A corpus fixture covers an epic blocking on children, including the empty-children case.
