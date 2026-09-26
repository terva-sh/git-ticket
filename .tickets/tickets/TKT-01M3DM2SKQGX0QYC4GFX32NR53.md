---
schema: 2
id: TKT-01M3DM2SKQGX0QYC4GFX32NR53
title: Gate dependents when a prerequisite moves to another store
type: task
status: in-progress
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
claim:
  actor: agent:codex/moved-dependent
  branch: feat/moved-dependent-gate
  worktree: /home/sothr/.t3/worktrees/git-ticket/moved-dependent-gate
  commit: cb5c790079691414000261846dc4c504c5ce60cc
  claimed_at: 2026-09-26T01:46:13Z
  expires_at: null
archive: null
created_at: 2026-09-26T01:08:11Z
updated_at: 2026-09-26T01:48:01Z
created_by:
  id: agent:codex/reference-design
  name: ""
updated_by:
  id: agent:codex/moved-dependent
  name: ""
extensions: {}
---

## Description

Implement schema-4 moved_to and the source-side move/resolve-move workflow in plan 6.3, 11, 12.5, and 12.8. The original ticket carries one typed destination; readiness treats it as unsatisfied in any status and excludes the moved original. check reports dependency_moved for each open local dependent until that dependent is manually resolved. Move and resolve-move keep reasons and destinations in the ticket record. Replace import --same-owner's invalid universal status done advice with valid source-side instructions. The receiving import must never mutate the sender.

## Acceptance criteria

- [ ] Schema 4 migration gates older readers before moved_to is written
- [ ] Moved original never satisfies dependencies; each open dependent gets dependency_moved
- [ ] move and resolve-move preserve destination and manual reason with revision safety
- [ ] Two-store run proves no premature ready result and valid source-side advice

## Implementation plan

Add schema-4 moved_to parsing, rendering and explicit migration. Gate readiness and report each open dependent until a person resolves it. Provide move and resolve-move with locked source revision checks and durable reason notes; update import advice. Verify schema compatibility, failures, and a two-store workflow before proposing the change.
