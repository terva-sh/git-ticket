---
schema: 2
id: TKT-01M3CV8RASXJ4PPHD7N78T7XHK
title: Decide what a move does to dependents left in the sending store
type: spike
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - question
  - area/integration
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references:
  - ref: ticket:TKT-01M3CV8R9J3QXQSN07B9VAGV7V
    path: null
claim: null
archive: null
created_at: 2026-09-25T17:54:32Z
updated_at: 2026-09-25T17:54:32Z
created_by:
  id: agent:claude-code/opus
  name: ""
updated_by:
  id: agent:claude-code/opus
  name: ""
extensions: {}
---

## Description

Plan 12.8 settles how edges behave on the receiving side of a move. Edges among the exported tickets are rewritten to the new IDs, a left-behind parent becomes an `origin-parent:` reference under `--same-owner`, and a dependency pointing outside the export is dropped and reported. It says nothing about the sending side, where a ticket that stays behind can depend on a ticket that moved.

### What happened

On 2026-09-25 a service-reconciliation ticket moved from the workspace ledger to the store in Sothr-Infrastructure/documentation. A DNS-cleanup ticket stayed in the ledger and depended on it. The close-at-origin advice leaves the original ticket in some final state, and whichever state is chosen, the dependent goes wrong:

- `done`: the dependency reads as satisfied, and the dependent enters `ready` while the real work has not started in the other store.
- `archived` out of anything but `done`: `dependency_archived_incomplete`, a warning, and the dependent is blocked for good with nothing pointing at the ticket that actually gates it.

The workaround was manual: unlink the dependency, add an unverified reference to the moved ticket's new ID, and write a note saying "do not start until that one is done". Nothing enforces the note.

### Options

1. The close-at-origin advice from `import --same-owner` also names every dependent at the origin and prints the commands that relink each one, so the person who owns both stores sees the edge before it goes wrong.
2. A sending-side reference form parallel to `origin-parent:`, for example `moved-to:<store>:<ID>` on the original, together with a rule that a dependency on a moved ticket reports `dependency_moved` and points at the new location instead of passing or blocking silently.
3. A cross-store wait as a first-class, unverified edge (`waits-on:` a reference), gating `ready` only when the other store is present to read. That runs against 5.6's "one store and not many", so it probably needs the store bindings from the reference-namespaces question first.
4. Document it and do nothing: a dependency across stores is a reference plus a note, by convention.

Option 1 is cheap and fits 12.8's "advice and never action". Option 2 or 3 is what closes the gap. The choice is the maintainer's.

Related: TKT-01M3CV8R9J3QXQSN07B9VAGV7V (Decide how a store declares where each reference namespace resolves), and TKT-01M3CT0GV02WDW2W7B3EJSC6R1 (Default series stays TKT after a store stops declaring it).
