---
schema: 1
id: TKT-01M1JPXG7J6K4MQNTWA6TVMHPB
title: Decide whether updated_at and updated_by earn their cost
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
updated_at: 2026-09-03T04:02:32Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Raised while settling the merge driver on TKT-01M1HE7KX06FY8W1GYXH9MXGBP. That spike found these two fields are the dominant source of merge conflicts, and a driver would resolve them. This asks the question a driver would paper over: should they be there at all?

5.3 already defends them and the defence has to be answered rather than ignored. It says they change on every mutation so every diff shows at least those two lines, and that this is intended, because the diff should say who touched the ticket and when.

### The measured cost

Two edits touching nothing in common still conflict. Branch A setting `priority` and branch B adding a label produced exactly one conflict hunk, on `updated_by`, while Git merged the priority and the label without complaint. The two agents disagreed about nothing and the file would not merge. `updated_at` behaves the same way whenever the edits fall in different seconds.

That is the whole conflict story for concurrent edits to one ticket, aside from append position in `Notes` and `Comments`.

### The case for dropping them

Git already records who and when, per commit, for every line. `git log -p` and `git blame` on a ticket file answer "who touched this and when" with more precision than a single last-writer field, because they answer it per change rather than for the file as a whole. The fields may be restating what the object database holds, at the price of a conflict on every concurrent edit.

Nothing in this repository reads them. `UpdatedAt` and `UpdatedBy` are written by `apply`, rendered by `render`, read back by `parse`, and published in the ticket envelope. No query, filter, sort, or check consumes either. A field nothing reads is carrying its cost on the strength of the diff argument alone.

### The case against dropping them

A store is not always read through Git. A tarball, an export, or a directory somebody copied has no history, and the fields are the only provenance those carry.

They are also a published surface. `updatedAt` and `updatedBy` are in the ticket envelope, so removing them is a breaking change under 12.4 and not an additive one. That is the real cost, and it is why this is worth deciding before Phase 3 rather than after a consumer depends on them.

### What has to be decided

Whether both stay, both go, or one goes. `updated_at` and `updated_by` are separable: a timestamp is cheap to defend and an actor is the field the conflict evidence actually indicts, since two agents always differ on it while two edits in the same second do not differ on the timestamp.

Also open is whether the merge driver settles this by itself. If a driver resolves both fields silently and correctly, the cost falls to nearly zero and the diff argument survives intact. That is the honest counter to this whole ticket, and whoever picks it up should build or read the driver first.
