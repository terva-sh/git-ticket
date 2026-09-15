---
schema: 2
id: TKT-01M2HVS7V6XN64EHJV0B3Z8ECF
title: Return the partial ImportResult when ApplyImport fails
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - integration
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references:
  - ref: code:import
    path: ticket/import.go
claim: null
archive: null
created_at: 2026-09-15T06:24:02Z
updated_at: 2026-09-15T06:24:11Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

`docs/handoff-terva-interchange-library.md` says, under "Things that will bite":

> The error path also returns no partial result today, so a caller cannot report
> what did land from the return value alone. If that matters to you, say so and
> it can return the partial `ImportResult` alongside the error.

terva is saying so. This ticket is that answer.

### Why it matters on terva's side

`ApplyImport` is deliberately not atomic, and the reason is good: making it
transactional needs a scratch branch, and plan 7.3 forbids a helper that rewrites
a worktree. terva is not asking for that to change.

What terva cannot do today is tell the truth after a partial failure. If a create
fails on the third of five tickets, two are filed and the error names the third.
The caller holds an error and nothing else, so the honest report it can give a
user is "the import failed and some unknown number of tickets may exist in your
store, go and look". Recovery is reading what landed and removing it, and the
caller cannot even name what to read.

That lands badly in an agent host specifically. terva records what it did, and a
tool result that cannot say what it wrote is a gap in the record rather than
merely an inconvenient message. The agent's next move after a failed import is
either to re-run it, which duplicates the two that landed, or to stop and ask,
which is the right move but only if it can say what to clean up.

### The mapping is the part that is unrecoverable

`ImportResult.Filed` carries `FromID` beside the minted `ID`, and the handoff is
explicit that this mapping is the one thing a caller cannot work out afterwards.
That argument applies with more force to the failure path than to the success
one. After a success the caller could, at a stretch, re-read the store and match
on title. After a partial failure it does not know how many landed, so it does
not know where to stop matching.

### Shape

The handoff already proposes it: return the partial `ImportResult` alongside the
error, so a caller reads `res.Filed` for what landed and the error for what
stopped it. Go's convention allows a non-nil value with a non-nil error where the
value is meaningful, and here it is.

A caller written against today's signature keeps working, because one that
ignores the result on a non-nil error continues to ignore it. The risk is the
opposite one: a caller that already reads `res` without checking `err` would
start seeing a short list rather than a nil dereference. Whether that is a break
worth naming under 12.4 is upstream's call, not terva's.

`PlanImport` needs nothing. It writes nothing, so it has no partial state to
report.

### Not asked for

Atomicity. Rollback. Any reach into the origin store. terva only wants the error
path to say what the success path already says.

## Acceptance criteria

- [ ] ApplyImport returns the tickets it filed before the failure, alongside the error naming what stopped it.
- [ ] A test files a plan whose third ticket cannot be created, and proves the returned result carries the first two with their FromID mapping.
- [ ] The handoff paragraph that offered this is updated to say it shipped, and names the version.
- [ ] Whether the new return shape is a break under 12.4 is decided and recorded, either way.

## Definition of done

- [ ] The worked example in docs/handoff-terva-interchange-library.md still compiles against the published module, as that document's status section requires.
