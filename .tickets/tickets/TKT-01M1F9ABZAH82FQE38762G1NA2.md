---
schema: 1
id: TKT-01M1F9ABZAH82FQE38762G1NA2
title: "Fix CI: the runner cannot resolve actions/checkout"
type: bug
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
created_at: 2026-09-01T20:07:10Z
updated_at: 2026-09-01T20:07:10Z
created_by:
  id: human:sothr
  name: ""
updated_by:
  id: human:sothr
  name: ""
extensions: {}
---

## Description

The first push turned CI red. It fails before running a single step of the workflow. The Forgejo runner resolves actions/checkout@v4 against https://git.local.sothr.com/actions/checkout and gets a 404, because that instance points DEFAULT_ACTIONS_URL at itself rather than at GitHub and does not mirror the actions repositories. actions/setup-go@v5 would fail the same way for the same reason. The runner does pull container images from container.local.sothr.com/library/, so a local registry mirror exists even though the actions are not mirrored. Three ways out, and which one is right depends on the instance: fully qualify the uses lines as https://github.com/actions/checkout@v4, which needs egress to github.com from the runner; mirror the two action repositories into the local Forgejo; or drop the actions entirely, run the job in a golang image from the local registry, and clone the repository with git in a step. Everything the workflow actually runs passes locally: gofmt, vet, go test ./..., and check --strict. This blocks the second Phase 2 exit criterion in plan section 13.
