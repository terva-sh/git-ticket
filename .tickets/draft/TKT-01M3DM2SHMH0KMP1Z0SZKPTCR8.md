---
schema: 2
id: TKT-01M3DM2SHMH0KMP1Z0SZKPTCR8
title: Carry reference lookup mappings through export and import
type: task
status: draft
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
updated_at: 2026-09-26T01:08:19Z
created_by:
  id: agent:codex/reference-design
  name: ""
updated_by:
  id: agent:codex/reference-design
  name: ""
extensions: {}
---

## Description

Implement the reference lookup sidecar and explicit receiver choices from plan 12.8. Export writes references.json with only used portable mappings and a SHA-256 binding to 0001-tickets.patch; git am DIR/*.patch stays valid. Import previews the offered mapping, verifies the digest before any write, and lets the receiver adopt, alias, or decline each. A conflicting local meaning must never silently win. Mapping adoption is an atomic locked registry write and is reported separately from partial ticket adoption. Preserve old two-file exports.

## Acceptance criteria

- [ ] Export sidecar contains used portable mappings and exact ticket-patch digest; git am glob still applies
- [ ] Import preview verifies sidecar and shows adopt, alias, and decline outcomes, including conflicts
- [ ] Explicit mapping write is atomic; partial ticket filing reports both filed tickets and mapping outcome
- [ ] Older two-file exports remain importable and undeclared references remain opaque
