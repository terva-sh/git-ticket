---
schema: 1
id: TKT-01M1HPCJJ1FFHG7HXC8QG1JRAG
title: Decide whether a query reads tickets from other branches
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
dependencies: []
blocks_on: none
references:
  - ref: review:backlog-md
    path: docs/review-backlog-md.md
claim: null
archive: null
created_at: 2026-09-02T18:34:03Z
updated_at: 2026-09-03T06:01:30Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

A deferred question raised by reading Backlog.md, and the one place their design is ahead of ours on a problem we actually have. See section D of docs/review-backlog-md.md.

Every query reads the working tree and nothing else, so a ticket claimed and moved to in-progress in another worktree is invisible here until the branch merges. AGENTS.md describes exactly that situation: several agents work this repository at once, in separate worktrees, each holding a main it believes is current.

Backlog.md reads the ticket directory out of every branch touched in the last 30 days, using git ls-tree per branch and comparing last-modified times, and shows the newest state it finds. src/core/cross-branch-tasks.ts is the whole mechanism and it is small.

Everything that read needs is read-only, so plan 7.4 permits the shape, but git ls-tree and the git log invocation behind the modification times are not in the 7.4 table and would have to be added there with their reasons before any code ran them.

Do not copy their conflict rule. Two branches that both touched a ticket are resolved by file mtime, and taskResolutionStrategy is a configuration knob choosing which guess to make, most_recent or most_progressed. A claim block has an actor and a timestamp and answers the question properly, which is an advantage we have and they do not.

The trigger: two agents in this repository claim the same ticket because neither could see the other claim. That has not happened, and the PR rule in AGENTS.md is what has been absorbing the pressure so far.

Open beyond the mechanism: whether cross-branch reads are a flag on list and ready, or a store setting, and whether the result marks which branch a ticket came from so a caller can tell a local ticket from a remote one. Answering with a merged view and no provenance would let an agent claim a ticket it cannot see the file for.

## Notes

**agent:terva/mieli** at 2026-09-03T06:01:30Z

Groomed. The trigger has not fired: no two agents in this repository have claimed the same ticket blind. The PR rule in AGENTS.md is still absorbing the pressure.

Worth saying plainly, because the timing invites the wrong conclusion. The merge driver shipped today, TKT-01M1JPW1, and it does not touch this. It resolves two edits to one ticket file once Git is already merging them. This ticket is about a claim being invisible until that merge happens, which is the window before the driver has anything to do. Closing this as covered by 7.5 would be a mistake.

One thing did move in this ticket favour. Plan 7.4 grew its first row today, for `config`, so the table is no longer a closed set and there is a worked example of what adding a row costs: the command, its reason, and the narrowest true statement of what it touches. `ls-tree` and `log` would each need that, and both are honest reads.
