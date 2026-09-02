---
schema: 1
id: TKT-01K3ZYG8K0Y52AD43XRGM4T7WZ
title: Rotate the signing key without downtime
type: epic
status: in-progress
status_reason: null
priority: urgent
due_on: "2026-10-14"
labels:
  - auth
  - docs
assignees:
  - human:sothr
  - agent:terva/session-123
milestone: v1.2
parent: TKT-01K3ZZRHN0393ED9MC6ZMFJTDT
dependencies:
  - TKT-01K3ZZPQ20RR5ARKYED1FCV2HV
  - TKT-01K3ZZMWF0K6YY0T5F5N4YTZY8
blocks_on: none
references:
  - ref: proposal:git-ticket
    path: docs/plan.md
  - ref: file:internal/auth/signing.go
    path: internal/auth/signing.go
  - ref: url:https://example.invalid/rfc/key-rotation
    path: null
  - ref: ticket:TKT-01K3ZZK1W0XWVS6RYX1JVVWSK3
    path: null
claim:
  actor: agent:terva/session-123
  branch: feat/key-rotation
  worktree: /Users/sothr/wt/key-rotation
  commit: a1b2c3d4e5f6
  claimed_at: 2026-08-31T12:04:00Z
  expires_at: null
archive: null
created_at: 2026-08-31T12:00:00Z
updated_at: 2026-08-31T12:06:00Z
created_by:
  id: human:sothr
  name: Drew Short
updated_by:
  id: agent:terva/session-123
  name: Mieli
extensions:
  terva:
    session: session-123
---

## Description

The signing key has never been rotated. Rotating it naively invalidates every
live session, so the work is to stand up a second key, accept both during a
window, and retire the first.

## Acceptance criteria

- [x] The verifier accepts a token signed by either key
- [ ] A new token is always signed by the newer key
- [ ] The old key is removed after the overlap window closes

## Definition of done

- [ ] Rotation is documented in the runbook
- [ ] A rotation drill has run against staging

## Implementation plan

Add a key set rather than a single key, keyed by `kid`. The verifier picks the
key named in the token header and rejects a token whose `kid` is unknown.

## Notes

The overlap window has to outlast the longest refresh token, which is thirty
days, so the retirement step cannot be part of the same release.

## Comments

**human:sothr** at 2026-08-31T12:05:00Z

Staging has a hardcoded key in its fixtures. That needs to move first.

## Summary

Two keys live at once, selected by `kid`, with a thirty day overlap.
