---
schema: 1
id: TKT-01M1H9QGDCV9KNVWAVD2MB655K
title: instructions tells an agent to claim a ticket that create leaves in draft
type: bug
status: ready
status_reason: null
priority: normal
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
references:
  - ref: workflow:instructions
    path: cli/instructions.md
claim: null
archive: null
created_at: 2026-09-02T14:52:49Z
updated_at: 2026-09-02T14:52:52Z
created_by:
  id: agent:terva/dev-loop
  name: ""
updated_by:
  id: agent:terva/dev-loop
  name: ""
extensions: {}
---

## Description

The workflow block from `git ticket instructions` says: "Claim it with git ticket claim ID, then git ticket status ID in-progress." A ticket that create just wrote is in draft, and claim on a draft fails with validation_failed: a ticket in draft cannot be claimed. So the documented first move fails on exactly the ticket an agent most often files and picks up itself, and the agent has to work out that status ID ready comes first.

Found by following the block on TKT-01M1H9KT. Either the block names the ready step, or claim promotes a draft the way the lifecycle would. TestInstructionsNameRealCommands holds the block to commands and flags that exist, which is why it did not catch this: every command in the sentence is real, and only the order is wrong.
