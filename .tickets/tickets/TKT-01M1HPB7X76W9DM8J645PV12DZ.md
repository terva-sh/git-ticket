---
schema: 1
id: TKT-01M1HPB7X76W9DM8J645PV12DZ
title: Give ac and dod a remove, and let their operations repeat in one call
type: task
status: ready
status_reason: null
priority: normal
labels: []
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

runChecklist registers --add, --check, and --uncheck, and exactlyOne refuses more than one of them per invocation. So there is no way to delete a criterion that turned out to be wrong, and checking three boxes takes three writes with three revisions.

Backlog.md takes --check-ac 1 --check-ac 3 --uncheck-ac 2 --remove-ac 4 in one edit, and documents that removals process high to low so the indexes stay stable underneath them. That last detail is the part worth copying exactly: plan 10.1 numbers checkbox lines from one over the section, so removing item 2 renumbers everything after it, and a caller who asked to remove 2 and 4 means the 4 they read.

See section B3 of docs/review-backlog-md.md.

Scope: --remove N on ac and dod, and make all four operations repeatable and combinable in one invocation. Everything the caller asked for lands as one write or none of it does, which is the rule runUpdate already follows. Removals apply high to low. An index outside the section is invalid_field naming the count.
