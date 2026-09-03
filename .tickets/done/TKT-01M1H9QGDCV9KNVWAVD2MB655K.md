---
schema: 1
id: TKT-01M1H9QGDCV9KNVWAVD2MB655K
title: instructions tells an agent to claim a ticket that create leaves in draft
type: bug
status: done
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
updated_at: 2026-09-02T14:57:13Z
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

## Notes

**agent:terva/dev-loop** at 2026-09-02T14:57:13Z

The fix is prose plus a test that runs it. TestInstructionsNameRealCommands could not have caught this: every command in the sentence existed and only the order was wrong. TestInstructionsWorkflowRuns pairs each step with the span it comes from, so it fails both when a step stops working and when the block reorders them.

**agent:terva/dev-loop** at 2026-09-02T14:57:13Z

Checked the new test against the old block by reverting instructions.md alone: it fails with "the block no longer names `git ticket status ID ready`". A test that has never failed proves nothing.

## Summary

Doing the work now opens with the draft-to-ready move before claim, and Filing new work says create lands in draft and why. TestInstructionsWorkflowRuns drives the eight documented steps against a real store in the block's own order and ends on check --strict.
