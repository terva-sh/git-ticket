---
schema: 2
id: TKT-01M2HGWRY8A93VVJHGF3GXX6N7
title: Decide whether imported tickets land in an inbox directory
type: spike
status: draft
status_reason: null
priority: low
due_on: null
labels:
  - question
  - format
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-15T03:13:44Z
updated_at: 2026-09-15T03:13:44Z
created_by:
  id: agent:terva/inbox-question
  name: ""
updated_by:
  id: agent:terva/inbox-question
  name: ""
extensions: {}
---

## Description

Whether imported tickets should land in a new `.tickets/inbox/` directory rather
than in the directory their status implies, with the import able to ask for
their locations to be preserved instead.

The need behind it is real. An adopted ticket lands in `draft/` mixed with the
drafts this project wrote itself, and somebody reading `draft/` to decide what to
promote cannot tell one from the other by looking.

### What already answers it

`refs` already separates them, measured rather than assumed. Every adopted ticket
carries `origin-ticket:` and, with `--from-store`, `origin-store:NAME`, so in a
store holding one local draft and two adopted ones, `git ticket refs
origin-ticket:` prints exactly the two that arrived and `git ticket refs
origin-store:ledger` scopes that to one sender. No format change, no schema move,
and it works in every store that has ever run an import.

What `refs` does not do is compose. It is its own command, so the provenance
question cannot be combined with the status, priority and sort vocabulary that
`list` carries. That gap is the part of this worth building, and it is much
smaller than a directory.

### What the plan already says

Section 4 settles the general form of this question and did so against a
concrete proposal. "Status is the only thing a directory keys on. A path is one
dimension, so any partition spends its single slot once." The rejected candidate
was `epics/`, and the reason transfers without modification: an epic "is also not
a boring file, so pulling epics out of `tickets/` would remove the most
interesting rows from the view this section exists to make readable." An arrived
ticket is the least boring draft in the store. `inbox/` would take exactly the
rows a person most needs to see out of the view they read to decide what to
promote.

The conditional half collides with a second sentence of section 4 rather than
the first: "The layout is the format. It is not configurable and not opt-in,
because a layout a store can decline is one no reader can count on." An import
flag that preserves locations is a layout the store declines per invocation,
which is the thing that sentence refuses.

### What it would cost mechanically

The directory is a pure function of status today. `statusDir` is the whole
mapping and `files()`, `Init`, `writeTicket`, `check` and `relTarget` all read
it, which is what makes `location_mismatch` decidable: a status implies exactly
one directory, so a file is either in it or not. With `inbox/`, a `draft` ticket
is legal in two places and nothing in the file says which, so `check` cannot tell
an inboxed ticket from a misplaced one without a frontmatter field recording it.
That is a schema move for a field whose only job is to remember where its own
file sits, and `check --fix` then needs to know not to repair the inbox away.

### The objection that may settle it

`draft/` already is the inbox. Plan 12.8 files every imported ticket as `draft`
whatever it was at source, "because promotion is where somebody weighs the work
against what the sender cannot see". The review gate this proposal asks for is
already built and already has a directory. Adding `inbox/` would split one
concept across two places rather than add a missing one.

The one hole in that objection is `--same-owner`, which shipped today: a ticket
adopted with it can arrive `done` or `archived` and land in `done/` or
`archive/`, never passing through `draft/` at all. But that is the flag doing
what it was asked to do, since asserting `--same-owner` is asserting the review
already happened, so it argues for nothing here.

### Worth considering, not prescribed

`list --ref REF` is the cheap answer: it makes provenance compose with the filter
vocabulary that already exists, adds no directory, moves no schema, and leaves
every store readable by every older binary. A line in the import output naming
the query that finds what just arrived is cheaper still and could ship with it.

A third option nobody has argued for yet is a label applied at adopt time, which
composes with `list` today and needs no code at all, but spends a name in every
receiving store's allowlist and turns provenance into vocabulary a person can
edit.

### Trigger

Somebody who cannot find what arrived, after `refs origin-store:` and a
composing `list` filter both exist. Not a preference for browsing directories,
and not the mixing itself, which is what `draft/` is for.
