---
schema: 1
id: TKT-01M1J2YR9D5242F6H7TPEV4M8K
title: Decide whether ready ranks by priority, and where priority sits against a deadline
type: spike
status: ready
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
updated_at: 2026-09-03T02:33:24Z
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
