---
schema: 2
id: TKT-01M3DM2SHMH0KMP1Z0SZKPTCR8
title: Carry reference lookup mappings through export and import
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
dependencies:
  - TKT-01M3DM2SGK9X1S1FN7RKPF02A1
blocks_on: none
references:
  - ref: ticket:TKT-01M3CV8R9J3QXQSN07B9VAGV7V
    path: null
claim: null
archive: null
created_at: 2026-09-26T01:08:11Z
updated_at: 2026-09-26T03:09:02Z
created_by:
  id: agent:codex/reference-design
  name: ""
updated_by:
  id: agent:codex/landing
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

**agent:codex/reference-lookup** at 2026-09-26T02:42:23Z

PR #227 review 759 (head 4e94a2238bf48ef38370546c5387ddb09dcf09cc, base 3567f3197260cb0dc9e413b0e990e56e67e7e8ad; Actions run #602, id 11683) raised a medium finding that explicit decline of an identical mapping still resolves. Disposition: the design record in plan 12.8 explicitly requires that behavior, because decline copies no mapping and cannot disable an identical local declaration. Follow-up review 760 (discussion comment 13912; Actions run #604, id 11685) agreed the finding does not apply. Commit ae35f84 adds a regression test and preview wording to make that consequence visible. Targeted tests passed; requesting a fresh full review of the clarification.

**agent:codex/reference-lookup** at 2026-09-26T02:49:33Z

PR #227 review 761 on c7a2eb0 (Actions run #605, id 11686) confirmed the identical-decline finding resolved and identified a high race: the mapping planner parsed references.yml, then read its revision from a second file snapshot. Accepted. Commit 48aa497 makes ReadReferenceRegistry and planning share readReferenceRegistrySnapshot, parsing and hashing the same bytes; a concurrent atomic-replacement test checks that declarations and revision always pair. Full just ci passed after the code fix; the targeted concurrency test passed under the race detector. Requesting a fresh targeted review.

**agent:codex/reference-lookup** at 2026-09-26T02:55:03Z

PR #227 review 762 identified that ticket adoption could use a registry changed after planning even when no mapping write was requested. Accepted. ImportPlan now records the registry revision; ApplyImport checks it under a single store lock and holds the lock through ticket filing. The CLI checks that mapping and ticket plans used the same registry snapshot and advances the ticket plan only after its own accepted mapping write. Choosing only a mapping-write precondition was insufficient for ticket-only imports; re-reading without the lock would leave the same race. A regression test changes references.yml after planning and verifies stale_revision before any ticket lands; another files a ticket after adopting mappings. Full just ci passed, including the race suite and strict store check.

**agent:codex/reference-lookup** at 2026-09-26T02:57:45Z

PR #227 review 763 (head 38d6bac, base 3567f31; Actions run #610, id 11695) confirmed review 762's registry race resolved and raised a low finding: mapping-only adoption printed the pre-write receiver state as though it were current. Accepted. The mapping-only preview now labels the old state as before the mapping write and reports the installed namespace after the write. Replanning after an alias write was rejected because the selected alias would then collide with the newly installed namespace. A CLI regression test checks both before and after wording. Full just ci passed. Requesting fresh targeted review.

**agent:codex/reference-lookup** at 2026-09-26T03:00:28Z

PR #227 review 766 (head 385ba5d, base 3567f31; Actions run #616, id 11701) confirmed review 763's stale-preview finding resolved and raised a low finding for an unchanged identical mapping reported as newly installed. Accepted. The preview now uses ApplyReferenceMappings.Result.Changed rather than the --adopt-mappings request to say whether a write happened, and it reports installation only for a namespace previously absent or explicitly aliased. A regression test runs mapping-only adoption twice and checks that the second, unchanged run does not claim installation. Full just ci passed. Requesting fresh targeted review.

**agent:codex/reference-lookup** at 2026-09-26T03:02:03Z

PR #227 targeted review is clean on head 6778e55252db4c826809cb39dd5d070f68b581d4, base 3567f3197260cb0dc9e413b0e990e56e67e7e8ad (request unchanged-mapping-review; Actions run #622, id 11713; clean summary comment 13948). Review 766's unchanged-mapping finding is resolved. Full local just ci passed on the reviewed code. Recording this ticket-only result and carrying the review status to the bookkeeping head; review success is not merge authorization.

## Summary

Reference lookup sidecar, explicit receiver mapping choices, and registry revision protection landed through PR #227. Full CI and targeted review passed on the merged code.
