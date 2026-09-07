---
schema: 2
id: TKT-01M1WQ1PWA0TPQATQ66978AVC2
title: Record the agent session on a claim, at schema 3
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels: []
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-07T01:17:14Z
updated_at: 2026-09-07T01:33:28Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

A claim records where the work happened in the repository: `branch`,
`worktree`, `commit`. It cannot record where the work happened in the record,
which is the agent session that did it.

terva knows its own session id and has nowhere to put it. Requested in its
second handoff, `handoff-git-ticket-session-on-claim.md`, alongside a survey
of what it checked and found sufficient.

### Why the field rather than a workaround

Both workarounds fail, and each fails differently. Measured on v0.13.0:

`decodeClaim` in ticket/parse.go has no `default` branch, so an unknown
sub-key under `claim:` is silently dropped rather than preserved or refused.
A session id written inside the claim block does not survive the next write.

An unknown top-level key does round-trip, so `x-session:` would survive. But
`ReleaseClaim.apply` sets `t.Claim = nil`, so the claim block vanishes on
release while a top-level key stays. The session id would outlive the claim
it describes and point at nothing. terva proposed this as its fallback
without knowing that, so it is worth telling them.

terva also declined to squat an ordinary `session:` key, correctly.
`promoteUnknown` refuses with "no rule for promoting it to schema N" when the
target level defines a key that has no promotion rule, so a squatted key
becomes a refused migration for whoever migrates next.

### Why it costs a schema level

Without one, an older reader meeting `claim.session` drops it on the next
write, silently, because `decodeClaim` ignores what it does not know. That is
the quiet wrong answer plan 5.6 replaced with one loud `schema_unsupported`.

Under plan 12.4 a schema bump is an ordinary minor: "The module version
tracks the Go API alone, which makes a `schema` bump an ordinary minor release
rather than a `/v2`."

### Shape

`Session *string` on `Claim` and `Session string` on `ClaimTicket`, matching
how `Branch`, `Worktree` and `Commit` already pair across the two. A named
typed field merges predictably and the merge driver can reason about it the
way it already reasons about branch and worktree.

terva argued against a free-form `map[string]string` on the claim, and this
agrees: it merges badly and becomes a junk drawer.

The `x-` embedder namespace terva offered as a general alternative is not
taken here. It solves squatting but not lifecycle, and this field wants
lifecycle. It stays available as a separate question if a second embedder
asks for one.

## Acceptance criteria

- [x] Claim carries Session *string and ClaimTicket carries Session string, matching how branch, worktree and commit pair
- [x] SchemaVersion is 3 and a schema-2 store migrates to it, with nothing to promote because a claim sub-key never enters Unknown
- [x] A round trip preserves claim.session, and releasing a claim drops it with the rest of the block
- [x] The CLI can set and read it, and the JSON envelope publishes it under claim
- [x] The fixture corpus covers a claim with a session and one without
- [x] docs/plan.md defines schema 3 and the field before the code does

## Notes

**agent:terva/mieli** at 2026-09-07T01:33:28Z

Built and shipped in v0.14.0. All six criteria tick, with one reworded before
the build; the original is recorded below.

`claim.session` exists at schema 3, in the first shape terva listed:
`Session string` on `ClaimTicket`, `Session *string` on `Claim`. It renders
after `commit`, updates on a renewal the way branch and commit do, and goes
when the claim goes. `claim --session` and the JSON envelope carry it.

### Acceptance criterion 2 was reworded

It read, verbatim:

    SchemaVersion is 3, and a schema-2 store migrates to it with a promotion
    rule so promoteUnknown does not refuse

That was unsatisfiable as written, and wrong about the mechanism.
`knownFieldsAt` in ticket/parse.go builds a set of top-level keys only, and
`promoteUnknown` walks `t.Unknown`, which holds top-level keys alone. A claim
sub-key never reaches either. So no promotion rule was needed, could be
written, or would have been reached by a test.

The criterion now asks that a schema-2 store migrates to 3 and that nothing
needs promoting, which is what the code actually has to satisfy.

### What was checked and deliberately not built

From terva's section 3, recorded so nobody re-derives it. Each was read in
the tree rather than taken on report:

`ClaimTicket.ExpiresIn` and `Force` are sufficient for a subagent claim that
should lapse rather than litter. Unchanged.

`Open`, `OpenOptions` and `Store.Root()` are sufficient for addressing more
than one store. Unchanged.

`Store.Template` and `Store.Templates` are already in the library, so the
lift terva expected to need does not exist to do.

`SetChecklistItem` already covers checking an acceptance criterion. terva had
told us informally that it did not and corrected itself.

`SetChecklistItem.Index` stays positional, counting from one in document
order. terva observed that the revision precondition makes it safe, because a
concurrent insert changes the revision and the write refuses. That is now the
recorded reason rather than an unexamined choice. No stable item ids.

### The `x-` namespace is not settled by this

terva offered a documented `x-` embedder namespace as the general alternative,
and it is not taken here, because it does not solve this field. It remains
available as its own question if a second embedder asks for one. Nothing in
this ticket decided it either way.

## Summary

Shipped in v0.14.0. `claim.session` records which agent session did the work,
at schema 3. Every other claim field says where the work happened in the
repository; this one says where it happened in the record, so a ticket becomes
an index into transcript history.

`Session string` on `ClaimTicket`, `Session *string` on `Claim`, rendered after
`commit` and gated by `hasClaimSession`. `claim --session` sets it and the JSON
envelope publishes `claim.session`, null below schema 3. A renewal updates it;
releasing drops it with the block.

It is a claim sub-key rather than a top-level key because `ReleaseClaim` nils
the block, and a top-level key would outlive the claim it describes. It needed a
schema level because `decodeClaim` silently drops sub-keys it does not know, so
an older reader would discard the value on its next write.

Acceptance criterion 2 was reworded before the build: no promotion rule was
needed or possible, because `promoteUnknown` only ever sees top-level keys. The
original wording is in the notes.
