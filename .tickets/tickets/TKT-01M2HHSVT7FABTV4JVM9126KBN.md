---
schema: 2
id: TKT-01M2HHSVT7FABTV4JVM9126KBN
title: Compact the note history git ticket show prints
type: task
status: in-progress
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
claim:
  actor: agent:terva/compact-notes
  branch: feat/compact-notes
  worktree: /home/sothr/workspace/git.local.sothr.com/terva-sh/git-ticket
  commit: ee39eb25ff563826ee9c27725c3fca98d7d85492
  claimed_at: 2026-09-15T03:30:14Z
  expires_at: null
archive: null
created_at: 2026-09-15T03:29:37Z
updated_at: 2026-09-15T03:30:14Z
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

- [ ] show prints the most recent note in full and replaces every earlier one with a single line naming how many there are and their numbers
- [ ] That line names the command that retrieves them, so the reader needs nothing else to get the text back
- [ ] note ID --list prints every note as number, actor, instant and opening line
- [ ] note ID --show N and --show N-M print the full text of those notes
- [ ] A ticket with one note or none prints exactly what it printed before, with no index line added
- [ ] show --json is unchanged, proven by comparing the envelope against the pre-change binary
- [ ] Measured on this store's worst ticket: show drops from 3,626 words to a small fraction, with the drop recorded

## Definition of done

- [ ] just ci is green and check --strict passes on this store
