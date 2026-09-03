---
schema: 1
id: TKT-01M1JPXG7J6K4MQNTWA6TVMHPB
title: Decide whether updated_at and updated_by belong in the published surface
type: spike
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - format
  - question
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-03T04:02:32Z
updated_at: 2026-09-03T04:31:17Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Raised while settling the merge driver on TKT-01M1HE7KX06FY8W1GYXH9MXGBP, and narrowed after review. The original framing led with the merge cost, and plan 7.5 retires that argument: the driver takes the later `updated_at` and the actor belonging to it, a rule needing no judgment. Do not re-litigate the conflict here. What is left is a question about the published surface, and it stands on its own.

`updatedAt` and `updatedBy` are in the ticket envelope. 12.4 covers every machine-readable surface, so removing either after a consumer depends on it is a breaking change rather than an additive one. Phase 3 is when a consumer arrives. The choice is therefore open now and closed later, which is the only reason this is worth deciding before somebody asks for it.

### What the fields do today

Nothing reads them. `apply` writes both on every mutation, `render` emits them, `parse` reads them back, and the ticket envelope publishes them. No query, filter, sort, or check consumes either. Claim expiry uses the claim own `expires_at` and not `updated_at`. No test asserts on either field. They are a write-only pair with a published surface attached.

### The case for keeping them

5.3 makes it directly: they change on every mutation so every diff shows at least those two lines, and that is intended, because the diff should say who touched the ticket and when.

The stronger version is about stores read outside Git. A tarball, an export, or a directory somebody copied carries no history, and these two fields are the only provenance it has. A format that is Markdown first should not need `git log` to answer who last touched a file.

### The case for dropping them

In a Git-native format they restate what the object database already holds, with less precision. `git log -p` and `git blame` answer who and when per change; a last-writer pair answers it once for the whole file and is wrong about every earlier edit.

A field nothing reads, in a format that is about to freeze its published surface, is worth a deliberate yes rather than an inherited one.

### What has to be decided

Keep both, drop both, or drop one, and say it in 12.4 terms so v1.0.0 makes the promise on purpose.

If they are split, the distinction is no longer the merge behaviour, since the driver treats them as one fact. It is that a timestamp is legible to any reader, while an actor id is only meaningful where actor strings are stable, which the format does not enforce and `config.yml` only advises.

### Trigger

Before Phase 3, and before v1.0.0. After that the answer is whichever one shipped.
