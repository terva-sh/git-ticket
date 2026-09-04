---
schema: 1
id: TKT-01M1PCYC14VD3TJYDE1EWC671Y
title: Decide how a ticket filed by mistake is removed
type: spike
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
created_at: 2026-09-04T14:25:12Z
updated_at: 2026-09-04T17:26:54Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

There is no delete. archive is the soft close and it keeps the file, which is right for work that happened and wrong for a ticket that should never have existed.

The case is concrete and AGENTS.md already documents it. Text passed to a body section carrying a heading of two hashes is split into sections by parseBody, so a description written with Markdown subheadings lands partly in body.extra. The CLI warns on stderr and writes anyway, because passing several sections in one string is sometimes meant. The repair AGENTS.md records is to remove the file by hand and create the ticket again, because update --description replaces only the description and the stray sections survive. TKT-01M1HVMQQQE3K6VZG7793RXVXN had to be filed twice for exactly this.

So the tool documents a repair it cannot itself perform. That is the gap.

What to settle. Whether a removal exists at all, given that archive covers the honest close and a Git-native store can always be repaired with rm and a commit. If it does, whether it refuses anything carrying history, meaning notes, comments or a claim, so it can only take a ticket nobody has worked. And whether it is its own command or a flag on one that exists.

The counter-argument deserves a fair hearing. rm on the file plus a commit is the whole operation, the store is files by design, and a command wrapping one rm earns its place only by refusing the cases where rm is wrong. That refusal may be the actual product here.

This carries no question label on purpose. That label marks a question parked in plan section 15, and every ticket holding it has an entry there. This is queued work rather than a parked question, so labelling it would break the invariant that section 15 names every question ticket.

## Acceptance criteria

- [x] docs/plan.md records whether a removal exists and what it refuses
- [x] The mis-sectioned repair in AGENTS.md names a command, or is rewritten to say the format has none

## Summary

Settled. Plan section 9.1 is the decision and `TKT-01M1PQ7T` is the build.

A removal exists, as its own command `remove ID`. But the finding that shaped
the answer is that the deletion was never the hard part: deleting an
unreferenced ticket file leaves `check --strict` green, measured rather than
assumed, so for the case that motivated the ticket `rm` already does the whole
job. A command wrapping one unlink would earn nothing.

What `rm` cannot do is say when it is wrong. Deleting a ticket another names in
`dependencies` or `parent` leaves `dependency_missing` or `parent_missing`, and
`check --fix` repairs neither, because choosing between unlinking the dependent,
re-pointing it, and re-creating the target is the same judgement it already
declines for `duplicate_id`. So the refusal is the product. That answers the
ticket's own counter-argument in its favour: the refusal is indeed the actual
product, and the design is almost entirely refusals.

It refuses two things and deletes otherwise. First, another ticket naming it in
`dependencies` or `parent`, and the refusal names them, which is the whole of
what it buys over `rm`: finding that out by hand means grepping the store for an
ID. Second, carrying `Notes`, `Comments`, or a `Summary`, or holding a claim,
because `archive` is the operation for work that happened.

"Nobody has touched it" turned out to be checkable rather than a judgement. An
untouched ticket has `claim: null` and `archive: null` and a body carrying a
`Description` and nothing else. A worked one has `Notes`, `Comments`, and a
claim block.

`--force` overrides both and reports the dangling references it created. It
overrides rather than refusing absolutely because a refusal a person routes
around with `rm` has taught them nothing.

Its own command, not a flag or a status. It cannot be a `Mutation`, because
every mutation rewrites a ticket in place and this deletes the file, so it sits
on the `Store` beside `Create`. It cannot be a status, because every status in
6.1 describes a ticket that still exists. It cannot be a flag on `archive`,
which means keep. It is also the one operation in section 9 returning no
resulting ticket, so it returns the one it removed.

It writes no Git history, per 7.4, so the deletion lands in the working tree for
a person to stage. That is also the undo.

The second acceptance criterion took the honest branch. AGENTS.md now says
plainly that there is still no `remove` command, points at 9.1 and the work
ticket, and records what `rm` costs in each case, rather than naming a command
the binary does not have. Naming an unbuilt command is the failure this
repository already guards against for `instructions`.

One thing found on the way and fixed here: the AGENTS.md paragraph describing
the heading warning still listed only the inline flags, which went stale when
the file flags shipped. All 14 spellings were checked, and every one warns and
names the flag the text arrived through.
