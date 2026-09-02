---
schema: 1
id: TKT-01K3ZYQJZ0JMSK4YH3J13BKYV2
title: Retry a failed provider request once
type: bug
status: ready
status_reason: null
priority: high
due_on: null
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references: []
claim:
  actor: agent:terva/session-991
  branch: fix/provider-retry
  worktree: null
  commit: 9f8e7d6c5b4a
  claimed_at: 2026-09-01T09:00:00Z
  expires_at: 2026-09-08T09:00:00Z
archive: null
created_at: 2026-08-31T12:00:00Z
updated_at: 2026-09-01T09:00:00Z
created_by:
  id: human:sothr
  name: Drew Short
updated_by:
  id: agent:terva/session-991
  name: Mieli
extensions: {}
---

## Description

The claim expired three weeks before the corpus reference time of
2026-09-30T00:00:00Z, so `check` warns and `ready` treats the ticket as
available again.

Expiry does not revoke the original agent's work and grants nobody
exclusivity, per 6.4. The status stays `ready` rather than moving, because a
claim is metadata and not a status.

`worktree` is null on purpose: a claim made in a bare checkout or through the
library has no worktree path to record, and that has to round-trip as null
rather than as an empty string.
