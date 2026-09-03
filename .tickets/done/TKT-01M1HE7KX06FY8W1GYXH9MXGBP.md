---
schema: 1
id: TKT-01M1HE7KX06FY8W1GYXH9MXGBP
title: Decide whether git-ticket ships a merge driver for ticket files
type: spike
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - format
  - question
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-02T16:11:32Z
updated_at: 2026-09-03T04:03:47Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

A deferred question in plan section 15. AGENTS.md records that two agents each adding a file under .tickets/tickets/ merge cleanly and two editing the same ticket do not. Plan 7.1 hands content resolution to Git and stops there, and the format's only answer today is the merge_conflict check finding, which reports a conflicted file rather than resolving one. A merge driver stays inside the 7.4 promise, because Git invokes the driver and the driver runs no Git command itself. What it has to decide is which fields merge by union, such as Notes and Comments, which are last-writer-wins, such as status and priority, and which must always conflict, such as a claim. It also has to decide how a user installs one, because a driver needs a .gitattributes entry and a git config line, and a repository cannot set the config line for itself.

## Summary

Settled. git-ticket ships a merge driver, and plan 7.5 is the design: the field-by-field rules, why a driver stays inside the 7.4 promise, and the install story. Building it is TKT-01M1JPW1QJR3QM4SE9T3GDTZBX, kept separate because the question was whether to ship one, not what its code looks like. No code shipped here and the spike stayed a spike.

### Running the question changed what the driver is for

The ticket framed the work as deciding which fields merge by union, which are last-writer-wins, and which must always conflict. That framing assumes the conflicts are about the data. They are mostly not.

Two branches were made to edit one ticket in ways that share nothing: one set `priority`, the other added a label. They conflicted on `updated_by` alone. Git merged the priority and the label without complaint and handed back a file whose only disputed lines were the two neither agent typed. Every mutation rewrites `updated_by` and `updated_at` beside it, so any two concurrent edits collide there regardless of what they were. `Notes` and `Comments` supply the rest by appending at the same offset.

So the first useful driver is a small one. Resolving two provenance stamps and two append-only sections clears the conflicts a real workflow hits and adjudicates nothing anybody wrote. 7.5 still carries the full table so the rest is decided rather than discovered later, and most of its rows say conflict. A claim always conflicts: resolving it silently hands one ticket to two agents, each holding a file that says it is theirs.

### One correction worth recording

An early reading of the probe said `updated_at` does not conflict. That was luck, not a property: branch A edited within the same second as the base commit, so it never changed the field and Git took B value uncontested. Two agents editing seconds apart collide there too. Both stamps are conflict sources.

### The install half is the harder half

A driver needs a `.gitattributes` entry and a `merge.*.driver` config line. The first is tracked, so `init` can write it and a clone gets it. The second is not, and Git refuses on purpose to take an executable name from a repository, because a clone would otherwise run a command the person cloning never chose. That boundary is correct. It makes the config line a persons decision every time, so the job is to make it one command they run rather than a paragraph they transcribe.

### What else this raised

TKT-01M1JPXG7J6K4MQNTWA6TVMHPB asks whether `updated_at` and `updated_by` earn their cost at all. 5.3 defends them, saying the diff should report who touched a ticket and when, and the conflict evidence is the bill for that. Nothing in this repository reads either field. A driver that resolves them quietly fixes the symptom and hides the question, which is why it is filed rather than folded in.
