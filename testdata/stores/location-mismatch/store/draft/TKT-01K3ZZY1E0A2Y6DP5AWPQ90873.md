---
schema: 1
id: TKT-01K3ZZY1E0A2Y6DP5AWPQ90873
title: Done ticket left behind in draft
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-08-31T12:00:00Z
updated_at: 2026-09-12T09:00:00Z
created_by:
  id: human:sothr
  name: Drew Short
updated_by:
  id: human:sothr
  name: Drew Short
extensions: {}
---

## Description

The status is `done` and the file is in `draft/`, which is the far end of the
pipeline sitting at the near end of it.

This is the case that catches a rule written as a pair of comparisons rather
than as a mapping. Neither directory here is `tickets/` and neither is
`archive/`, so an implementation that reasoned about the working set and the
archive alone would find nothing wrong with it.

The repair is the same as for the other two: `check --fix` moves the file to
`done/`, because 6.3 makes the status authoritative and the status says where it
goes.
