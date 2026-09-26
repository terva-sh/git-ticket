---
schema: 2
id: TKT-01M3DM2SGK9X1S1FN7RKPF02A1
title: Implement portable reference registry and offline validation
type: task
status: in-progress
status_reason: null
priority: normal
due_on: null
labels:
  - area/format
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
  actor: agent:codex/reference-registry
  branch: feat/reference-registry
  worktree: /home/sothr/.t3/worktrees/git-ticket/reference-registry
  commit: cb5c790079691414000261846dc4c504c5ce60cc
  claimed_at: 2026-09-26T02:04:34Z
  expires_at: null
archive: null
created_at: 2026-09-26T01:08:11Z
updated_at: 2026-09-26T02:04:45Z
created_by:
  id: agent:codex/reference-design
  name: ""
updated_by:
  id: agent:codex/reference-registry
  name: ""
extensions: {}
---

## Description

Implement the portable reference registry and offline validation designed in plan 5.5 and 11. Keep .tickets/references.yml separate from config.yml, parse and render version 1 without losing unknown bytes through config rewrites, and use ignored references.local.yml only for optional local checkout resolution. Expose a library resolver that returns a URL or local target while preserving the stored ref. No network access belongs in check. The registry does not change ticket schema.

This is the core required before export can offer or import can adopt mappings. Include the measured legacy values ticket:report and abbreviated origin-ticket: in compatibility tests; they remain valid when undeclared. A declared bad identifier and malformed registry need their distinct published check codes.

## Acceptance criteria

- [ ] Versioned tracked registry and ignored local overrides parse and render without config.yml loss
- [ ] check reports invalid registry and declared identifiers offline; undeclared legacy references stay valid
- [ ] Library resolver returns safe URL or local target and leaves stored reference bytes intact

## Implementation plan

Define a version-1 tracked references.yml parser and deterministic renderer, with strict namespace, store and template validation. Resolve declared references offline to an HTTPS browse URL or a safe local checkout target; keep undeclared references opaque and unchanged. Read ignored references.local.yml only when resolving, while check validates the tracked registry and declared identifiers. Add fixture-backed finding codes, compatibility tests for legacy refs, and a binary smoke run.
