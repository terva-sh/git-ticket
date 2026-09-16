---
schema: 2
id: TKT-01M2NQVK39AD3TYKRPSD13430C
title: Generate a labels.md with a region the project writes itself
type: task
status: draft
status_reason: null
priority: normal
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
created_at: 2026-09-16T18:32:23Z
updated_at: 2026-09-16T18:32:23Z
created_by:
  id: agent:claude/t3code
  name: ""
updated_by:
  id: agent:claude/t3code
  name: ""
extensions: {}
---

## Description

Labels are barely used in this store, and the guess worth testing is that
nothing shows anybody how to use them. A generated `labels.md` beside `epics.md`
would name the vocabulary a store has, with a region the project writes itself
holding the examples and the grouping that make the vocabulary mean something
here rather than in general.

### What the store measures today

Counted across `draft/`, `tickets/`, `done/` and `archive/`: **137 tickets, 63
of them carrying no label at all.** Of the 74 that carry one, ten carry more
than one and exactly one carries three.

So the pattern is not "labels are ignored". It is that a ticket usually gets one
label or none, and almost never a combination. A document that showed
combinations would be showing something nobody does yet.

### The competing explanation, which has evidence behind it

AGENTS.md says of this repository's own store: the allowlist has no `tui` label,
"which is why the TUI tickets ship unlabeled". `config.yml` allows exactly
eight: `ci`, `claims`, `format`, `integration`, `mcp`, `policy`, `question`,
`release`.

That is a narrow, enforced vocabulary, and `update` has no `--label` flag, so
adding one is `--add-label` against a list somebody has to widen first. A ticket
with no fitting label ships unlabeled and the author moves on. If that is the
real cause, a document of examples will not move the number, because the problem
is not that people do not know how to label; it is that the label they want does
not exist and adding it is a second act of maintenance.

Both explanations can be true. Whichever it is, this document is cheap and the
measurement above is the baseline to compare against afterwards.

### The part that is not like epics.md

`epics.md` is generated whole. `renderEpicsIndex` writes every byte,
`epicsIndexStale` compares the file against a freshly rendered one, and
`check --fix` overwrites it. There is no region a person may edit and no
mechanism for one: any hand edit is reported as `epics_index_stale` and then
destroyed by the next `--fix`.

So the fenced region is the new thing here, and it is the whole design question.
The precedent is not `epics.md` but `instructions --write`, which writes a
generated block into a hand-maintained `AGENTS.md` between
`<!-- git-ticket:begin -->` and its matching end marker, leaving everything
outside alone. `labels.md` is the inverse arrangement: a generated file with a
hand-written island inside it, rather than a hand-written file with a generated
island.

That inversion is what needs deciding, and it makes the staleness rule harder
than `epics.md`'s. Today staleness is "these bytes are not those bytes". With a
preserved region it becomes "the generated part disagrees", which means the
comparison has to parse the file into regions first, and `check --fix` has to
rewrite one part while carrying the other through untouched. A `--fix` that got
that wrong would delete prose a person wrote, which is a worse failure than any
`--fix` can currently produce: every repair today moves or regenerates a derived
file, and none of them can lose an original.

### Worth considering, not prescribed

- Whether the generated half is the allowlist from `config.yml`, the labels
  actually in use with counts, or both, and what happens to a label in use that
  the allowlist does not have, which `check` already reports as
  `label_unknown`.
- Whether a store with an empty allowlist gets the file at all.
- Whether this composes with doctor rather than duplicating it. The first hard
  rule on TKT-01M2NJDAVHTXKEPJ0ZCAJP3QY0 is about label presence, and a rule
  about label vocabulary would be a rule, not a document. A document says what
  the labels mean; a rule says a ticket is missing one. They are different jobs
  and both may be wanted.
- Whether `init --instructions` should seed the hand-written region with a
  starting example, since an empty fenced region teaches nothing and is the
  state every store would begin in.

## Acceptance criteria

- [ ] labels.md is generated from the store's labels, beside epics.md
- [ ] A region of it is the project's own and survives regeneration
- [ ] check reports it stale only when the generated part disagrees, and --fix never destroys the written part
- [ ] Whether the generated half lists the allowlist, the labels in use, or both is decided and recorded
- [ ] Whether this belongs with a doctor rule about label vocabulary, rather than instead of one, is decided
- [ ] The 63-of-137 baseline is remeasured afterwards, so the document is known to have worked or not
