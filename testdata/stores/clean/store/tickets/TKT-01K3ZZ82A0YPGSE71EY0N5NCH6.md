---
schema: 1
id: TKT-01K3ZZ82A0YPGSE71EY0N5NCH6
title: Ship the token refresh work
type: epic
status: review
status_reason: null
priority: high
due_on: null
labels:
  - auth
  - docs
assignees:
  - human:sothr
milestone: v1.2
parent: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-08-31T12:04:00Z
updated_at: 2026-09-28T16:00:00Z
created_by:
  id: human:sothr
  name: Drew Short
updated_by:
  id: human:sothr
  name: Drew Short
extensions: {}
---

## Description

Status `review`, type `epic`. Both labels are in the allowlist and the
milestone is set, so this ticket also covers the non-null `milestone` path.

## Acceptance criteria

- [x] Refresh happens before expiry
- [x] A failed refresh does not log the token
