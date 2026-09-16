---
schema: 2
id: TKT-01M1W3TM8BBJRSRC17WPFMDXXS
title: Decide whether the block splits further than its two forms
type: spike
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - question
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-06T19:41:19Z
updated_at: 2026-09-16T18:57:02Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:claude/t3code
  name: ""
extensions: {}
---

## Description

Decide whether the block is worth splitting further than the two forms it now
has, from section D of `docs/review-backlog-md.md`.

This was filed as a task to split the block into named guides, on the Backlog.md
model of `overview`, `task-creation`, `task-execution` and `task-finalization`,
where `AGENTS.md` carries only a pointer at `backlog instructions overview` and
each phase is fetched when the agent is about to do that phase.

### What TKT-01M2HMZ308RYY9ZXXMJS738S48 already took

The win this ticket named was the always-loaded part getting smaller, and that
has happened. `instructions --write` now installs a 951-word short form instead
of the 2,473-word long one, and the long one stays behind a bare
`git ticket instructions`, which the short form names in its second paragraph.

Four of the five acceptance criteria this ticket carried are satisfied by that
work: the written block is shorter and points at the command that prints the
rest, `TestInstructionsNameRealCommands` and `TestInstructionsWorkflowRuns` are
table-driven over both forms, and the vocabulary question is moot because
neither form is named after a phase.

The cost this ticket worried about did not arrive. Splitting one document into
several was going to make the ordering test either run each guide alone, losing
the cross-guide sequence, or stitch them back together. Two forms of the whole
sequence sidestep that: each runs end to end on its own.

### What is actually still open

Only whether to subdivide further, into per-phase guides, now that the cheap
half of the win is taken. The remaining case for it is narrow. An agent filing a
ticket loads the sections on doing and finishing work it will not use in that
turn, and per-phase guides would cut that too.

The case against is that the short form is already 951 words, so the saving is a
few hundred at best, and it is bought with a real cost: the reader has to know
which phase they are in before they can fetch the right guide, and an agent that
guesses wrong reads the wrong rules. The two-form split has no such failure,
because both forms carry the whole sequence.

Settle it with a measurement rather than a preference. If the per-phase saving
is under roughly 300 words against the current short form, the lookup is not
worth it and this closes as declined.

## Acceptance criteria

- [ ] The per-phase saving against the current 951-word short form is measured, not estimated
- [ ] The decision is recorded in plan section 15 with a reopen trigger, whichever way it goes
- [ ] If declined, plan 12.1 says why, so the next reader of the Backlog.md review does not refile it

## Definition of done

- [ ] NOTICE credits Backlog.md for the idea, with no file header because no code was adapted

## Notes

**agent:terva/import-preview-number** at 2026-09-15T05:00:22Z

Rewritten from a task into a spike because TKT-01M2HMZ308RYY9ZXXMJS738S48 took the win it was filed for. Its original acceptance criteria are left in place and unticked rather than removed: four of the five are satisfied, but by different work than this ticket proposed, and ticking them here would claim this ticket did it.

**agent:terva/import-preview-number** at 2026-09-15T05:00:37Z

Supersedes the previous note on this ticket, which said the original acceptance criteria were left in place. They are not; they described building named guides, which is the thing this ticket no longer proposes doing. A spike whose criteria describe an implementation reads as work somebody abandoned. Replaced with the three that describe settling the question.
