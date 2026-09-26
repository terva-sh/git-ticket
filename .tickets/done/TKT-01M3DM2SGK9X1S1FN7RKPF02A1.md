---
schema: 2
id: TKT-01M3DM2SGK9X1S1FN7RKPF02A1
title: Implement portable reference registry and offline validation
type: task
status: done
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

Implement the portable reference registry and offline validation designed in plan 5.5 and 11. Keep .tickets/references.yml separate from config.yml, parse and render version 1 without losing unknown bytes through config rewrites, and use ignored references.local.yml only for optional local checkout resolution. Expose a library resolver that returns a URL or local target while preserving the stored ref. No network access belongs in check. The registry does not change ticket schema.

This is the core required before export can offer or import can adopt mappings. Include the measured legacy values ticket:report and abbreviated origin-ticket: in compatibility tests; they remain valid when undeclared. A declared bad identifier and malformed registry need their distinct published check codes.

## Acceptance criteria

- [x] Versioned tracked registry and ignored local overrides parse and render without config.yml loss
- [x] check reports invalid registry and declared identifiers offline; undeclared legacy references stay valid
- [x] Library resolver returns safe URL or local target and leaves stored reference bytes intact

## Implementation plan

Define a version-1 tracked references.yml parser and deterministic renderer, with strict namespace, store and template validation. Resolve declared references offline to an HTTPS browse URL or a safe local checkout target; keep undeclared references opaque and unchanged. Read ignored references.local.yml only when resolving, while check validates the tracked registry and declared identifiers. Add fixture-backed finding codes, compatibility tests for legacy refs, and a binary smoke run.

## Notes

**agent:codex/reference-registry** at 2026-09-26T02:17:48Z

Implementation decision: Keep the portable version-1 registry in references.yml, independent of config.yml, because older binaries rewrite config and would erase an unfamiliar references key. Keep checkout bindings only in ignored references.local.yml; check reads tracked declarations only, while the resolver reads local bindings on demand and falls back to a portable HTTPS URL. Validate declared identifiers offline; leave undeclared ticket:report and abbreviated origin-ticket: references opaque because imposing a built-in ticket grammar would invalidate existing data. Require literal HTTPS navigation hosts, complete path/query placeholders and whole id capture for ticket-store mappings; reject traversal and unsafe local paths. Full just ci passed, including race tests and strict store check. Built-binary scratch run proved valid registry, distinct reference_identifier_invalid and reference_registry_invalid findings, and that Git ignores malformed local bindings while check remains clean.

**agent:codex/reference-registry** at 2026-09-26T02:20:59Z

PR #226 targeted review run #588 reviewed fc1dbbc against cb5c790 and found that percent-encoded URL template braces passed validation but were not expanded by the resolver. Accepted and fixed in eb5fc48: reject encoded braces before URL parsing, test URL path and browse query cases, and state the literal-brace rule in plan 5.5. Targeted tests and full just ci passed. Requesting a fresh review of the substantive fix.

**agent:codex/reference-registry** at 2026-09-26T02:23:42Z

PR #226 review run #591 on abdaa83 confirmed the encoded-placeholder fix and found a tracked-registry symlink could make check depend on an outside target or silently treat a dangling link as absence. Accepted and fixed in 9ba240e: Lstat requires references.yml to be a regular file before reading; regression tests cover live and dangling links. Full just ci passed. Requesting a fresh review.

**agent:codex/reference-registry** at 2026-09-26T02:26:55Z

PR #226 review run #593 on 6d107d1 confirmed the symlink fix and found that a later .gitignore negation could expose references.local.yml in an adopted store. Accepted and fixed in a4555d8: initialization appends a final positive ignore rule unless it is already last; regression test covers a later !*.yml. Full just ci passed. Requesting a fresh review.

**agent:codex/reference-registry** at 2026-09-26T02:28:39Z

PR #226 clean targeted review: head 480b375aa562eecc86de9416f4ad0ad74158c2ff, base cb5c790079691414000261846dc4c504c5ce60cc; request local-binding-ignore-fix, Actions run #596 (id 11669), clean run 159c037a-47a0-4f6f-9e38-c753709d75d6. Reviewer confirmed earlier findings resolved. CI on that head passed. The only subsequent commit is this ticket record; carry the review status to it. PR remains open pending merge authorization.

## Summary

Portable reference registry and offline validation landed through PR #226. Strict store validation and the reference registry tests passed; the reviewed branch is on main.
