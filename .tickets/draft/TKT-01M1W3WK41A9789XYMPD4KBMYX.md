---
schema: 2
id: TKT-01M1W3WK41A9789XYMPD4KBMYX
title: Credit Backlog.md in NOTICE for the lifts already shipped
type: chore
status: draft
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
updated_at: 2026-09-06T19:42:23Z
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

- [ ] NOTICE carries a Backlog.md entry reproducing its MIT copyright line, Copyright (c) 2025 Backlog.md
- [ ] The entry names all three shipped lifts: the acceptance criteria and definition of done fields, the refreshable instruction block, and the cross-branch reads behind git ticket refs
- [ ] No file header is added anywhere, because none of the three adapted code
- [ ] TestNoticeAgreesWithTheHeaders still passes, confirmed by running it rather than by reading it

## Definition of done

- [ ] just ci is green
