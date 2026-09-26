---
schema: 2
id: TKT-01M3CV8RASXJ4PPHD7N78T7XHK
title: Decide what a move does to dependents left in the sending store
type: spike
status: in-progress
status_reason: null
priority: normal
due_on: null
labels:
  - area/integration
  - question
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references:
  - ref: ticket:TKT-01M3CV8R9J3QXQSN07B9VAGV7V
    path: null
claim:
  actor: agent:codex/reference-design
  branch: feat/reference-namespace-design
  worktree: /home/sothr/.t3/worktrees/git-ticket/file-reference-questions
  commit: dc64d4fdc67b4440851b1cf86c083dd18450e16d
  claimed_at: 2026-09-26T01:07:36Z
  expires_at: null
archive: null
created_at: 2026-09-25T17:54:32Z
updated_at: 2026-09-26T01:07:47Z
created_by:
  id: agent:claude-code/opus
  name: ""
updated_by:
  id: agent:codex/reference-design
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

## Acceptance criteria

- [ ] A move records a machine-readable destination on the original ticket through a source-side action; the receiving import does not write the sending store.
- [ ] The sending store identifies each open dependent of a moved ticket and gives it a distinct finding; readiness does not treat the moved original as satisfying that dependency before manual resolution.
- [ ] A person can resolve each dependent after checking the receiving work, with the destination and the reason preserved in the record.
- [ ] The design uses existing deps --dependents where it fits and specifies the marker, finding severity, readiness rule, and valid lifecycle commands for draft, ready, and in-progress origins.
- [ ] Plan sections 6.3 and 12.8 record the offline source-side workflow; a two-store run proves no premature ready result and check reports unresolved dependents.

## Implementation plan

Reproduce the sending-store failure in two scratch stores; specify a schema-gated moved_to reference, a source-side move command, per-dependent check/readiness behavior, and an explicit manual resolution command. Record valid lifecycle advice in plan 6.3 and 12.8, then file implementation work.

## Notes

**agent:codex/reference-groom** at 2026-09-26T00:47:03Z

Groomed. The trigger fired in the reported service-reconciliation move. I reproduced both local outcomes in a scratch store: marking the original done makes its ready dependent ready; archiving the original before done leaves that dependent waiting. I also reproduced an interim safe path: keep the original blocked with a destination reference and reason, and the dependent stays unready while check --strict passes. The existing deps ID --dependents query already lists affected tickets.

The user chose a source-side flag and manual resolution as the first work. The design should record a machine-readable move destination on the original, report each local dependent distinctly, and keep it unready until a person resolves its dependency after checking the receiving work. Automatic cross-store readiness is deferred: it would need trusted store binding, availability rules, and a definition of which remote status satisfies a local obligation. Advice alone loses the local gate and was rejected for this scope.

Two corrections to the filed options matter. Import only receives the ticket patch and a plain FromStore name; it cannot enumerate the sender's dependents itself, so option 1 as written needs a source-side query or extra export data. The source query already exists as deps --dependents. The current close-at-origin advice prints status ID done for every nonfinished source. That fails for draft without a reason and for ready or blocked because the transition is forbidden; I ran both draft and ready cases. The chosen first work must replace that advice with valid lifecycle steps. A new source-side marker and finding are preferable to relying on blocked status alone because every dependent then gets an explicit explanation of where its gate went. The area label now leads, matching this store's convention.

**agent:codex/reference-groom** at 2026-09-26T00:50:57Z

PR #222 review recorded. The targeted Terva review ran on head 3bca1d6eaa0be750be136a6e90f537743876473c against base adbcaa2452744a07efa706df2eebc7f50de96ef0, request groomed-ready-3bca1d6, run ce377a7a-4b8b-41f7-ab4e-de097aef61c6: https://git.local.sothr.com/terva-sh/git-ticket/actions/runs/566. The maintained clean summary is on https://git.local.sothr.com/terva-sh/git-ticket/pulls/222. It reported no findings, so there is no review change to accept, dispute, or defer. It saw only the ticket diff and could not verify runtime behavior; the local scratch-store runs and full just ci result are the evidence for those claims. This note records the result without changing either design scope.

**agent:codex/reference-design** at 2026-09-26T01:07:47Z

Design decision for plan 6.3, 11, 12.5, and 12.8: schema 4 adds moved_to as a complete typed reference on the original ticket. A source-side move command writes it from any status and records a reason; receiving import only prints advice. A moved original is excluded from ready and cannot satisfy an in-store dependency even if it is done or archived from done. check reports dependency_moved as an error on every open dependent edge, naming the destination. The manual resolve-move command removes one such edge, carries the destination as a reference on the dependent, records the human reason, and can add a local replacement dependency. It never polls a foreign store. Schema 4 is required because an older reader that ignored a marker on a done original would recommend the dependent too early.

Scratch evidence in two stores: source TKT-01M3DKYVKAPMSR1H1WF9WJ9WWS (Remote prerequisite) had dependent TKT-01M3DKYVKV6TY379E5C9G27835 (Local dependent). It was exported and adopted in the receiver as TKT-01M3DKYV4G4ZKCKRC8RYEP42JY (Remote prerequisite). deps --dependents named the local dependent. The current import advice printed status SOURCE done, which failed from ready with invalid_transition. After a valid ready -> in-progress -> done path, ready incorrectly listed Local dependent. Reopening with a reason, blocking the origin, and adding a destination-ticket: reference removed that premature ready result; check --strict still said No problems found. Manually unlinking the old edge, adding the destination reference on the dependent, and writing a reason restored ready and left check --strict clean. These are baseline and interim-behavior measurements; the new dependency_moved finding needs an implementation test.

Alternatives rejected: advice alone names the problem but cannot gate readiness; automatic remote status polling conflicts with offline check and needs a trust and availability model; a bare reference has no safety meaning to old readers. A blocked status alone works as an interim hold but does not say which dependent requires resolution, and a later done transition can release it early. The new source-side command replaces the invalid universal close-at-origin advice. The plan states the lifecycle, finding, and manual resolution contract. Implementation is separate work.
