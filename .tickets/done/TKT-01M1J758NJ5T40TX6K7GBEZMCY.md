---
schema: 1
id: TKT-01M1J758NJ5T40TX6K7GBEZMCY
title: Give readiness a reason, because it cannot say why a draft is unready
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
created_at: 2026-09-02T23:27:09Z
updated_at: 2026-09-03T02:15:02Z
created_by:
  id: human:sothr
  name: ""
updated_by:
  id: agent:terva/mieli
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

## Summary

Shipped. `readiness.reason` names why a ticket cannot be picked up, and is empty exactly when `isReady` is true. In this repository's own store it turned ten of eleven open tickets from three empty arrays into `draft`.

Decision 1 took Option A, one enum rather than booleans, so the precedence is answered once here instead of separately in every consumer. Decision 2 held: a dependency-blocked ticket carries a reason like everything else.

Two naming calls came out of the work rather than the ticket. The dependency reason is `waiting_on_dependencies`, not `dependencies`, because `isBlocked` already means the graph and the `blocked` status means a person marked it, so each word now keeps one meaning per field. And the claim reason is `claimed`, not the `claimed_by_other` the ticket proposed, because `readinessOf` takes a store and a clock with no actor anywhere in the call. It cannot tell your claim from somebody else's, and a name asserting that difference would be one the code cannot check. A caller comparing `claim.by` against itself can.

The precedence is status, then dependencies, then the claim, and it reads as what has to change first. A draft waiting on a dependency reports `draft`, because promoting it is the first move. Nothing is hidden by that: `blockingDependencies` and `blockingChildren` are still populated, so the reason ranks the blockers and the arrays carry them.

Every status except `ready` is its own reason, so `UnreadyReasons` is derived from `Statuses` the way `OpenStatuses` is. A status added to 6.1 becomes a reason with no edit and no decision, which matters because custom statuses are still open in section 15.

Three mutations verified it. Flipping the precedence failed two tests. Unpublishing the two non-status reasons failed the guard in both directions. Renaming `ReasonClaimed` was the interesting one: the whole ticket package passed, because its assertions name the constant and moved with it, and only the literal `"claimed"` in the CLI test caught it. That is the same trap TKT-01M1HWBZ turned up one ticket earlier, and writing the CLI expectation as a literal on purpose is why it was caught this time.

Plan 8 carries the field, the precedence, and both naming decisions; 10.1 and 10.4 carry the JSON; 12.4 records it as additive.
