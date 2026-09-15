---
schema: 2
id: TKT-01M2HHSVT7FABTV4JVM9126KBN
title: Compact the note history git ticket show prints
type: task
status: done
status_reason: null
priority: high
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
created_at: 2026-09-15T03:29:37Z
updated_at: 2026-09-15T03:35:18Z
created_by:
  id: agent:terva/compact-notes
  name: ""
updated_by:
  id: agent:terva/compact-notes
  name: ""
extensions: {}
---

## Description

`git ticket show` prints every note in full, so reading a worked ticket costs
whatever its whole history costs. Measured in this store: the worst ticket prints
463 lines and 3,626 words from one `show`, a ticket closed the same day prints
1,612, and across 75 note-carrying tickets the notes hold 32,835 words at a
median of 322 per ticket.

That is the cost this ticket is about. For a person it is scrolling. For an agent
it is context: `show` is the ordinary way to read a ticket, and on a worked one it
returns several thousand tokens of history to answer a question usually about the
current state. The ask came from a model that had encountered exactly that.

### What changes

`show` prints the most recent note in full and replaces the rest with one line
naming how many there are and how to read them. Nothing is deleted and no file
changes: this is display.

The newest-in-full rule was chosen over a size threshold because a threshold is a
magic number that makes output unpredictable between stores, and over hiding
every note because a one-note ticket would then cost a second command to read
what used to be free. The newest note is nearly always the live one.

### What it does not change

`--json` is untouched. `body.notes` is a string today and a consumer reads it as
one; truncating it would be a break under 12.4 for no gain, since a caller that
asked for the envelope can slice it. The compaction is additive, so no existing
invocation changes meaning and no store needs a repair.

### Retrieval

`note ID --list` prints the index: number, actor, instant and opening line.
`note ID --show N` or `--show N-M` prints the text of those entries.
`ticket.Entries` already parses a log section into numbered entries carrying
actor and instant, so the numbering exists and this exposes it.

### Existing functionality

The envelope already publishes `comments` as a numbered list through
`entryJSON`, while `notes` is a bare string, and nothing found in the plan or the
tree argues for the asymmetry. It looks like comments got the treatment and notes
never did. Worth noting when deciding how far to take this.

### Trigger

Not applicable; this is a build rather than a deferred question.

## Acceptance criteria

- [x] show prints the most recent note in full and replaces every earlier one with a single line naming how many there are and their numbers
- [x] That line names the command that retrieves them, so the reader needs nothing else to get the text back
- [x] note ID --list prints every note as number, actor, instant and opening line
- [x] note ID --show N and --show N-M print the full text of those notes
- [x] A ticket with one note or none prints exactly what it printed before, with no index line added
- [x] show --json is unchanged, proven by comparing the envelope against the pre-change binary
- [x] Measured on this store's worst ticket: show drops from 3,626 words to a small fraction, with the drop recorded

## Definition of done

- [x] just ci is green and check --strict passes on this store

## Summary

Shipped. `show` prints the most recent note in full and replaces the earlier ones
with one line naming the count, the numbers, and a runnable command carrying the
ID. `note ID --list` is the index and `note ID --show N|N-M|all` is the text.
Plan 12.9 records it.

Measured across this store rather than argued: 89 tickets print exactly what they
printed before, 39 compact, and those 39 go from 45,491 words to 31,362. The
worst single ticket drops from 463 lines and 3,626 words to 181 and 1,319.

The rule is a count and not a size threshold, because a threshold makes the same
ticket print differently in two stores. A ticket with one note or none is
byte-identical to before, which was checked against a binary built from
origin/main rather than assumed, and that is the case worth protecting since most
tickets have one note.

`--json` is untouched, and that was proven the same way: the envelope from the
pre-change binary and from this one were compared across all 128 tickets in this
store and differ on none. body.notes is a string a consumer reads as one, so
truncating it would have been a break under 12.4 for no gain.

Two things worth carrying. The reading half went on `note` rather than on `show`
because `show ID` answers what a ticket is and `note ID --list` answers what was
written on it; those are two questions, not two formats of one. And the flags are
picked out before the shared runTextEntry body sees them, since that function
serves note, comment, plan and summary and treats every non-flag word as text to
append, so a --list it did not expect would have been appended as prose.

One test was wrong rather than the code, and the pre-change binary settled it:
`note ID -- TEXT --actor X` fails on main too, because everything after -- is
positional. The test now puts the flags first and says why.

Updating the 12.1 usage block also turned up that `--same-owner` was never added
to it when import gained the flag earlier today. That line is corrected here.

Left out deliberately. Folding a range of notes is
TKT-01M2HHTCKMN244FP2BRRSQY2PN, filed rather than built: compacting what show
prints destroys nothing and folding rewrites a log, and once notes stop flooding
output the remaining argument for folding is a superseded note rather than a long
one. Whether a numbered form of notes belongs in the JSON envelope is left to
section 15 beside the same question for import, because an envelope is a surface
12.4 says is settled with the user before it ships.
