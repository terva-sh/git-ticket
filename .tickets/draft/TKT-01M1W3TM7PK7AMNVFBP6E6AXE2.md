---
schema: 2
id: TKT-01M1W3TM7PK7AMNVFBP6E6AXE2
title: Archive finished work in bulk by age
type: task
status: draft
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
created_at: 2026-09-06T19:41:19Z
updated_at: 2026-09-06T19:41:19Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Archive finished work in bulk by age, from section D of
`docs/review-backlog-md.md`.

`git ticket archive` takes one ID and an optional `--reason`, verified from
its own `--help`. Sweeping a store means running it once per ticket.
Backlog.md's `backlog cleanup` moves old done tasks out of the board's way in
one command, and this store is at 80 done tickets out of 92, which is what
the operation is for.

The shape the review proposed is `archive --status done --before DATE` with
`--dry-run`, and it wants the same care `check --fix` takes about naming what
it touched. `check --fix --dry-run` is the working precedent: it plans every
repair, writes nothing, and exits 1 when one is pending.

This is an idea lift and not a code lift, so under the ruling on this work it
takes a NOTICE entry and no file header.

Three things to settle before building.

Whether a bulk mode is a patch or a minor under plan 12.4. Flags beside old
ones have been patches, which is v0.5.1 and v0.7.1, but a mode that rewrites
many files in one invocation is a larger promise than a flag, and 12.4's
new-surface half has moved the minor before.

What `--if-revision` means when the write covers many tickets, since every
other write honours it against one.

Whether the sweep reports per ticket or in total, given that archiving moves
each file between directories and `pathsChanged` is what a caller reads.

## Acceptance criteria

- [ ] archive --status done --before DATE archives every matching ticket in one invocation
- [ ] --dry-run plans the sweep, writes nothing, and names every file it would move
- [ ] The JSON envelope reports every path the sweep touched
- [ ] Its release size under plan 12.4 is settled with the user before it ships

## Definition of done

- [ ] NOTICE credits Backlog.md for the idea, with no file header because no code was adapted
- [ ] check --strict is clean on a store that has been swept
