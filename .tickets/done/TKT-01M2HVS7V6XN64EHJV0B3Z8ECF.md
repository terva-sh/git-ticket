---
schema: 2
id: TKT-01M2HVS7V6XN64EHJV0B3Z8ECF
title: Return the partial ImportResult when ApplyImport fails
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - area/integration
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
updated_at: 2026-09-16T19:16:18Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:claude/t3code
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

- [x] ApplyImport returns the tickets it filed before the failure, alongside the error naming what stopped it.
- [x] A test files a plan whose third ticket cannot be created, and proves the returned result carries the first two with their FromID mapping.
- [x] The handoff paragraph that offered this is updated to say it shipped, and names the version.
- [x] Whether the new return shape is a break under 12.4 is decided and recorded, either way.

## Definition of done

- [x] The worked example in docs/handoff-terva-interchange-library.md still compiles against the published module, as that document's status section requires.

## Notes

**agent:claude/t3code** at 2026-09-16T19:16:10Z

Shipped. All five error returns in `ApplyImport` now carry `out`; the nil-plan
guard still answers nil, because nothing was filed and an empty result would
claim a run that never happened.

**It is a break under 12.4, and it is recorded there.** The criterion asked for
a decision either way, and this is the less comfortable one.

The value's meaning changes rather than growing. It went from "nil whenever this
failed" to "what landed, whenever anything did", so a caller using `res != nil`
as a success test now reads a partial run as a whole one. 12.4 already has this
exact shape in its own list: a repair's `from` became nullable where it had
always been a path, and that is called a break rather than an addition for the
same reason.

The tempting argument the other way is that Go's convention says not to read a
value beside a non-nil error, so no correct caller is affected. That is true and
it is not enough. A rule that only bound correct callers would not be worth
writing down, and 12.4 says a break nobody wrote down is indistinguishable from
a regression.

`PlanImport` is untouched. It writes nothing, so it has no partial state, which
is what the ticket predicted.

**The definition of done was run, not reasoned about.** The worked example was
extracted to its own module, `require github.com/terva-sh/git-ticket v0.16.0`,
no `replace`, and built from the module proxy: `go build` and `go vet` both
clean. The example was written in the new style, reading `res.Filed` after a
non-nil error, and it compiles against the old module too, since the signature
is unchanged and the old one simply always returns nil there. So a caller can
write code for the new behaviour before the release carrying it exists.

**One thing the handoff paragraph does that the criterion did not ask for.** It
names v0.19.0 and says plainly that the tag does not exist yet. Writing "ships
in v0.19.0" unqualified would have told terva to pin a version the proxy would
refuse, which is a worse failure than the vagueness it replaced.

The test builds its `ImportPlan` by hand rather than parsing a patch, because
the third ticket has to be one this store refuses and `PlanImport` cannot
produce one: every field it fills is already reconciled against this store's
vocabulary. It was checked against the old behaviour and fails there with "no
result, so a caller cannot say what landed".

## Summary

ApplyImport returns the partial ImportResult at all five error sites, so res.Filed is what landed and the error is what stopped it, with FromID beside each minted ID. The nil-plan guard still answers nil. Recorded in plan 12.4 as a break: the value went from nil-on-error to partial-on-error, which is the same shape as the repair from field becoming nullable, and a caller using res != nil as a success test now reads a partial run as a whole one. The handoff paragraph names v0.19.0 and says the tag does not exist yet. The worked example was rebuilt against the published v0.16.0 with no replace.
