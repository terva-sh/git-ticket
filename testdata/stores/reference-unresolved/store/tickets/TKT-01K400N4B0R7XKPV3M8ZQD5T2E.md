---
schema: 1
id: TKT-01K400N4B0R7XKPV3M8ZQD5T2E
title: One reference resolves and one does not
type: task
status: ready
status_reason: null
priority: normal
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references:
  - ref: file:docs/present.txt
    path: docs/present.txt
  - ref: proposal:the-one-that-moved
    path: docs/missing.md
  - ref: url:https://example.invalid/spec
    path: null
claim: null
archive: null
created_at: 2026-08-31T12:00:00Z
updated_at: 2026-08-31T12:00:00Z
created_by:
  id: human:sothr
  name: Drew Short
updated_by:
  id: human:sothr
  name: Drew Short
extensions: {}
---

## Description

Three references, one finding. `docs/present.txt` resolves against the
repository root and says nothing. `docs/missing.md` does not resolve and is the
recorded warning. The `url:` reference has no path at all, so there is nothing
to resolve.

Per 5.5 a path resolves against the root of the Git repository holding the
store. This case directory stands in for that root, which is why the test
injects it rather than letting the tool run `git rev-parse`: the expectation
must not depend on where this repository is checked out.
