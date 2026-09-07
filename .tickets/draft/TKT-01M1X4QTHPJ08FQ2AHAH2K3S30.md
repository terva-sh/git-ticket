---
schema: 2
id: TKT-01M1X4QTHPJ08FQ2AHAH2K3S30
title: Widen the Windows CI lane to the cli and tui suites
type: chore
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - ci
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-07T05:16:30Z
updated_at: 2026-09-07T05:16:30Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

`.github/workflows/ci.yml` runs `go build ./...` and `go vet ./...` over the whole tree for Windows, but `go test` only over `./ticket/...`. TKT-01M1X3WD's fourth criterion asked for the suite and shipped unticked because of this.

The scope was narrow for a reason rather than by oversight. A lane that arrives red teaches everyone to ignore it, and two things were known to be broken on Windows before it was written:

- `cli/copy_test.go` execs `/bin/sh` in `TestRunClipboardToolOutlivesNoForkedChild` and `TestRunClipboardToolReportsTheToolsWords`. Neither has a Windows equivalent, since the point of the first is the fork-and-exit shape that leaves a child holding the inherited descriptors.
- The 18 test files under `tui/` have never run on Windows at all, so what breaks there is unmeasured.

### The work

Widen the lane one package at a time, and measure before promising. Add `./cli/...` first, since a Windows run of it is what tells us how much is actually broken beyond the two known tests. Then `./tui/...`.

Neither package is unknown territory now: the first Windows run of `./ticket/...` failed on CRLF rather than on anything platform-specific in the code, so the tests may be closer to Windows-clean than the caution suggests.

When the lane covers the suite, tick criterion 4 on TKT-01M1X3WD by PR rather than rewording it, the way criterion 5 was ticked once its evidence run existed.

### One caution

The lane triggers on a push of `main` plus `workflow_dispatch`, and the mirror only receives `main` at release time. Once the workflow exists on the mirror's default branch, `workflow_dispatch` can run it against any ref pushed there, which is the cheaper way to measure a widening than pushing `main` and hoping.

## Acceptance criteria

- [ ] The Windows lane runs go test ./... with no package excluded
- [ ] The two /bin/sh tests in cli/copy_test.go run or skip with a stated reason on Windows
- [ ] TKT-01M1X3WD criterion 4 is ticked by PR once the lane covers the suite
