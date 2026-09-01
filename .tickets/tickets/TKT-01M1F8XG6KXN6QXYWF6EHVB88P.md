---
schema: 1
id: TKT-01M1F8XG6KXN6QXYWF6EHVB88P
title: Decide whether the module path should follow where the code lives
type: spike
status: draft
status_reason: null
priority: normal
labels:
  - release
assignees: []
milestone: null
parent: null
dependencies:
  - TKT-01M1F7Z31HR5GV06D6Y7WZWJK4
references:
  - ref: proposal:git-ticket
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-01T20:00:08Z
updated_at: 2026-09-01T20:00:09Z
created_by:
  id: human:sothr
  name: ""
updated_by:
  id: human:sothr
  name: ""
extensions: {}
---

## Description

Deferred question 7 of plan section 15. go.mod declares github.com/terva-sh/git-ticket, and the repository is hosted at git.local.sothr.com/terva-sh/git-ticket with no public mirror, so the declared path is not fetchable. A module path is a promise about where a consumer can find the code, and nothing enforces it: a replace directive or a local path works either way, which is why this can sit wrong for a long time without failing. It is also hard to change once anything depends on it. Either publish a mirror at the declared path, or rename the module to the host that serves it. Phase 3 is the first consumer, so decide before terva depends on either answer.
