---
schema: 1
id: TKT-01M1SJNFB4R78S3M62NAFBK8WP
title: Centralize shortestUnique in the ticket package
type: chore
status: done
status_reason: null
priority: normal
due_on: null
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-05T20:02:55Z
updated_at: 2026-09-05T20:06:09Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

The rule that abbreviates a ticket ID for a listing exists twice, byte
for byte: `shortestUnique` in cli/commands.go and again in
tui/view/list.go, each with its own copy of `commonPrefixLen` and its
own `abbrevLen = 8`. The TUI copy's comment says it "mirrors the CLI's
rule in cli/commands.go", so the duplication is known and documented
rather than accidental.

Two copies of a rule is two places to be wrong. The rule is not
trivial: it shortens to what actually resolves, because a ULID opens
with ten characters of timestamp and a fixed width prints one
abbreviation on two rows for tickets created in the same millisecond.
A change to it that lands in one copy and not the other gives the CLI
and the TUI different answers for the same store, and nothing in the
suite compares the two surfaces against each other.

The trigger is the prefixed-ID-track findings recorded on
TKT-01M1R4B5. If a store ever carries more than one ID prefix, this
rule has to become prefix-aware: today it normalizes an ID, strips the
prefix, and pastes `TKT-` back on, which is what printed `TKT-IDEA-01M`
for a ticket named `IDEA-01M1SHS5631486JEM819TNMKV4`. Making that
correct means changing the rule, and it should be one change to one
function rather than the same change made twice and verified once.

The home is the `ticket` package, beside `NormalizeRef` and
`ResolveRef` in ticket/id.go. Abbreviation is the inverse of
resolution, the two have to agree by construction, and putting them in
one file means the track-aware change lands in one place.

The signature takes IDs rather than tickets. The CLI already passes
`[]string`; the TUI passes `[]*ticket.Ticket` and can map to IDs at the
call site, because the rule needs nothing from a ticket but its ID.

## Acceptance criteria

- [x] One implementation of the abbreviation rule exists, exported from the ticket package, and neither cli nor tui/view defines shortestUnique or commonPrefixLen.
- [x] The eight-character floor is defined once, beside the rule it bounds.
- [x] The CLI and the TUI print the same abbreviations they printed before the move, held by the tests that already cover each surface.
- [x] A unit test of the rule lives in the ticket package and covers two IDs that share their ULID timestamp, which is what a fixed width gets wrong.

## Notes

**agent:terva/mieli** at 2026-09-05T20:06:09Z

Built and all four criteria are ticked. The rule now lives in
ticket/id.go as `ShortestUnique`, beside `ResolveRef`, with
`commonPrefixLen` and the `abbrevLen = 8` floor. `cli/commands.go`
calls it from `storeAbbreviations`, and `tui/view/list.go` calls it
through a five-line `abbreviateIDs` that maps tickets to IDs. A grep
of the repository finds one definition of the rule, one of the helper,
and one of the floor.

The regression evidence is a before-and-after on this store rather
than a test alone. The installed binary was v0.10.0 at d85c66c, which
predates the move, and the built binary carried it. `list --all`,
`ready`, and `list --status draft` were byte-identical between them
across roughly eighty tickets.

Two tests came with the rule. `TestShortestUnique` moved from
cli/graph_test.go, keeping its two IDs that agree for twelve
characters, which is what tickets created in the same millisecond look
like. `TestShortestUniqueRoundTrips` is new and states the property
the rule exists for: every abbreviation resolves back through
`ResolveRef` to the ticket it came from. Testing the two against each
other is better than testing either against a constant, because they
have to agree and now they say so. cli/graph_test.go keeps
`TestListingAbbreviationsResolve`, the end-to-end property at that
layer.

One thing the move turned up that was not in the description. `cli`
already had a function called `abbreviate`, which cuts a title to a
column width. The first draft of the TUI helper had the same name for
a different operation on a different thing. Different packages, so it
compiled, but one word for two jobs is worse than a synonym for one.
It is `abbreviateIDs` now and the comment says why.

This does not make the rule prefix-aware. That is the change
TKT-01M1R4B5 describes, and it now has one place to land instead of
two.

## Summary

Shipped. The ID abbreviation rule is `ticket.ShortestUnique` in
ticket/id.go, beside `ResolveRef` because abbreviation is resolution's
inverse and the two have to agree. `commonPrefixLen` and the
`abbrevLen = 8` floor moved with it. The CLI calls it from
`storeAbbreviations`; the TUI calls it through `abbreviateIDs`, which
maps tickets to IDs and does nothing else. The repository now holds one
definition of each.

The move is behaviour-preserving and was checked that way: the v0.10.0
binary and the built one printed byte-identical `list --all`, `ready`,
and `list --status draft` on this store.

The unit test moved with the rule as `TestShortestUnique`, and
`TestShortestUniqueRoundTrips` was added to hold abbreviation and
resolution to each other directly. The end-to-end property stays in
cli/graph_test.go.

The motivation is TKT-01M1R4B5: making the rule prefix-aware is a real
change, and it now lands in one function instead of the same change
made twice and verified once.
