---
schema: 2
id: TKT-01M298KXYN12CRTK78F7F5RAQ1
title: Build git ticket export --since REF
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
dependencies:
  - TKT-01M298KEGY8KV17MCFN9V1A4WG
blocks_on: none
references:
  - ref: plan:12.8
    path: docs/plan.md
  - ref: plan:7.4
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-11T22:15:10Z
updated_at: 2026-09-11T23:06:09Z
created_by:
  id: agent:terva/mieli
  name: Mieli
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

`export` takes ticket IDs and nothing else. The commonest reason to export is a
branch that filed a ticket and fixed the thing, and nothing helps you get from
that branch to the set of IDs.

### What it costs today

Measured end to end in a scratch repository:

```sh
IDS=$(git diff --name-only main...HEAD -- .tickets/ | xargs -n1 basename | sed 's/.md//')
git ticket export $IDS --out ./handoff
git format-patch --start-number 2 -o ./handoff main..HEAD
```

It works, and it has four problems.

The discovery step is a shell incantation rather than a command, and
`xargs -n1 basename | sed` is where a person gives up and an agent invents
something subtly wrong.

The two ranges differ by one character. `main...HEAD` for the tickets and
`main..HEAD` for the code, both correct, for different reasons, with no signpost
saying so.

It takes every ticket the branch added. The test branch filed "Crash on empty
input" and "Follow-up found on the way", and the second is exactly what a sender
does not want to hand over. Selection is the real work.

There is no packaging. The bundle this feature arrived in carried a tarball,
`SHA256SUMS` and an `APPLYING.md`, all made by hand.

### This needs no change to the 7.4 table

Worth stating plainly, because it looks blocked and is not. `treeAt(ref)` in
`ticket/crossbranch.go` already lists the ticket files on a ref, and it does it
with `ls-tree`, which is already a row in the table. Two calls, `HEAD` minus the
named ref, is exactly the set. No new git command, no review to force.

It does move one sentence in 12.8, which says neither command runs git. That
sentence is a promise about reads rather than a count, so it needs rewording
rather than reversing. Say so in the same change.

### Why it waits on the refactor

The set is a library question, not a flag question, and a host that wants to
hand work upstream wants the same answer. It belongs beside `treeAt` in the
`ticket` package, which is where the dependency points.

### Open, and worth settling before building

Whether `--since` selects or merely proposes. Exporting every ticket a branch
touched is wrong often enough that a flag which silently does it will send the
wrong thing. Printing the set for the sender to confirm, or taking IDs to
exclude, are both plausible, and the choice is a CLI contract under 12.4.

## Acceptance criteria

- [ ] The ticket package answers which tickets a ref adds relative to another, using ls-tree alone
- [ ] export --since REF exports that set without a shell pipeline
- [ ] The three-dot and two-dot range trap is gone from the documented workflow, or explained where a reader meets it
- [ ] Settled and recorded: whether --since selects outright or proposes a set the sender confirms
- [ ] Plan 12.8's "neither command runs git" sentence is reworded to be about reads rather than a count
- [ ] A test drives a real branch that adds two tickets and exports one of them
