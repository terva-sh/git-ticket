---
schema: 2
id: TKT-01M1W3TM8BBJRSRC17WPFMDXXS
title: Split the agent instruction block into named guides
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
created_at: 2026-09-06T19:41:19Z
updated_at: 2026-09-06T19:41:19Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Split the agent instruction block into named guides, from section D of
`docs/review-backlog-md.md`.

Backlog.md splits its instructions into `overview`, `task-creation`,
`task-execution`, and `task-finalization`, and the block it writes into
`AGENTS.md` says only to run `backlog instructions overview`. So the part
that sits in every agent's context is short, and the detail is fetched when
the agent is about to do that particular thing.

Ours is one piece. `cli/instructions.md` is embedded with `go:embed` and
printed whole by `git ticket instructions`. It is already long enough that
`TestInstructionsWorkflowRuns` has to execute the sequence it prints, in
order, against a real store, to catch an ordering that reads fine and cannot
run. That test exists because the block once said to claim before ready, and
a draft cannot be claimed.

The win is the always-loaded part getting smaller. The cost is that the two
tests holding the block honest were written for one document.
`TestInstructionsNameRealCommands` checks every command and flag the prose
names against what the binary has, and `TestInstructionsWorkflowRuns` runs
the sequence. Splitting means deciding whether each guide runs on its own,
which loses the cross-guide ordering that test was written to catch, or
whether the suite stitches them back into one sequence and runs that.

This is an idea lift and not a code lift, so under the ruling on this work it
takes a NOTICE entry and no file header.

Worth settling before building: whether the guide names are theirs or ours.
Taking `task-creation` verbatim imports a vocabulary where the noun is task
and ours is ticket, and this project's convention is that names are singular
and the codebase is the word list.

## Acceptance criteria

- [ ] Named guides can each be printed on their own
- [ ] The block written into AGENTS.md is shorter than today's and points at the guides
- [ ] TestInstructionsNameRealCommands still holds across every guide
- [ ] The cross-guide ordering TestInstructionsWorkflowRuns was written to catch is still covered
- [ ] Guide names use this project's vocabulary, ticket rather than task

## Definition of done

- [ ] NOTICE credits Backlog.md for the idea, with no file header because no code was adapted
