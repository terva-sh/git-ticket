---
schema: 2
id: TKT-01M1W3WK41A9789XYMPD4KBMYX
title: Credit Backlog.md in NOTICE for the lifts already shipped
type: chore
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - policy
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-06T19:42:23Z
updated_at: 2026-09-06T19:59:41Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Credit Backlog.md in NOTICE. The ruling on this work is that a lift takes a
NOTICE entry whether it is code or an idea, and a file header only where
actual code was adapted. NOTICE currently names terva and nothing else, and
we have already borrowed three times.

### What was borrowed, and where it is credited today

The acceptance criteria and definition of done fields come from Backlog.md.
`docs/plan.md` section 16 says so, and the README says so at the paragraph
comparing the three prior projects.

The refreshable instruction block comes from section D of
`docs/review-backlog-md.md`. `git ticket instructions --write` "replaces an
earlier one in place" and `cli/instructions.md` opens with a
`<!-- git-ticket:begin -->` marker, which is the marker-delimited block that
section proposed.

Cross-branch visibility comes from the same section and shipped as
`git ticket refs`. Plan section 8 says it outright, under "Reading tickets
from other branches": the 30-day window "comes from the mechanism described
in section D of `docs/review-backlog-md.md`".

All three are credited in prose and none of them in NOTICE. That is the gap.

### The licence

Backlog.md is MIT, "Copyright (c) 2025 Backlog.md", read from
`MrLesk/Backlog.md@main`'s LICENSE on 2026-09-06. This project is MIT too, so
the regime is the same one the terva lift already follows.

None of the three is a code lift. Their implementation is TypeScript and
nothing here was adapted from it, so no file gets a header and the entry is
prose describing what the ideas were.

### What constrains the wording

`TestNoticeAgreesWithTheHeaders` reads NOTICE and collects only the lines
that begin `tui/` and end `.go`, then holds that set against the files whose
headers claim derivation. A prose entry naming no Go files is inert to it, so
this change cannot break that test. Confirm that rather than assume it.

### One question this does not settle

Section 16 also names `wedow/ticket`. Whether the same regime reaches every
project listed there, or only the ones something was taken from, is not
decided here and should not be decided by whoever happens to edit NOTICE
next.

## Acceptance criteria

- [x] NOTICE carries a Backlog.md entry reproducing its MIT copyright line, Copyright (c) 2025 Backlog.md
- [x] The entry names all three shipped lifts: the acceptance criteria and definition of done fields, the refreshable instruction block, and the cross-branch reads behind git ticket refs
- [x] No file header is added anywhere, because none of the three adapted code
- [x] TestNoticeAgreesWithTheHeaders still passes, confirmed by running it rather than by reading it

## Definition of done

- [x] just ci is green

## Notes

**agent:terva/mieli** at 2026-09-06T19:59:41Z

Done. NOTICE now credits Backlog.md.

The entry names all three lifts that had already shipped and says where each
one is recorded: the acceptance criteria and definition of done fields, in
plan section 16 and the README; the refreshable instruction block, where
`git ticket instructions --write` replaces an earlier block in place and
`cli/instructions.md` opens with a `git-ticket:begin` marker, which is
section D's marker-delimited design; and the cross-branch reads behind
`git ticket refs`, where plan section 8 records that the 30-day window comes
from the same section D. All three claims were checked against the tree
before they were written into a permanent file.

### The full license text is deliberately not reproduced

MIT's condition is that the copyright and permission notice travel with
"copies or substantial portions of the Software". This repository holds
neither. Backlog.md is TypeScript and nothing here is adapted from it. So
the credit is voluntary, and the entry says that outright rather than
implying a compliance obligation it does not carry. The copyright line is
reproduced, which is criterion 1, and the entry links the full text.

Reproducing the whole license would have looked safer and been less honest,
because it reads as discharging a condition that was never triggered, which
tells a later reader that code was copied.

### A second change the entry forced

NOTICE had one unlabeled entry and now has two, so each gained an underlined
header. That touches a part of the file this ticket had no other reason to
edit, and it is here because a reader cannot navigate two unlabeled prose
blocks.

### Evidence

`TestNoticeAgreesWithTheHeaders` passes, run rather than reasoned about,
which is criterion 4. It collects only the lines that begin `tui/` and end
`.go`, then pairs them against the files carrying a terva header, so a prose
entry naming no Go files touches neither set.

No file header was added anywhere, which is criterion 3. Two Go files
mention Backlog.md, `cli/instructions.go` and `ticket/query_test.go`, and
both are incidental prose inside comments rather than attribution headers.
Neither is modified on this branch.

The wedow/ticket question this ticket's description raised is still open. It
was not answered here and should not be answered by whoever edits NOTICE
next.

## Summary

Done. NOTICE credits Backlog.md.

Three lifts had already shipped with nothing in NOTICE: the acceptance
criteria and definition of done fields, the refreshable instruction block,
and the cross-branch reads behind `git ticket refs`. All three were credited
in `docs/plan.md` section 16 and the README, but not in the file that carries
third-party notices. The new entry names each one, says where it is recorded,
and reproduces the copyright line, Copyright (c) 2025 Backlog.md.

No file header was added, because none of the three adapted code. The full
MIT text is not reproduced either. That condition applies to copies and
substantial portions, and this repository contains neither, so the entry
states that the credit is voluntary rather than implying an obligation it
does not carry.

NOTICE also gained a header for each of its two entries, since a reader
cannot navigate two unlabeled prose blocks.

Whether the same regime reaches wedow/ticket, also named in plan section 16,
is still undecided and was deliberately not settled here.
