---
schema: 1
id: TKT-01M1HPB7ZT0FREYR04ZSHTMW3F
title: Add a milestones allowlist to config.yml
type: task
status: ready
status_reason: null
priority: normal
labels:
  - format
assignees: []
milestone: null
parent: null
dependencies: []
references:
  - ref: review:backlog-md
    path: docs/review-backlog-md.md
claim: null
archive: null
created_at: 2026-09-02T18:33:19Z
updated_at: 2026-09-02T18:34:29Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

milestone is a bare scalar with no registry. Nothing validates it, so v1.2 and v1.2.0 are two milestones and check cannot tell a typo from a new milestone. A store accumulates near-duplicates and the list filter quietly answers about the wrong one.

config.yml already establishes the pattern. Plan 4.1 makes labels an advisory allowlist that check warns about and never errors on, which is exactly the strength this wants: a typo is worth reporting and a new milestone should not need a config edit before the ticket can be filed.

Backlog.md goes much further, with a file per milestone carrying a description and a due date, and add, rename, remove, archive, and list commands. rename cascades into every task and remove makes the caller choose clear, keep, or reassign. That cascade is real and a UI cannot supply it, but it should wait for a store that wants dates on a milestone. See section B4 of docs/review-backlog-md.md.

Scope: a milestones key in config.yml, and a milestone_unknown warning in plan section 11 beside label_unknown. Adding a warning code is additive under 12.4. The corpus needs a fixture and a sidecar, because TestCorpusCoversEveryPlanCode holds section 11 and testdata to each other.
