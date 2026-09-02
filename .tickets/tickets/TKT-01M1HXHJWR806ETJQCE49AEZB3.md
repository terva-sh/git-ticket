---
schema: 1
id: TKT-01M1HXHJWR806ETJQCE49AEZB3
title: Let an epic block on its children without enumerating them
type: task
status: ready
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
updated_at: 2026-09-02T20:50:00Z
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

All three decisions are settled and recorded at the end of plan section 15. What follows states the answers and their reasoning. The plan is the record.

### Settled: the shape is an enum, not a list

`blocks_on` takes `none`, `listed`, or `children`, and defaults to `none`.

`none` is today's behaviour, so every existing ticket and every fixture is unchanged. `listed` is today's enumerated `dependencies` list. `children` derives the blocking set from the direct children at read time and enumerates nothing.

Enumerating was the alternative and it loses on concurrency rather than on expressiveness. An epic that lists its children is edited by every decomposition. The enum moves that edit off the epic and onto the child, where `parent` already records the relationship and two additions are two separate files that merge cleanly.

### Settled: an epic with blocks_on children and no children is not blocked

`check` warns instead. The proposed code is `blocks_on_no_children`, a warning, and section 11 gains it in the commit that builds this, together with its corpus fixture.

Blocking it would put a ticket in the `blocked` state with nothing to name as the blocker. Section 8 already refuses that for a draft and for a ticket somebody else holds, on the grounds that it sends a reader looking for a dependency that is not there.

Status is the guard that matters here. A new ticket is `draft` and never reaches `ready`, so an undecomposed epic can only be offered as startable after somebody promotes it by hand. That is an authoring mistake, and a warning is the instrument for an authoring mistake.

### Settled: children get their own field in readiness

Not `blockingDependencies`. That field is published in 10.2 and versioned under 12.4, and a consumer rendering "waiting on" from it would print a child ID labelled as a dependency, with nothing to signal the difference.

A new field is additive, so a consumer that ignores it behaves exactly as it does today. The cost is that `Blocked` widens to cover both edge kinds, so a consumer showing `blockingDependencies` whenever `Blocked` is true prints an empty list for a children-blocked epic. Missing beats wrong, and widening `Blocked` is the right answer to "can this be started", which is the question that field exists to answer.

### What this deliberately does not change

Membership and blocking stay separate, which is the part of the current design worth keeping. `parent` says what an epic contains. `dependencies` says what gates it. A child can belong to an epic without blocking it, and `blocks_on: listed` keeps that behaviour exactly.

Many-to-many needs no new field. `dependencies` is a list on the epic and any number of epics can name the same child, so `deps --dependents CHILD` already answers what a ticket is part of. Storing membership on the child would add a field to buy a query that exists.

Milestones stay out of it. A milestone string is a shipping event with no acceptance criteria that nobody claims. An epic is a deliverable with a definition of done that somebody could claim. Nothing links the string `v1.2` to a ticket titled `v1.2`, because linking them by name would turn a string into a reference and reintroduce the registry problem the advisory allowlist in 5.1 exists to paper over.

### Derived, not stored

The blocking set for `blocks_on: children` is computed at read time and never written to a file, like `readiness`, `revision`, and `path`. `readinessOf` already receives the whole store, so this is a second pass over tickets already in memory rather than a new read. At the scale section 8 targets, hundreds to a few thousand, that is not a new complexity class. The plan should say so rather than leave a reader to assume it is free per ticket.

### deps stays pure

Section 8 keeps `deps` to `dependencies` and nothing else, and when the answer is empty on an epic the human output points at `list --parent`. Once children actually block, `deps` reports nothing about a ticket that is genuinely blocked, so that pointer stops being a courtesy and becomes a gap.

The fix belongs in `readiness`, which already carries what stands in the way, and not in `deps`. Mixing two edge kinds into one dependency walk is what section 8 declined on purpose.

### A cycle can now alternate edge kinds

`parent_cycle` and dependency cycles are checked separately, which is correct while the two edges point in opposite directions. An epic blocking on its children, plus a child depending on that epic, is a real cycle that neither check sees, because each edge kind alone is acyclic.

That finding has to exist before `blocks_on: children` ships. It is the one item here that is a correctness problem rather than a design preference, and it is the reason this ticket is larger than one field.

### Not a directory

Epics stay in `tickets/` with everything else. The store partitions on status or not at all, per the decision recorded on TKT-01M1HVMQQQE3K6VZG7793RXVXN.

## Acceptance criteria

- [x] All three decisions in the description are settled and recorded in docs/plan.md before any code lands.
- [ ] blocks_on defaults to none, so every existing ticket and every fixture is unchanged.
- [ ] The blocking set for blocks_on children is derived at read time and never stored.
- [ ] check reports a cycle that alternates parent and dependency edges, which neither existing cycle check sees.
- [ ] deps still walks dependencies alone, and the plan records why children are not in it.
- [ ] A corpus fixture covers an epic blocking on children, including the empty-children case.
