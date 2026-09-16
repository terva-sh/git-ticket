---
schema: 2
id: TKT-01M2NJDAVHTXKEPJ0ZCAJP3QY0
title: "Ship doctor's first hard and soft rules: label presence and order"
type: task
status: draft
status_reason: null
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
updated_at: 2026-09-16T16:58:00Z
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

- [ ] Every ticket carries at least one label ships as the first hard rule
- [ ] Labels are ordered most-descriptive first ships as the first soft rule
- [ ] The soft finding reads as a question rather than a verdict, and does not affect the exit code
- [ ] The hard rule reports presence only, and does not restate check's label_unknown
