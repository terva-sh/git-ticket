---
schema: 2
id: TKT-01M3DM2SKQGX0QYC4GFX32NR53
title: Gate dependents when a prerequisite moves to another store
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - area/integration
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references:
  - ref: ticket:TKT-01M3CV8RASXJ4PPHD7N78T7XHK
    path: null
claim: null
archive: null
created_at: 2026-09-26T01:08:11Z
updated_at: 2026-09-26T03:09:03Z
created_by:
  id: agent:codex/reference-design
  name: ""
updated_by:
  id: agent:codex/landing
  name: ""
extensions: {}
---

## Description

Implement schema-4 moved_to and the source-side move/resolve-move workflow in plan 6.3, 11, 12.5, and 12.8. The original ticket carries one typed destination; readiness treats it as unsatisfied in any status and excludes the moved original. check reports dependency_moved for each open local dependent until that dependent is manually resolved. Move and resolve-move keep reasons and destinations in the ticket record. Replace import --same-owner's invalid universal status done advice with valid source-side instructions. The receiving import must never mutate the sender.

## Acceptance criteria

- [x] Schema 4 migration gates older readers before moved_to is written
- [x] Moved original never satisfies dependencies; each open dependent gets dependency_moved
- [x] move and resolve-move preserve destination and manual reason with revision safety
- [x] Two-store run proves no premature ready result and valid source-side advice

## Implementation plan

Add schema-4 moved_to parsing, rendering and explicit migration. Gate readiness and report each open dependent until a person resolves it. Provide move and resolve-move with locked source revision checks and durable reason notes; update import advice. Verify schema compatibility, failures, and a two-store workflow before proposing the change.

## Notes

**agent:codex/moved-dependent** at 2026-09-26T01:56:04Z

Implementation decision: Schema 4 is explicit because an older reader would otherwise count a moved done ticket as satisfying dependencies. The original keeps its status and a typed moved_to marker; changing status to done as a move signal loses the distinction and releases local dependents. Resolution requires the original revision checked under the store lock and a written reason; polling the receiving store would claim certainty this store cannot establish. Import advice leaves the receiver namespace for the owner to choose, because an invented ticket namespace could collide across stores. Verification: full just ci including race tests and strict store check passed; a built binary moved a done prerequisite between two scratch stores, kept its local dependent unready, reported dependency_moved, and cleared it only after resolve-move.

**agent:codex/moved-dependent** at 2026-09-26T01:59:32Z

PR #225 review: head 0e8f16cc36fec48a11c9db16945df1fc267b184b, base cb5c790079691414000261846dc4c504c5ce60cc; targeted review run #579 (id 11646, request ready-review), review 754. Accepted its medium finding: move --json omitted affected dependents. Commit 406237f adds affectedDependents with ID and title to every mutation-result (empty for other mutations), queries before emitting either output mode, documents the field, and tests the JSON path. Full just ci passed after the fix. Request a fresh review of this substantive change.

**agent:codex/moved-dependent** at 2026-09-26T01:59:37Z

Review references: PR https://git.local.sothr.com/terva-sh/git-ticket/pulls/225 ; run https://git.local.sothr.com/terva-sh/git-ticket/actions/runs/579 ; review https://git.local.sothr.com/api/v1/repos/terva-sh/git-ticket/pulls/225/reviews/754 .

**agent:codex/moved-dependent** at 2026-09-26T02:01:33Z

PR #225 second targeted review: head e7885d800738c4184782e1d09468990872179e5e, base cb5c790079691414000261846dc4c504c5ce60cc; review 755 at https://git.local.sothr.com/api/v1/repos/terva-sh/git-ticket/pulls/225/reviews/755 and run #581 at https://git.local.sothr.com/terva-sh/git-ticket/actions/runs/581 (request after-json-fix). It confirmed review 754 was resolved. Accepted its medium finding: a post-write dependent lookup could fail and make a committed move look failed. Commit ea65cba queries dependents before Apply, so a lookup failure leaves the marker untouched. The list is advisory if another writer changes edges after the snapshot; check remains authoritative. Full just ci passed after the fix.

**agent:codex/moved-dependent** at 2026-09-26T02:03:20Z

PR #225 clean targeted review: head 76a4d987b91c29d890a45d00284ff7de08cdff6b, base cb5c790079691414000261846dc4c504c5ce60cc; run 955a5943-7f3b-45e7-800f-0f80aa6e5f48, request before-write-fix, https://git.local.sothr.com/terva-sh/git-ticket/actions/runs/583 ; clean summary comment https://git.local.sothr.com/terva-sh/git-ticket/pulls/225#issuecomment-13879 . It marks review 755 finding resolved and reports no new concrete issue. Deterministic PR CI and local just ci passed. No merge has been performed.

**agent:codex/moved-dependent** at 2026-09-26T03:07:00Z

Landing integration: merged origin/main at 9d1332f9505f6073c6b8d16406ed84a2d5419637 into this branch without rewriting its published commits. The first merge resolved two overlapping declarations: docs/plan.md retains both the moved-dependency and reference-registry finding rows, and ticket/errors.go retains both pairs of codes. It also corrected the plan's now-obsolete wording that called both implemented code pairs future. The subsequent reader merge was conflict-free. Full just ci passed on combined head e25e90bb4911cdc046b44c18f810b53050558f45, including vet, race suite, and strict ticket-store check. Requesting a fresh targeted review against the combined main before merge.

**agent:codex/moved-dependent** at 2026-09-26T03:08:09Z

PR #225 targeted review is clean on combined head ca4aa2755595ac9304713b33c6d3cb2717b588eb, base 9d1332f9505f6073c6b8d16406ed84a2d5419637 (request combined-main-review; Actions run #633, id 11724; clean summary comment 13879). Review 755's post-write lookup finding remains resolved. Full local just ci passed on the integrated code. Recording this ticket-only review result and carrying the status to the bookkeeping head before merge.

## Summary

Schema-4 moved prerequisite gating, manual resolution, and corrected import advice are implemented in PR #225. Full CI and targeted review passed against the combined main; this branch carries the completed ticket into the merge.
