---
schema: 1
id: TKT-01M1RVH49H79T30QFBPDATX6SA
title: Let a ticket arrive done, from create and from draft
type: task
status: done
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
claim:
  actor: agent:terva/mieli
  branch: feat/create-done
  worktree: /home/sothr/workspace/git.local.sothr.com/terva-sh/git-ticket
  commit: a43bb6379b993603fd9cf290dcc4b6c568be723c
  claimed_at: 2026-09-05T13:31:17Z
  expires_at: null
archive: null
created_at: 2026-09-05T13:18:36Z
updated_at: 2026-09-05T13:37:29Z
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

- [x] create --status done files directly into done/, per an amended plan
- [x] the 6.2 table permits draft to done, and the TUI status picker offers it via PermittedTransitions
- [x] a dependency on a ticket that arrived done counts as satisfied, with a test
- [x] the plan records which statuses create accepts, and why the draft gate survives

## Notes

**agent:terva/mieli** at 2026-09-05T13:37:29Z

Built and all four criteria are ticked. The three open questions were
settled with the user before code: --status accepts done and archived,
draft to done requires a --reason, and --created backdates the ULID.

The plan moved first: 6.2's draft row gains done, the reason rule
covers closing a draft as done, and the new 6.2.1 is the design
section, including why the archive path keeps from_status draft, so an
abandoned backport satisfies no dependency, which
TestCreateArrivesArchived holds.

Two mechanics worth recording. --created replaces the create's clock
wholesale, so the ULID time part, created_at, and updated_at all agree
with the backdated instant, and TestCreateBackdates asserts the ID
sorts before a fresh one, which is the property the flag exists for. A
future instant is refused.

The reason rules moved into an exported ticket.ReasonRequired, and the
status picker now asks it instead of hardcoding blocked. That fixed a
pre-existing gap this ticket did not name: reopening done to
in-progress through the picker used to apply bare and bounce after the
fact. TestStatusPickerReasonRequiredTransitionsAsk covers both pairs.

TestArrivedStoreIsClean proves the store integrity claim: a store
holding a created-done and a created-archived ticket raises no check
finding, because writeTicket routes by status and the archive block is
complete at birth.

## Summary

Shipped, per the new plan 6.2.1. `create --status done` and `--status
archived` file a backported ticket where it belongs, `--created`
backdates the ULID so imported history sorts chronologically, and a
draft can close straight to done with a required --reason. An
archived-at-create ticket keeps from_status draft and satisfies no
dependency, which is the semantic line between "the work exists" and
"the work will not happen". ticket.ReasonRequired is now the one list
of reason-requiring transitions, shared by SetStatus and the TUI
picker, which also fixed the picker's silent bounce on reopening from
done.
