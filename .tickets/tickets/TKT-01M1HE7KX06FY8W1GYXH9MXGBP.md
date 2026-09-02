---
schema: 1
id: TKT-01M1HE7KX06FY8W1GYXH9MXGBP
title: Decide whether git-ticket ships a merge driver for ticket files
type: spike
status: draft
status_reason: null
priority: normal
labels:
  - format
assignees: []
milestone: null
parent: null
dependencies: []
references: []
claim: null
archive: null
created_at: 2026-09-02T16:11:32Z
updated_at: 2026-09-02T16:11:32Z
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
