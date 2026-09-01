---
schema: 1
id: TKT-01M1F7Z2V612JYMNWR0GHHMMNK
title: Run the suite and git ticket check in CI
type: chore
status: ready
status_reason: null
priority: high
labels:
  - ci
assignees: []
milestone: null
parent: null
dependencies: []
references:
  - ref: proposal:git-ticket
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-01T19:43:32Z
updated_at: 2026-09-01T19:43:32Z
created_by:
  id: human:sothr
  name: ""
updated_by:
  id: human:sothr
  name: ""
extensions: {}
---

## Description

Phase 2 closes on two exit criteria. The scripted end-to-end run is met by TestLifecycle. This is the other: git ticket check --strict green over this repository's own store, alongside go test ./... and gofmt. The build has to stay offline, so GOFLAGS=-mod=mod and GOPROXY=off.
