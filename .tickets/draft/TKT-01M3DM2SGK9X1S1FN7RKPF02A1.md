---
schema: 2
id: TKT-01M3DM2SGK9X1S1FN7RKPF02A1
title: Implement portable reference registry and offline validation
type: task
status: draft
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

Implement the portable reference registry and offline validation designed in plan 5.5 and 11. Keep .tickets/references.yml separate from config.yml, parse and render version 1 without losing unknown bytes through config rewrites, and use ignored references.local.yml only for optional local checkout resolution. Expose a library resolver that returns a URL or local target while preserving the stored ref. No network access belongs in check. The registry does not change ticket schema.

This is the core required before export can offer or import can adopt mappings. Include the measured legacy values ticket:report and abbreviated origin-ticket: in compatibility tests; they remain valid when undeclared. A declared bad identifier and malformed registry need their distinct published check codes.

## Acceptance criteria

- [ ] Versioned tracked registry and ignored local overrides parse and render without config.yml loss
- [ ] check reports invalid registry and declared identifiers offline; undeclared legacy references stay valid
- [ ] Library resolver returns safe URL or local target and leaves stored reference bytes intact
