---
schema: 1
id: TKT-01M1FCK1TW0QS7A8E0JGJWSXD9
title: Relax the go directive so it does not pin a patch release
type: chore
status: done
status_reason: null
priority: normal
labels:
  - integration
assignees: []
milestone: null
parent: TKT-01M1FCHH3QN04AYHZAP1M8DNQK
dependencies: []
references: []
claim: null
archive: null
created_at: 2026-09-01T21:04:20Z
updated_at: 2026-09-01T21:04:21Z
created_by:
  id: human:sothr
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

go.mod declared go 1.25.4. A go directive is the minimum toolchain a consumer must have, so a patch-level value in a library binds every consumer to that patch for no reason. terva declares go 1.25.0, so the pin would have forced a toolchain bump on the first repository that imports this one.

Found while reading both go.mod files during Phase 3 planning. Nothing in the code needs a 1.25 patch release: patch releases add no language features.

## Summary

go.mod now says go 1.25. gofmt, vet, go test -race, and check --strict all pass unchanged, and the build did not rewrite go.mod or go.sum. Consumers now need Go 1.25 or later rather than 1.25.4 or later.
