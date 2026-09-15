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
updated_at: 2026-09-15T03:45:54Z
created_by:
  id: agent:terva/compact-notes
  name: ""
updated_by:
  id: agent:terva/committed-check
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

### Whether the tool can tell a fold is safe

It can. This section replaces an earlier version of it that said the opposite;
the note below records what was claimed and how it was falsified.

"Is this ticket's current text in history" needs `rev-parse HEAD` for the commit,
`ls-tree -r HEAD -- PATH` for the blob name the file carries there, and
`ticket.BlobSHA` of the bytes on disk, which computes git's object name locally
and is held to `git hash-object` by `TestBlobSHAMatchesGit`. Equal names mean
every byte now on disk is reachable from HEAD, so a destructive edit loses
nothing that `git show` cannot return. All three rows are already in plan 7.4 and
`crossbranch.go` already uses two of them, so this needs no new row and no plan
change.

Every edge fails safe. A file absent from HEAD, a repository with no commits, a
store outside a repository, and content staged but not committed all answer "not
in history", which is the conservative direction for a caller about to destroy
something.

That turns the safety question from "can we know" into a design choice about what
to do when the answer is no: refuse and say what to commit first, warn and
proceed, or print the removed text so it is at least in the terminal. That choice
is what this ticket still has to settle.

### Where the machinery already is

`ticket.Entries` parses a log section into numbered entries carrying actor and
instant, and `mergeEntries` in ticket/merge.go already reasons about entries as
records. `cli/notes.go` carries `parseNoteRange`, which already reads `N`, `N-M`
and `all` for `--show` and would serve `--fold` unchanged. A fold would be a
mutation over that view; the library half is small and the ruling is the
expensive part.

### What a fold would have to record

A folded note that did not say what it replaced would let a ticket silently lose
nine stamped entries. At minimum the count, the span of instants, and the actors
whose entries went, composed by the tool the way `originRecord` composes
provenance for an adopted ticket.

### Trigger

Somebody who has a superseded note pair and wants one correct note, after the
display compaction has shipped and the volume argument no longer applies.

## Notes

**agent:terva/committed-check** at 2026-09-15T03:45:54Z

Corrected the safety section of the description, which was wrong when this
ticket was filed earlier today. The original wording is preserved here because a
reader should not have to run git log to learn what was claimed.

It said: "The tool cannot tell whether the notes it would destroy are committed.
Plan 7.4 lists every git command this code may run and has no row for status or
diff, and admitting one is a maintainer's decision rather than a new command's,
per the format-patch ruling in section 15. So a fold run before a commit loses
the original text with no recovery." It went on to offer three mitigations and
said "None of them make the tool able to check."

Both halves were wrong. The tool can check, and it needs no new row. rev-parse
and ls-tree are already in the 7.4 table and crossbranch.go already uses both,
and ticket.BlobSHA already computes git's blob name locally without running git,
held to git hash-object by TestBlobSHAMatchesGit. Comparing the blob name
ls-tree reports for HEAD against BlobSHA of the bytes on disk answers whether
the current text is reachable from HEAD.

Falsified by prototype rather than by reading, over five cases in a scratch
repository: a freshly committed store reports committed; a store with one
uncommitted note reports not committed; the same content staged but not
committed still reports not committed, which is the safe direction since the
index is not history; committing it flips the answer back; and a ticket file
that has never been committed is absent from HEAD entirely. The locally computed
blob name matched git hash-object exactly in the control.

What this changes about the ticket. The safety problem is not "we cannot know",
it is "what should happen when the answer is no", which is a smaller and
different question: refuse and name what to commit, warn and proceed, or print
the removed text. That is now the open part.

The lesson is the one AGENTS.md already carries and this is a fresh instance of
it, on work half an hour old. A claim about mechanism went into a ticket without
the run that would falsify it, and the claim read as a finding because it cited
a real table and a real ruling. Both citations were accurate and the conclusion
drawn from them was not. Plan 7.4 now records that the question is answerable
with the rows it already has, so the next reader does not derive the same gap
and propose the same unnecessary row.
