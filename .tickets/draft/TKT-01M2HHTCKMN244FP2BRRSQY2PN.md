---
schema: 2
id: TKT-01M2HHTCKMN244FP2BRRSQY2PN
title: Decide whether a range of notes can be folded into one
type: spike
status: draft
status_reason: null
priority: low
due_on: null
labels:
  - question
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-15T03:29:54Z
updated_at: 2026-09-15T03:29:54Z
created_by:
  id: agent:terva/compact-notes
  name: ""
updated_by:
  id: agent:terva/compact-notes
  name: ""
extensions: {}
---

## Description

Whether `git ticket note ID --fold RANGE` should replace a range of notes, or all
of them, with one note the author writes.

Deferred from the same session that built the display compaction of
TKT-01M2HHSVT7FABTV4JVM9126KBN, and deliberately not built with it: compacting
what `show` prints destroys nothing, while folding rewrites a log, so the two do
not belong in one change.

### Why it may no longer be needed

The stated pain was that notes get long. Most of that cost was paid at read time,
and the display change takes it away without touching a file. What folding
answers that display does not is a note that is now wrong: AGENTS.md supersedes a
note by name, appending a second that argues against the first, so the ticket
carries both positions forever and a reader must work out which won. Folding is
the operation that would let the superseded pair become one correct note.

So the question is not "are notes long" any more. It is whether a superseded note
should be removable, and that is worth asking separately.

### The safety problem, which is real

The tool cannot tell whether the notes it would destroy are committed. Plan 7.4
lists every git command this code may run and has no row for `status` or `diff`,
and admitting one is a maintainer's decision rather than a new command's, per the
`format-patch` ruling in section 15. So a fold run before a commit loses the
original text with no recovery.

Mitigations worth considering: print the full text of every entry removed, so it
is at least in the terminal; require an explicit flag; or accept it and say so.
None of them make the tool able to check.

### Where the machinery already is

`ticket.Entries` parses a log section into numbered entries carrying actor and
instant, and `mergeEntries` in ticket/merge.go already reasons about entries as
records. A fold would be a mutation over that view. The library half is small;
the ruling is the expensive part.

### What a fold would have to record

A folded note that did not say what it replaced would let a ticket silently lose
nine stamped entries. At minimum the count, the span of instants, and the actors
whose entries went, composed by the tool the way `originRecord` composes
provenance for an adopted ticket.

### Trigger

Somebody who has a superseded note pair and wants one correct note, after the
display compaction has shipped and the volume argument no longer applies.
