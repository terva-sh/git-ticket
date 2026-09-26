---
schema: 2
id: TKT-01M3DM2SJQSX7T51TX8WHK5ZGY
title: Show resolved reference targets across readers
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - area/cli
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

Expose resolved reference targets without changing stored ref or path bytes, per plan 5.5. Show, UI, and JSON should display an HTTPS destination or a locally bound foreign ticket where one is available, and remain useful when the foreign checkout is absent. refs keeps exact namespace and identifier lookup; add a discoverable resolution path without changing its current matching semantics. Exercise a PR reference, a foreign ticket, and an undeclared legacy reference end to end.

## Acceptance criteria

- [ ] Show, UI, and JSON expose resolved target without changing stored ref or path
- [ ] Refs lookup semantics remain unchanged and a resolution query is discoverable
- [ ] PR, foreign ticket, absent checkout, and undeclared legacy examples are exercised
