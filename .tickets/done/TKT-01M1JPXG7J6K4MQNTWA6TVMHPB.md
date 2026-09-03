---
schema: 1
id: TKT-01M1JPXG7J6K4MQNTWA6TVMHPB
title: Decide whether updated_at and updated_by belong in the published surface
type: spike
status: done
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
updated_at: 2026-09-03T17:13:38Z
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

## Summary

Settled: both fields stay, and they are one fact rather than two. 5.3 holds the reasoning, 12.4 records the compatibility promise, and section 15 moves the entry from open to settled.

Once 7.5 retired the conflict cost, three arguments for dropping were left. Two did not survive being checked.

Nothing reads either field, which is true and is not a criterion, because nothing reads created_at or created_by either. No query, filter, sort or check consumes any of the four. Provenance answers a question a person asks of a diff, not one a caller asks of a store, so being unread is the ordinary condition of the group. A rule that dropped a field for having no reader takes created_by with it.

The stronger argument was that a Git-native format restates what the object database already holds with more precision. That is false here and measurably so. Nothing in this project commits, per 7.4, so a mutation sits in the working tree until somebody commits it. And a commit carries the committer Git identity while an actor is an agent session: this store holds agent:terva/mieli, agent:terva/dev-loop and human:sothr across its tickets against one commit author on every commit. Git collapses three into one, so it holds strictly less than the pair on the axis an agent workflow cares about.

Splitting is not a third option. 7.5 resolves the actor by the later timestamp, so dropping updated_at publishes an updated_by with no rule behind it.

The find worth keeping: the surface being argued about had no guard. No test in either package named updatedAt or updatedBy, so removing both from the envelope would have passed the whole suite in silence, the same failure --version shipped with in v0.4.0. cli/provenance_test.go now holds the four keys as literal strings and holds the pair to moving together. Two mutations confirmed it bites.

No production code changed, which is the correct shape for an answer of keep. The diff is the plan, one test file, and this ticket.
