---
schema: 2
id: TKT-01M2NJDAVHTXKEPJ0ZCAJP3QY0
title: "Ship doctor's first hard and soft rules: label presence and order"
type: task
status: done
status_reason: "Worked straight from draft at the user's request, without passing through ready or in-progress. The work is this branch: ticket/doctorrules.go and ticket/doctorrules_test.go, committed with this ticket's ID."
priority: normal
due_on: null
labels: []
assignees: []
milestone: null
parent: null
origin: TKT-01M2NHGHTZ4PWHBKRHE43XJG8D
dependencies:
  - TKT-01M2NHGHTZ4PWHBKRHE43XJG8D
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-16T16:57:13Z
updated_at: 2026-09-16T19:49:58Z
created_by:
  id: agent:claude/t3code
  name: ""
updated_by:
  id: agent:claude/t3code
  name: ""
extensions: {}
---

## Description

The first two hygiene rules `git ticket doctor` ships, carved out of
TKT-01M2NHGHTZ4PWHBKRHE43XJG8D so that the framework ticket settles identifiers
and configuration without also arguing about vocabulary.

These two exist as a pair on purpose. They are the proof that the hard and soft
levels are a real distinction rather than two words for severity, because they
are the same subject at both levels.

- **Hard.** Every ticket carries at least one label. True or false for a given
  ticket, so the command may say so without qualification.
- **Soft.** A ticket's labels are ordered with the most descriptive first.
  Nothing mechanical knows which of `auth` and `ui` describes a ticket better,
  so the finding has to read as a question and must never be what makes the
  command exit non-zero.

### Why label order is worth a rule at all

`git-ticket-canvas` renders `labels.slice(0, 2)` on a card, three when the cards
are compact, and collapses the rest behind a `+N` disclosure. Label order
therefore decides which labels somebody sees while glancing over a board, and a
ticket whose most descriptive label sits fourth reads as something else
entirely.

The canvas does not colour a card by its first label; the card's colour comes
from its status. That was claimed while filing the parent and corrected there.
If the canvas ever does colour by label, this rule gains a second reason rather
than a different one.

### What the hard rule must not repeat

A store whose `config.yml` enforces a label allowlist already gets
`label_unknown` from `check` for a label outside the set. This rule is about
presence, not membership, and should not restate what `check` already reports.

### Depends on the framework

Both rules need the identifier and configuration decisions from
TKT-01M2NHGHTZ4PWHBKRHE43XJG8D before they can be written, because a shipped
rule's identifier is a name somebody puts in a config file and cannot be
renamed casually afterwards.

## Acceptance criteria

- [x] Every ticket carries at least one label ships as the first hard rule
- [x] Labels are ordered most-descriptive first ships as the first soft rule
- [x] The hard rule reports presence only, and does not restate check's label_unknown
- [x] The level each rule ships at is chosen against what it does to a real store, not in the abstract
- [x] The soft finding reads as a question rather than a verdict, and does not make the run fail

## Notes

**agent:claude/t3code** at 2026-09-16T17:31:11Z

Both rules were run against this store before either was built, because a
default is an opinion and an opinion should know what it does to the store that
ships it. 137 tickets, counted across `draft/`, `tickets/`, `done/` and
`archive/`.

**The hard rule would fire on 63 of 137 tickets, 46% of the store, on the day it
ships.** That is not by itself an argument against it. It is an argument that
the level it ships at matters more than the description assumed, because a
finding on half the store is one a reader learns to skim, which is the exact
failure the second note on the framework ticket warns about.

There is a second-order problem underneath it. AGENTS.md says this store's
allowlist has no `tui` label, "which is why the TUI tickets ship unlabeled". So
a narrow enforced allowlist pushes tickets toward carrying no label, and a rule
demanding one then argues with the allowlist rather than with the person. The
two config features interact, and this ticket and the framework both treat them
as independent. Worth deciding whether the hard rule should say something
different in a store whose allowlist is enforced and small.

**The soft rule has almost no subjects here. Ten tickets carry more than one
label and exactly one carries three.** The rule's justification is that the
canvas renders `labels.slice(0, 2)` and hides the rest behind a `+N` disclosure,
so order decides what a person sees at a glance. With a maximum of three labels
on one ticket, that disclosure essentially never appears in this store, and
label order changes what is visible on exactly nothing.

The rule may still be right for a store that labels heavily. This store is not
evidence for it, and it is the flagship soft rule: the one whose job is to prove
that hard and soft is a real distinction rather than two words for severity. A
flagship that is a no-op in its own repository proves the distinction to nobody.

Neither finding changes the plan yet, per the decision taken with the user: keep
both rules, record what they do here, and let the level each ships at absorb it.
But whoever builds this should look at the numbers again first, because they are
the only measurement anyone has of what these defaults actually say.

**agent:claude/t3code** at 2026-09-16T18:16:49Z

The exit-status decision on TKT-01M2NHGHTZ4PWHBKRHE43XJG8D falsified this
ticket's third criterion the same day it was written. It said, verbatim:

- [ ] The soft finding reads as a question rather than a verdict, and does not
  affect the exit code

`doctor --strict` now exits by a graded informational bucket, and the grade is
the worst level that fired, so a soft finding does affect the exit code: it is
what puts a run in the soft-only grade rather than the clean one. What a soft
finding still may not do is make the run fail.

The first half of the criterion is untouched. The soft rule reads as a question
either way, and that was never about the exit status.

**agent:claude/t3code** at 2026-09-16T18:37:24Z

Both rules ship. `label_missing` is hard, `label_order` is soft, and
`DefaultRules()` is no longer empty, so the framework's fourth criterion stops
being vacuous.

**The measurement recorded on this ticket was wrong, and the corrected number is
better news.** The note above says the hard rule would fire on 63 of 137. It
fires on **11**, because `Doctor` reads the open set and not every file: 25 open
tickets, 11 of them unlabelled, while the other 54 unlabelled ones are done or
archived.

That scope was implicit in a bare `Filter{}` when the framework was built, which
is not good enough for a decision this size, so it is now stated and commented
in `doctor.go` and held by `TestDoctorReadsTheOpenSet`. Hygiene is about a
ticket somebody might pick up, and nobody picks up an archived one. The
difference is the whole feature: 11 findings a person can act on today against
65 mostly about work finished months ago, and a report that large is one a
reader learns to skim.

**The fourth criterion could not be satisfied the way it was written, and this
is the honest version.** It asks that the level each rule ships at be chosen
against what it does to a real store. But the framework settled that level means
whether a rule can be settled mechanically, not how severe or how noisy it is.
Whether a ticket has a label is objectively checkable, so `label_missing` is
hard no matter how often it fires; moving it to soft to quiet it would be
claiming the tool cannot tell, which is false. Level is not a dial.

What the measurement did decide, which is the real content of the criterion:

- **Scope.** The open set rather than every file, on the numbers above.
- **The soft rule's threshold.** It fires only when a label is actually hidden,
  which with a card showing two means three or more labels. A ticket with two
  labels has no ordering choice a reader can see, and a rule that fired there
  would be asking about a decision that does not exist.

**The soft rule fires exactly once in this store, as predicted.** That was
flagged before it was built and it is still true: one ticket carries three
labels. It is not a design fault. `visible` is a parameter, so a store whose
board renders compact cards sets it to three and the rule goes quiet, and a
store that labels heavily gets more from it than this one does. That makes the
near-silence a fact about this store rather than about the rule, which is the
best that can be done without inventing subjects.

It is also the first shipped rule to read a parameter, so the framework's
params plumbing now has a user rather than only a test.

**The remedy was verified rather than guessed.** Labels keep insertion order:
`addUnique` appends and nothing sorts them, confirmed against a scratch store.
There is no reorder flag, but `update` applies removals before additions within
one write, so `--remove-label NAME --add-label NAME` moves a label to the end in
a single command. That was measured before it was put in a remedy a person is
told to run.

**TKT-01M2NHGHTZ4PWHBKRHE43XJG8D's third criterion is now ticked**, on its
ticket. It was left unticked when the framework shipped because "a soft finding
reads as a question rather than a verdict" needed a soft rule to be true of
anything. `label_order` is that rule, and the assertion that its message ends in
a question mark is in `TestLabelOrderFiresOnlyWhenALabelIsHidden`.

**agent:claude/t3code** at 2026-09-16T18:37:44Z

draft to done: Worked straight from draft at the user's request, without passing through ready or in-progress. The work is this branch: ticket/doctorrules.go and ticket/doctorrules_test.go, committed with this ticket's ID.

**agent:claude/t3code** at 2026-09-16T19:49:58Z

The soft rule's threshold moved from "a card hides a label" to "the order is a
choice", which is two or more labels. Decided with the user, against the
alternative of recording it and revisiting when canvas ships.

**The trigger.** git-ticket-canvas TKT-01M26XAVP3 ("Give labels project-wide
colors and inherit them on cards") rules that a card body inherits the colour of
the ticket's *first* label and explicitly does not scan later ones. That makes
order visible at two labels, where the old threshold was silent. Its split-out
sibling TKT-01M27EPDKKW6HGKNKS7A98EQER states the convention this rule assumes,
in its own words: "the first label is the ticket's primary label". Both are
drafts, so the rule now fires ahead of the reason being true, which was the
argument against and was overruled deliberately.

**Two criteria were reworded.** Verbatim, as they stood on this ticket:

- [ ] Labels are ordered most-descriptive first ships as the first soft rule

That one is unchanged in substance. What changed is the test names beneath it:
`TestLabelOrderFiresOnlyWhenALabelIsHidden` became
`TestLabelOrderAsksWhereverTheOrderIsAChoice`, and
`TestLabelOrderTakesItsThresholdFromParams` became
`TestLabelOrderNamesWhatACardHides`, because `visible` no longer decides whether
the rule fires, only what the finding may claim.

**The cost was measured across three real stores, before and after.**

| store | open | soft before | soft after |
|---|---|---|---|
| git-ticket | 27 | 1 | 8 |
| terva | 114 | 0 | 48 |
| git-ticket-canvas | 30 | 10 | 19 |

terva is the one that matters. It went from zero to forty-eight, and it is the
store doing labels best: `area/` then `scope/` on nearly every ticket, with
`CONVENTIONS.md` settling which leads. Every one of those 48 findings asks a
question terva answered once, in writing, which is precisely the noise the
framework's second note warned turns a report into something people skim.

**So `min` is now a parameter.** The old threshold was reachable through
`visible`; the new one was hardcoded, which took away the only narrow answer and
left "disable the rule" as the blunt one. A store whose convention settles the
dimension order sets `min: 3` and keeps the rule for the tickets where a real
choice remains. Default stays at 2, as decided.

That gap was mine: moving a threshold into a constant removed a knob that had
been configurable, and a rule system whose whole argument is that stores differ
should not lose one silently.

## Summary

label_missing ships hard and label_order soft, so DefaultRules is no longer empty. The hard rule reports presence only and leaves membership to check's label_unknown. The soft rule fires only when a card actually hides a label, which is three or more against a default of two, and reads that threshold from params, making it the first shipped rule to use them. On this store: 11 hard findings and 1 soft. The 63-of-137 baseline recorded earlier was wrong; doctor reads the open set, so it is 11 of 25 open tickets and the other 54 unlabelled ones are done or archived.
