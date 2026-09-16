---
schema: 2
id: TKT-01M2P0FWJ1BEDGV3603X5YKR5G
title: label_order fires on nearly every ticket, so it guides nothing
type: bug
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - area/doctor
assignees: []
milestone: null
parent: null
origin: TKT-01M2NJDAVHTXKEPJ0ZCAJP3QY0
dependencies: []
blocks_on: none
references:
  - ref: ticket:report
    path: .tickets/draft/TKT-01M2NZE9Z4G033QJDFG62N4Y9D.md
claim: null
archive: null
created_at: 2026-09-16T21:03:17Z
updated_at: 2026-09-16T21:14:14Z
created_by:
  id: agent:claude/t3code
  name: ""
updated_by:
  id: agent:claude/t3code
  name: ""
extensions: {}
---

## Description

`label_order` shipped in v0.19.0 firing on any ticket with two or more labels. Measured against the stores that use it, that is almost every ticket, and a rule that fires on everything carries no information.

| Store | Open | Multi-label | Fires | % of open |
|---|---|---|---|---|
| terva | 121 | 120 | 120 | 99% |
| ketju | 131 | 131 | 131 | 100% |
| git-ticket-canvas | 30 | 19 | 19 | 63% |
| git-ticket | 26 | 8 | 8 | 30% |

### The figure on record is wrong

`AGENTS.md` says moving the default from 3 to 2 took terva "from 0 soft findings to 48". That 48 was measured from terva's working tree, which sits on another branch 68 commits behind `origin/sothr-main`. The real number was always 120. The decision to move the threshold was taken against a figure less than half the true one.

### What to do instead

The stores already answer the question the rule asks. terva leads with `area/` on 116 of 120 multi-label tickets; ketju leads with `area:` on 130 of 131. The tool can read that convention off the store and report only the tickets that break it, rather than asking every ticket a question the store settled once.

Fire when a ticket's leading label *dimension* disagrees with the dominant leading dimension in the same store, and stay silent unless the store has enough multi-label tickets to infer a convention from and the convention is strong enough to be one.

Expected: terva 4, ketju 1, canvas 0, git-ticket 1.

Canvas producing zero is correct rather than a gap. It has no dimensional convention, so there is nothing objective to say, and silence beats 19 unanswerable questions.

### Already stale prose to fix with it

Two passages describe the pre-v0.19.0 visibility framing and were wrong the day v0.19.0 shipped: `AGENTS.md` in the store-conventions section, and `cli/instructions.md`, which ships to other projects in the agent block.

## Acceptance criteria

- [x] label_order reports only tickets whose leading label dimension differs from the store's dominant one.
- [x] The rule stays silent when the store has too few multi-label tickets to infer a convention, or when no dimension is dominant enough to be one.
- [x] Both thresholds are parameters with defaults justified by measurement across the four real stores.
- [x] Measured counts against terva, ketju, git-ticket-canvas and git-ticket are recorded, with terva read from origin/sothr-main rather than its stale working tree.
- [x] The wrong terva figure in AGENTS.md is corrected, and the two already-stale doctor passages are brought in line with what the rule does.
- [x] Whether two new config params make this a minor rather than a patch is decided against plan 12.4 and recorded.

## Summary

label_order now reads the store's convention instead of asserting one. It takes
the dimension each ticket leads with, the text before the first `/` or `:`,
finds the one the store leads with most often, and reports only the tickets that
disagree. Measured: terva 120 to 4, ketju 131 to 1, git-ticket-canvas 19 to 0,
git-ticket 8 to 1. terva's four are the live-test and flake tickets its own
maintainer had already found by hand, which is the evidence that the inferred
convention matches the written one.

Two guards stop it inventing a convention. `sample`, default 5, is how many
ordered tickets it takes before a majority counts as a practice; `confidence`,
default 0.8, is how dominant the leading dimension must be. Below either it says
nothing. The defaults were measured rather than picked: terva sits at 97% and
ketju at 99%, so any threshold from 0.70 to 0.95 gives both the same answer, and
only git-ticket's own store at 88% over eight tickets sits near a boundary.
Canvas gets zero because it has no dimensional convention, which is the right
answer rather than a gap.

Leading with undimensioned labels counts as a convention too, so a store that
never prefixes still hears about the one ticket that suddenly does.

Recorded in plan 12.4 as a decision rather than a break: the rule ID, the
doctor-report kind and the existing params are unchanged and the two new ones
add beside them, and config params are not Go API, so this is a patch. What does
change is the finding count, steeply, and the release notes have to say it is
the rule getting quieter rather than the stores getting cleaner.

The wrong figure that produced the original threshold is corrected wherever it
was cited. AGENTS.md said terva went from 0 findings to 48; that came from a
working tree 68 commits behind its ref and the real number was 120. Two doctor
passages that were already stale when v0.19.0 shipped, in AGENTS.md and in
cli/instructions.md which ships to other projects, now describe what the rule
actually does.

Not done, and TKT-01M2NZE9Z4G033QJDFG62N4Y9D stays open for it: the explicit
`order: [area/, init/, scope/]` parameter that store asked for. This infers one
dominant leading dimension rather than reading a declared total order, so it
cannot express a sequence or judge the labels after the first, and it is silent
on the 34 terva tickets carrying two area/ labels where the alphabet chose which
leads.
