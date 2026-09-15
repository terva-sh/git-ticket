---
schema: 2
id: TKT-01M2H6TM1T2W0WF99HMDZCM520
title: Carry parent and criteria state through git ticket import --adopt
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
references:
  - ref: origin-ticket:TKT-01M2DQPQS36PP4TV2M4SHX4T8W
    path: null
  - ref: origin-store:ledger
    path: null
claim: null
archive: null
created_at: 2026-09-15T00:17:48Z
updated_at: 2026-09-15T00:17:48Z
created_by:
  id: agent:claude/skill-bundle-2
  name: ""
updated_by:
  id: agent:claude/skill-bundle-2
  name: ""
extensions: {}
---

## Description

Filed from a reflect backlog item, 2026-09-13. Adopting a ticket into a project store with git ticket import --adopt drops the parent link and every acceptance-criteria tick, so the adopted copy lands in draft with every box unchecked while the origin copy is ticked and done. Observed moving TKT-01M2CM7X1KAEXZY6ERVFNTXQR3 into warricksothr/agent-session. This belongs to terva-sh/git-ticket; export it there when picked up. Suggested mechanism: a move operation, or an --adopt option, that carries parent, criteria state, and provenance, and closes the origin with a summary naming the adopted id. Also add a one-paragraph procedure to docs/workspace-layout.md.
