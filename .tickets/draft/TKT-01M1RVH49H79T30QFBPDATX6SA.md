---
schema: 1
id: TKT-01M1RVH49H79T30QFBPDATX6SA
title: Let a ticket arrive done, from create and from draft
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-05T13:18:36Z
updated_at: 2026-09-05T13:18:36Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Filed 2026-09-05 from a user ask: work sometimes arrives already
finished, and the lifecycle has no honest way to say so. Two cases,
one mechanism.

Backporting from another system means filing tickets for work that is
done, so the new store carries the project's history and its
dependency graph from day one. Today `create` always files a draft and
nothing else, so a backport walks every ticket through
draft-ready-in-progress-done, five writes each, all of them fiction.
The Backlog.md importer (TKT-01M1F7Z30) has exactly this need for
every finished task it imports, so building this first makes the
importer smaller.

The second case is a draft that should close: the work turned out to
be already done, or was decided against. The decided-against half
already has its path, draft to archived with a reason, which 6.2
permits today. The already-done half does not, and the difference is
not cosmetic. Plan 6.3 satisfies a dependency with a ticket in done or
archived out of done, and a draft archived directly has from_status
draft, so it satisfies nothing. Marking a draft done says the work
exists; archiving it says the work will not happen. Those are
different claims and the store should be able to record either.

The shape: `create --status done` files directly into done/, and the
6.2 table gains done as a permitted target from draft. The TUI status
picker follows for free, because it offers ticket.PermittedTransitions
and nothing else. statusDir already maps done to done/, so no store
change is needed, and check has nothing new to learn.

### Open questions

Whether create accepts only done or any status. The convention that
everything lands in draft and promotion is a human call is
load-bearing in this store, and `create --status ready` would quietly
bypass it. The narrow form, done only, covers both real cases and
keeps the gate; the general form should need its own argument.

Whether draft to done wants a required reason or a recommended
summary. 6.2 requires a reason into blocked and out of done, both
places a later reader asks why. A draft that jumps to done raises the
same question, and a summary saying where the work actually happened
may be the honest artifact.

What a backported ticket's timestamps mean. The ULID stamps creation
at import time, not when the work happened. Whether that is fine, or
whether create grows a --created override for backports, is worth
deciding before the first big import rather than after.

### Trigger

None. Startable feature work, parked as a draft until promoted. The
Backlog.md importer firing would promote it implicitly, since the
importer needs it.

## Acceptance criteria

- [ ] create --status done files directly into done/, per an amended plan
- [ ] the 6.2 table permits draft to done, and the TUI status picker offers it via PermittedTransitions
- [ ] a dependency on a ticket that arrived done counts as satisfied, with a test
- [ ] the plan records which statuses create accepts, and why the draft gate survives
