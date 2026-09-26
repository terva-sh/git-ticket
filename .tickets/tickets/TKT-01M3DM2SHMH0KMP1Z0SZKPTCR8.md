---
schema: 2
id: TKT-01M3DM2SHMH0KMP1Z0SZKPTCR8
title: Carry reference lookup mappings through export and import
type: task
status: review
status_reason: null
priority: normal
due_on: null
labels:
  - area/integration
assignees: []
milestone: null
parent: null
origin: null
dependencies:
  - TKT-01M3DM2SGK9X1S1FN7RKPF02A1
blocks_on: none
references:
  - ref: ticket:TKT-01M3CV8R9J3QXQSN07B9VAGV7V
    path: null
claim:
  actor: agent:codex/reference-lookup
  branch: feat/reference-lookup-interchange
  worktree: /home/sothr/.t3/worktrees/git-ticket/reference-lookup-interchange
  commit: abdaa83986a75838757464f8c06ed5abfc483f11
  claimed_at: 2026-09-26T02:21:53Z
  expires_at: null
archive: null
created_at: 2026-09-26T01:08:11Z
updated_at: 2026-09-26T02:37:55Z
created_by:
  id: agent:codex/reference-design
  name: ""
updated_by:
  id: agent:codex/reference-lookup
  name: ""
extensions: {}
---

## Description

Implement the reference lookup sidecar and explicit receiver choices from plan 12.8. Export writes references.json with only used portable mappings and a SHA-256 binding to 0001-tickets.patch; git am DIR/*.patch stays valid. Import previews the offered mapping, verifies the digest before any write, and lets the receiver adopt, alias, or decline each. A conflicting local meaning must never silently win. Mapping adoption is an atomic locked registry write and is reported separately from partial ticket adoption. Preserve old two-file exports.

## Acceptance criteria

- [x] Export sidecar contains used portable mappings and exact ticket-patch digest; git am glob still applies
- [x] Import preview verifies sidecar and shows adopt, alias, and decline outcomes, including conflicts
- [x] Explicit mapping write is atomic; partial ticket filing reports both filed tickets and mapping outcome
- [x] Older two-file exports remain importable and undeclared references remain opaque

## Implementation plan

Build a versioned JSON sidecar from used namespaces, bind it to the exact ticket patch, and verify all declarations and namespace use before import. Plan receiver choices separately from ticket adoption, with explicit alias and opaque rewrites. Write accepted registry mappings under the store lock with a revision precondition, report that write separately from partial ticket filing, and preserve old exports. Exercise both CLI routes and git am.

## Notes

**agent:codex/reference-lookup** at 2026-09-26T02:37:24Z

Implemented references.json as a versioned sidecar bound to the exact ticket patch; only used declared namespaces and their stores travel, with undeclared names listed. A sidecar beats patching references.yml because git am must leave receiver mapping adoption as a separate choice. The receiver defaults to decline; adopt and alias require explicit --map and --adopt-mappings, while conflicting or undeclared names that match a local declaration require an opaque decline target before ticket filing. Silent reuse of a local name was rejected because it could point at unrelated work. The mapping write uses the store lock, an atomic replacement and a registry revision precondition. Preview and write share the same plan; ticket filing keeps partial result IDs and reports the earlier mapping outcome separately. Old exports still import, with collisions treated as unknown destinations. Full just ci passed after integration with the reviewed registry branch. Built-binary two-store run passed sidecar export, alias adoption and check --strict; tests cover git am, digest tampering, conflicts, old exports, store-key aliasing, foreign-ticket provenance and partial filing.

**agent:codex/reference-lookup** at 2026-09-26T02:37:55Z

PR #227 is open against feat/reference-registry (PR #226), head 4f74fb0223c9c84e8fa369f99f36c0cd7a3deb21 before ticket-only review bookkeeping. Branch contains the reviewed registry fixes and is clean. Full just ci and a built-binary source-to-receiver alias adoption run passed. Awaiting targeted review and PR CI; both PRs remain open without merge authorization.
