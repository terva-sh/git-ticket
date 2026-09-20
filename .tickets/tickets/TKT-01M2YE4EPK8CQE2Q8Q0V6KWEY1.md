---
schema: 2
id: TKT-01M2YE4EPK8CQE2Q8Q0V6KWEY1
title: Say that the canvas places cards by rule
type: task
status: in-progress
status_reason: null
priority: normal
due_on: null
labels:
  - area/cli
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
claim:
  actor: agent:claude/t3code-a6d0ff31
  branch: t3code/canvas-applied
  worktree: /home/sothr/workspace/git.local.sothr.com/terva-sh/git-ticket
  commit: 0f46dec0c114324e20e28695d868ca11cdce118f
  claimed_at: 2026-09-20T03:35:38Z
  expires_at: null
archive: null
created_at: 2026-09-20T03:35:37Z
updated_at: 2026-09-20T03:44:26Z
created_by:
  id: agent:claude/t3code-a6d0ff31
  name: ""
updated_by:
  id: agent:claude/t3code-a6d0ff31
  name: ""
extensions: {}
---

## Description

git-ticket-canvas v0.5.0 places every unpinned card on a board with pens by the first pen in ruleOrder whose labels it carries, through one function of its own that reads the rule the way layout.Route does. The canvas-board and canvas-explain envelopes answered applied:false since v0.21.0 and canvas show and explain said routing was not applied yet, both waiting for exactly this. Flip the flag, drop the not-applied wording, and let placement say rules on a board with pens; plan 10.10 and 12.10 record it. Canvas side: TKT-01M2ND1RKK6S4GXZQKKPP6H87P in the canvas store.

## Acceptance criteria

- [x] canvas-board and canvas-explain answer applied:true
- [x] explain's placement is pinned, rules, or status-lanes, and says which on the page
- [x] plan 10.10 and 12.10 say the canvas places by rule since git-ticket-canvas v0.5.0
