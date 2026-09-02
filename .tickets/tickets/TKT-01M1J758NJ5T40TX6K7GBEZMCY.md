---
schema: 1
id: TKT-01M1J758NJ5T40TX6K7GBEZMCY
title: Give readiness a reason, because it cannot say why a draft is unready
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
created_at: 2026-09-02T23:27:09Z
updated_at: 2026-09-02T23:27:09Z
created_by:
  id: human:sothr
  name: ""
updated_by:
  id: human:sothr
  name: ""
extensions: {}
---

## Description

`readiness` reports `isReady: false`, `isBlocked: false`, and three empty arrays for every one of the 13 drafts in this store. That is correct and it is useless. A caller learns the ticket cannot be picked up and nothing about why.

The README promises more. It says a consumer drawing a board can "grey out a card and say why without calling `ready` and then `deps` per row". It cannot say why for 13 of the 14 open tickets here. The reason has to be re-derived from `status` and `claim`, which is the restating of the rule that `readiness` exists to prevent.

The README is honest about the gap one paragraph later: "A draft, and a ticket somebody else is holding, are both unready with nothing in the way but their own state." That sentence names both cases. It just does not put them in the payload.

### Scope

A field on `readiness` naming why a ticket is not ready when no dependency is in the way. No change to the verdict `ready` filters on, because the verdict itself is right. No new finding code, no change to the store format.

### Decision 1: a reason enum or a set of booleans

Option A, a `reason` string whose values `schema` publishes, such as `draft`, `claimed_by_other`, `done`, and `archived`, empty when `isReady` is true. One field to switch on, and `schema` already exists as the place to publish legal values.

Option B, booleans beside the existing arrays. Nothing to version, but a consumer then has to know the precedence when more than one is true, and a ticket that is both `done` and still claimed is not far-fetched.

Recommendation is A. The precedence question in B is real, and an enum forces it to be answered once here rather than separately in every consumer.

### Decision 2: does a dependency-blocked ticket also carry a reason

Yes, set it to `dependencies` rather than leaving it empty. A field that is sometimes empty for a reason the caller has to infer is the defect this ticket is filing, and reproducing that inside the fix would only be funny once.

### Compatibility

`readiness` belongs to the `ticket` and `ticket-list` kinds, so this changes a machine-readable surface under 12.4 and lands in a minor release. The change is additive, so a consumer that ignores the field is unaffected.

### Where this came from

A review of the work-finding path on 2026-09-02, which read `readiness` across the whole store and found it empty-handed on every draft.
