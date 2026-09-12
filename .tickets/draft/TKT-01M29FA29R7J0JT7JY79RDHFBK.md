---
schema: 2
id: TKT-01M29FA29R7J0JT7JY79RDHFBK
title: Align the status column in an export's diffstat body
type: chore
status: draft
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
updated_at: 2026-09-12T00:42:55Z
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

- [ ] Titles start at the same column on every row of an export's commit body
- [ ] The padding follows the widest status in the set rather than a fixed width
- [ ] The before-image is regenerated and its diff shows the alignment change alone
- [ ] git am still applies an export whose body changed, proven against real git rather than inferred

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
