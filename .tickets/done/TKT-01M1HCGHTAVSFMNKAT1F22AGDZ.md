---
schema: 1
id: TKT-01M1HCGHTAVSFMNKAT1F22AGDZ
title: Prove the worktree-shared store lock with a test
type: chore
status: done
status_reason: null
priority: high
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
references:
  - ref: plan:acceptance
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-02T15:41:27Z
updated_at: 2026-09-02T15:42:42Z
created_by:
  id: agent:terva/dev-loop
  name: ""
updated_by:
  id: agent:terva/dev-loop
  name: ""
extensions: {}
---

## Description

Plan 14 lists "worktrees sharing a Git directory share the lock" as an acceptance criterion, and 7.3 says separate worktrees serialize. lockPath() implements it by putting the lock under the Git common directory, with a fallback to a lock inside the store when there is no repository. Neither branch has a test: grep for lockPath across every _test.go finds nothing, and TestTwoProcessesOneWins runs both processes in one tree, so it would pass unchanged if lockPath started returning a per-worktree path. The failure it would miss is silent, two agents in two worktrees both writing with nothing to serialize them, which is the case terva sub-agents are in by construction.

## Acceptance criteria

- [x] A test creates a second worktree and asserts both stores resolve to one lock path
- [x] Holding the lock in one worktree makes a write from the other return lock_timeout
- [x] The outside-a-repository fallback path is covered too
- [x] The test fails when lockPath is changed to a per-worktree path

## Notes

**agent:terva/dev-loop** at 2026-09-02T15:42:42Z

Checked the test can fail by cutting the common-dir branch out of lockPath. The new test failed on both halves, the path comparison and the contention. The point worth keeping: TestTwoProcessesOneWins, TestLockTimeoutWhenHeld and the fallback test all still passed with the lock broken, because every one of them opens a single store directory. Only a second worktree can tell a shared lock from a per-store one.

## Summary

ticket/lock_test.go covers both branches of lockPath. TestWorktreesShareOneLock builds a repository, commits the store, adds a real second worktree, and asserts the two stores are different directories resolving to one lock path, that holding it in one blocks a write from the other with lock_timeout, and that releasing it lets the write through. TestLockPathFallsBackOutsideARepository pins the no-repository case to a lock inside the store.
