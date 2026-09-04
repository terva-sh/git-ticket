---
schema: 1
id: TKT-01M1J2YR9D5242F6H7TPEV4M8K
title: Decide whether ready ranks by priority, and how it weighs a deadline
type: spike
status: done
status_reason: null
priority: low
due_on: null
labels:
  - question
assignees: []
milestone: null
parent: null
dependencies:
  - TKT-01M1HPCVRK1989NDNR9PJS36S4
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-02T22:13:41Z
updated_at: 2026-09-04T17:47:57Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Raised while settling the deadline field on TKT-01M1HPCVRK1989NDNR9PJS36S4. That ticket adds one sort key to `ready` and this is the question it made visible.

`ready` sorts by ID, which sorts by creation time, so the oldest startable ticket is first. Priority is not a key. An `urgent` ticket filed today places below a `low` one filed last week, and nothing about the output says so.

That is defensible today because `ready` makes no claim to rank. It answers what is startable and leaves the choice to the reader. Adding the deadline key changes the frame: once `ready` orders by one property of the ticket, the order reads as a recommendation, and a reader who sees a deadline honoured will assume priority is honoured too.

### What has to be decided

Whether `ready` ranks by priority at all, and if it does, where priority sits against `due_on`.

The two orderings disagree in the case that matters. A `low` ticket due in a week against an `urgent` with no date: priority-first puts the urgent one on top and the deadline passes, deadline-first puts the dated one on top and the urgent work waits on something nobody called important.

### What not to do

Do not invent an urgency score that folds the two into one number. A weight nobody can see turns an ordering a person can predict into one they have to reverse-engineer, and the weights become a configuration question the store then has to validate.

### Prior art in this repository

The deadline settlement rejected a flag on the grounds that a flag nobody learns is not a feature. That argument applies here too, so `--sort priority` is not the easy way out of choosing.

## Summary

Settled and shipped. `ready` ranks by priority, then `due_on`, then the ID, and `--sort` gains `priority` so the field that ranks one command can order the other.

Both orderings are wrong somewhere, and the case that decides it is the one this ticket named: a `low` ticket due next week against an `urgent` with no date. What settled it is which wrong answer a person can correct honestly. Under deadline-first the only way to lift the urgent ticket is to invent a date for it, which corrupts a field to move a sort. Under priority-first the way to lift the dated ticket is to raise its priority, which is what the field means. An ordering should be gamed by telling the truth.

The other half is that the plan already says undated means never due, and never is the far end of the scale. Deadline-first has to defend a never-due `urgent` sorting below a `low` due in two years, and it cannot.

### What the store added to the argument

Across 53 tickets nothing carried a `due_on` at all, while 23 carried a priority other than the default. The key that shipped ranked nothing and fell through to the ID tiebreak on every row, while the field people actually set was not a key. A command whose own documentation says its ranking is part of what it answers was ordering by filing date. That was stronger evidence than the ticket had.

The urgency score stays ruled out, unchanged. A weight nobody can see turns an order a person can predict into one they have to reverse-engineer.

### Consistency

Section 8 says one field has one order, so `--sort priority` gives exactly what `ready` gives, with the readiness filter in front of it. `SortByPriority` and `SortByDueOn` share one `dueOnLess` comparator for the same reason: a deadline means the same thing wherever it is a key. `list` with no flag is unchanged and still answers in ID order, because a list reports and only reorders when asked.

An unrecognized priority sorts below `low`, which `slices.Index` gives for free by returning -1. A value the format does not define must not outrank one somebody set on purpose.

### Verification

Three mutations, each confirmed red and reverted. The direction flip fails four tests. Dropping the deadline tiebreak fails the pre-existing deadline test too, which proves the secondary key is live rather than decorative. The one worth keeping is `Ready` dispatching to `SortByDueOn`: every comparator test still passed while both store-level tests failed, which is the shape of bug a comparator test structurally cannot see.
