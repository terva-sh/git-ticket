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
updated_at: 2026-09-12T00:12:07Z
created_by:
  id: agent:terva/mieli
  name: Mieli
updated_by:
  id: agent:terva/mieli
  name: Mieli
extensions: {}
---

## Description

`exportCommitBody` in `ticket/export.go` writes one line per ticket as `"  %s  %s  %s\n"` over the ID, the status and the title. Two literal spaces between fields means the title starts at a different column on every row, because a status is between 5 and 11 characters.

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
