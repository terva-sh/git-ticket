---
schema: 2
id: TKT-01M29FA29R7J0JT7JY79RDHFBK
title: Align the status column in an export's diffstat body
type: chore
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - format
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references:
  - ref: plan:12.8
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-12T00:12:07Z
updated_at: 2026-09-12T00:52:48Z
created_by:
  id: agent:terva/mieli
  name: Mieli
updated_by:
  id: agent:terva/mieli
  name: Mieli
extensions: {}
---

## Description

`exportCommitBody` in `ticket/export.go` writes one line per ticket as `"  %s  %s  %s\n"` over the ID, the status and the title. Two literal spaces between fields means the title starts at a different column on every row, because a status is between 4 and 11 characters.

The pinned before-image shows it:

    TKT-01K3ZZ67Q0PT427VFD1F4WFWSH  blocked  Can the daemon survive a provider outage
    TKT-01K3ZZ82A0YPGSE71EY0N5NCH6  review  Ship the token refresh work

This is cosmetic. The text sits in the commit body of the patch that carries the tickets, so a person reads it in `git log` on the receiving side and nothing parses it.

It was found while pinning the before-image for TKT-01M298KEGY8KV17MCFN9V1A4WG and deliberately left alone: fixing it inside a move that had to prove byte-identity would have made the artifact differ for a reason unrelated to the move.

### What this costs

The before-image changes, which is the point of doing it as its own ticket. `cli/testdata/export-before-image/0001-tickets.patch` has to be regenerated, and that file exists specifically to make an unreviewed change to the artifact impossible. So the regeneration is the reviewable act here, and the diff should show the alignment and nothing else.

Pad the status to the width of the widest status in the set rather than to a constant, so a set of short statuses does not carry a trench of spaces.

## Acceptance criteria

- [x] Titles start at the same column on every row of an export's commit body
- [x] The padding follows the widest status in the set rather than a fixed width
- [x] The before-image is regenerated and its diff shows the alignment change alone
- [x] git am still applies an export whose body changed, proven against real git rather than inferred

## Notes

**agent:terva/mieli** at 2026-09-12T00:42:42Z

Groomed. The defect and the fix hold. One number in the description is wrong,
and the blast radius is smaller than the description leaves open.

`exportCommitBody` is at `ticket/export.go:93` and line 98 is
`fmt.Fprintf(&b, "  %s  %s  %s\n", t.ID, t.Status, t.Title)`, two literal spaces
and no padding, as filed. The before-image shows it at lines 9 and 10 of
`cli/testdata/export-before-image/0001-tickets.patch`.

### Correction: a status is 4 to 11 characters, not 5 to 11

The description says "a status is between 5 and 11 characters". `done` is four.
The seven are draft 5, ready 5, in-progress 11, blocked 7, review 6, done 4,
archived 8.

The fix does not change, because the ticket already asks for padding to the
widest status in the set rather than to a constant, and that rule is indifferent
to where the floor sits. Only the stated range was wrong. The original wording
is quoted above so a later reader does not have to reach for `git log`.

### The blast radius is one file, measured

`0001-tickets.patch` is the only artifact carrying these rows. The cover letter
prints `TKT-...  [spike, blocked, normal]`, a bracketed triple that is
self-delimiting, so it has no alignment to fix and does not need regenerating.
No test hardcodes the two-space rows: a grep for them across `*_test.go` returns
nothing, and `exportCommitBody` is named in no file but `ticket/export.go`. So
the reviewable diff is one file and the change is the alignment alone, which is
exactly what the third criterion asks for.

### The trigger fired

This was deferred for one stated reason: fixing it inside TKT-01M298KEG would
have made the before-image differ for a cause unrelated to the move that had to
prove byte-identity. That move is done and released in v0.16.0, so the reason to
wait is discharged. Nothing else gates it, and it carries no open decision.

**agent:terva/mieli** at 2026-09-12T00:47:27Z

draft to ready: Promoted by the user. The grooming found its one blocker discharged: it was deferred only to keep the export before-image byte-identical through the interchange move, which shipped in v0.16.0. No open decision, and a one-file blast radius.

**agent:terva/mieli** at 2026-09-12T00:52:28Z

Built on `fix/export-body-alignment`. All four criteria hold, and the fix is
wider than the description prescribed.

### The deviation: the ID column is padded too

The description ends "Pad the status to the width of the widest status in the
set". That is necessary and not sufficient, and the first criterion is the one
that governs, because it asks for titles on a single column rather than for a
particular padding.

An ID is `SERIES-ULID`. `SeriesPattern` is `^[A-Z][A-Z0-9]{1,7}$` and
`series add` appends to the list, so one store can declare several prefixes of
different lengths and an ID runs 29 to 35 characters. In a multi-series export
the ID column is ragged too, and padding the status alone leaves the titles
exactly where they started. So `exportCommitBody` pads both columns to the
widest value in the set.

That supersedes the final paragraph of the description, which is incomplete
rather than wrong. The before-image could not have caught it: both of its
tickets are `TKT`, so their IDs are the same width and a status-only fix passes
it.

### Falsification, run rather than assumed

`TestExportCommitBodyAlignsEveryTitle` drives series `AB`, `LONGSERI` and `TKT`
against statuses `done`, `in-progress` and `review`. Reverting the format string
to the unpadded form turns it red with the titles at columns 39, 52 and 42;
restoring it turns it green. `TestExportCommitBodyPadsToTheSetNotAConstant` is
the other half, holding a uniform set to exactly two spaces so nobody pads to
the widest status that exists and leaves a trench after `done`.

### How the before-image was regenerated

By hand, one line, because the golden test deliberately has no `-update` flag
and adding one to regenerate it would defeat what the file is for. The hand edit
was not assumed correct: `TestExportMatchesThePinnedBeforeImage` compares bytes,
so it passes only if the file equals what the new code emits. `git diff` over
the artifact is one insertion and one deletion, `review  Ship` becoming
`review   Ship`. `0000-cover-letter.txt` is untouched, as the grooming predicted.

### git am, against real git

git 2.43.0. Two tickets ragged on both axes exported from a scratch store into a
fresh repository that has never heard of git-ticket: `git am` exited 0 and both
files landed. The half worth checking is that the padding survives `mailinfo`,
and it does. `git log -1 --pretty=%B` on the receiving side puts both titles at
column 52, and `cat -A` shows no row ending in spaces.

**agent:terva/mieli** at 2026-09-12T00:52:44Z

Task worklog for this ticket, from the session task board.

### Tasks

- [x] task-9 Titles start at the same column on every row of an export's commit body — TestExportCommitBodyAlignsEveryTitle in ticket/export_test.go drives a ragged set (series AB, LONGSERI and TKT; statuses done, in-progress, review) and asserts every title starts at the same column, plus that no row ends in spaces. Falsified rather than assumed: reverting the format string to the unpadded form turns it red with titles at columns 39, 52 and 42, and restoring it turns it green. The fix pads the ID column as well as the status, because an ID is 29 to 35 characters when a store decl…
- [x] task-10 The padding follows the widest status in the set rather than a fixed width — exportCommitBody now computes idWidth and statusWidth as the max over the set and formats with %-*s, rather than a constant. Proven by the golden test: "review" (6) padded to the width of "blocked" (7), producing one extra space, with no trench where statuses are short.
- [x] task-11 The before-image is regenerated and its diff shows the alignment change alone — cli/testdata/export-before-image/0001-tickets.patch regenerated: git diff shows 1 insertion, 1 deletion, the single line "review Ship" becoming "review Ship". No other byte of either artifact moved, and 0000-cover-letter.txt is untouched because it prints a self-delimiting [spike, blocked, normal] form. The hand edit was not assumed correct: TestExportMatchesThePinnedBeforeImage compares bytes and passes, which is only possible if the file equals what the new code generates.
- [x] task-12 git am still applies an export whose body changed, proven against real git rather than inferred — Real git 2.43.0, not inferred. Built binary exported two tickets with both columns ragged (TKT- 29 chars / LONGSERI- 35 chars; ready 5 / in-progress 11) from a scratch store into a fresh repo that has never heard of git-ticket. git am exited 0, "Applying: Tickets: 2 from another store", and both .tickets/tickets/*.md files landed. The padded body survived mailinfo intact: git log -1 --pretty=%B shows both titles at column 52 (ALIGNED), and cat -A confirms no row ends in spaces.

## Summary

Done on `fix/export-body-alignment`. `exportCommitBody` pads both leading
columns to the widest value in the set, so the titles in an export's commit body
start on one column.

The fix is wider than the description asked for. It prescribed padding the
status; an ID is 29 to 35 characters when a store declares several series, so
the ID column is ragged too and padding the status alone would have aligned
nothing. The first criterion is the outcome and it governs. That is recorded in
a note, which supersedes the description's last paragraph as incomplete rather
than wrong.

The before-image moved by exactly one line, `review  Ship` becoming
`review   Ship`, which is the alignment and nothing else. It was regenerated by
hand, because the golden test has no `-update` flag by design, and the hand edit
is proven by that test comparing bytes rather than by inspection. The cover
letter needed no regeneration, as the grooming predicted, because it prints a
self-delimiting `[spike, blocked, normal]`.

Two new tests carry the property the before-image cannot. Its two tickets are
both `TKT` and the same width, so a status-only fix would pass it.
`TestExportCommitBodyAlignsEveryTitle` drives three series against three status
widths and was falsified by reverting the format string, which puts the titles
at columns 39, 52 and 42. `TestExportCommitBodyPadsToTheSetNotAConstant` keeps
the padding tied to the set, so no short-status export carries a trench.

`git am` was proven against real git 2.43.0 rather than inferred: an export
ragged on both axes applied clean into a repository that has never heard of
git-ticket, and the alignment survived `mailinfo` with both titles at column 52
on the receiving side.
