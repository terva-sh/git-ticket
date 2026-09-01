---
schema: 1
id: TKT-01K3ZZ4D40CJTD1X3WJF727MAQ
title: Delete the vendored copy of the old client
type: chore
status: in-progress
status_reason: null
priority: normal
labels: []
assignees:
  - agent:terva/session-123
milestone: null
parent: null
dependencies: []
references: []
claim:
  actor: agent:terva/session-123
  branch: chore/drop-vendored-client
  worktree: /Users/sothr/wt/drop-vendored-client
  commit: 4d5e6f7a8b9c
  claimed_at: 2026-09-29T08:00:00Z
  expires_at: null
archive: null
created_at: 2026-08-31T12:02:00Z
updated_at: 2026-09-29T08:00:00Z
created_by:
  id: human:sothr
  name: Drew Short
updated_by:
  id: agent:terva/session-123
  name: Mieli
extensions: {}
---

## Description

Status `in-progress`, type `chore`. The claim is what keeps this store free of
`in_progress_unclaimed`, and `expires_at: null` keeps it free of
`claim_expired`.
