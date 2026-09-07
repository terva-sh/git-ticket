---
schema: 2
id: TKT-01M1X3WD866F5BVCBSHBP4P17G
title: Fix the store lock on Windows, which fails every mutation
type: bug
status: done
status_reason: null
priority: high
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
created_at: 2026-09-07T05:01:32Z
updated_at: 2026-09-07T05:06:50Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

`ticket/lock_other.go` carries `//go:build !unix` and fails every lock
acquisition on purpose, because a silent no-op would let two writers into the
store. Windows matches `!unix`, so on Windows every mutation fails with
`lock_timeout: the store lock needs flock, which this platform does not
provide`.

`s.lock()` has six callers and all of them are mutations: `apply.go` twice,
`fix.go`, `remove.go`, `migrate.go` and `series.go`. Read paths never lock. So
on Windows today reads work and every write fails.

terva found this. Its GitHub CI run 34081172303, job `test (windows-latest)` on
terva commit `60c379d6`, failed eight tests with that one message, and it blocks
terva v0.134.1 from going public. The handoff is
`docs/plans/handoff-git-ticket-windows-lock.md` in the terva tree.

The defect is old rather than new: v0.11.3 carries a byte-identical
`lock_other.go`. It surfaced now because nothing had ever run this code on
Windows. This repository's CI is Forgejo `runs-on: docker`, and its only GitHub
workflow is `release.yml` on ubuntu.

### The fix

Add `ticket/lock_windows.go` implementing `tryFlock` and `unflock` over
`LockFileEx` and `UnlockFileEx` from `golang.org/x/sys/windows`, which is
already a direct requirement. Narrow `lock_other.go` to `!unix && !windows` so
the two files do not both claim Windows.

`ERROR_LOCK_VIOLATION` maps to `(false, nil)`, not to an error. That is the
branch `EWOULDBLOCK` takes on unix, and it is what makes `lock()` poll to its
deadline instead of turning every contended write into an immediate
`lock_timeout`.

### The lane

Add `.github/workflows/ci.yml` with a `windows-latest` job, so the next platform
bug is caught here rather than at terva's release gate. It is guarded by
`github.server_url` for the reason `release.yml` is: Forgejo Actions executes
`.github/workflows` too and has no Windows runner.

## Acceptance criteria

- [x] ticket/lock_windows.go implements tryFlock and unflock over LockFileEx, and ERROR_LOCK_VIOLATION maps to (false, nil) so lock() polls to its deadline
- [x] go list reports exactly one lock implementation per target: lock_windows.go on windows, lock_unix.go on linux and darwin, lock_other.go on plan9
- [x] GOOS=windows go build ./... and GOOS=windows go vet ./... both exit 0
- [ ] .github/workflows/ci.yml runs the suite on windows-latest, guarded by github.server_url so Forgejo skips it
- [ ] A Windows runner executes TestWorktreesShareOneLock green, proving both the granted and the contended branch

## Notes

**agent:terva/mieli** at 2026-09-07T05:05:48Z

Criteria 4 and 5 ship unticked, for different reasons.

Criterion 4 asks that the lane run "the suite" on windows-latest. It runs `go build ./...` and `go vet ./...` over the whole tree and `go test ./ticket/...`, which is less. The narrowing is deliberate rather than an omission: `cli/copy_test.go` execs `/bin/sh` in two tests, and the 18 tui test files have never run on Windows at all, so a lane scoped to the whole suite would arrive red. A red lane nobody can fix in the same change teaches everyone to ignore it. The criterion stays as written and unticked, and a follow-up ticket covers widening it once cli and tui are Windows-clean.

Criterion 5 cannot be proven from this branch. The lane triggers on a push of main, and the mirror only receives main at release time, so no Windows runner sees this code until then. The evidence run is assigned to the v0.14.2 release verification: push main to the mirror, read the Windows lane, then tag. The tick lands by a later PR, per the TKT-01M1S02QA precedent for evidence that cannot exist until a later event.

**agent:terva/mieli** at 2026-09-07T05:05:48Z

One deliberate deviation from the handoff, on the ERROR_IO_PENDING branch.

The handoff wrote: "I pass a zeroed `Overlapped` on a synchronous handle, so `ERROR_IO_PENDING` should not arise. If it can, it belongs in the contended branch and not in the fatal one."

The shipped code puts it in the fatal branch, which is the default case, and the comment says why. `os.OpenFile` never passes FILE_FLAG_OVERLAPPED, so the handle is synchronous and LockFileEx cannot queue asynchronously on it. Reaching ERROR_IO_PENDING therefore means an assumption in this file is wrong. Treating it as contention would spin the poll loop to its deadline and then report `another process holds the store lock`, which blames a holder that does not exist and hides the real condition. The fatal branch reports `lock_timeout: Overlapped I/O operation is in progress`, which names the condition and can be searched for.

Both branches produce a failed write, so this is not a correctness difference. It is a difference in what the failure tells the reader.

## Summary

`ticket/lock_windows.go` takes the store lock through LockFileEx and releases it through UnlockFileEx, one byte at offset zero, and `lock_other.go` narrows to `!unix && !windows` so the two files no longer both claim Windows. ERROR_LOCK_VIOLATION maps to `(false, nil)`, the branch EWOULDBLOCK takes on unix, so `lock()` polls to its deadline rather than failing at once. ERROR_IO_PENDING is fatal on purpose, and a note says why.

`TestOneLockImplementationPerPlatform` asks the toolchain which lock file each target compiles, over windows, linux, darwin and plan9. It fails on the old tag, reporting that windows compiled both `lock_other.go` and `lock_windows.go`, and passes on the new one. That is the test the project never had: compiling for one platform proves nothing about the tags of another, which is how this defect survived from v0.11.3 to v0.14.1 with a green suite.

`.github/workflows/ci.yml` adds the Windows lane, guarded by `github.server_url` so Forgejo skips a job it has no runner for. It builds and vets the whole tree and runs `go test ./ticket/...`, which is narrower than criterion 4 asked for, because cli and tui are not Windows-clean yet.

Verified here: `just ci` green, `GOOS=windows go build ./...` and `go vet ./...` exit 0 on amd64 and arm64, and the go list partition is exclusive and total. Not verified here: any of it running on a real Windows machine. The lane produces that evidence at the v0.14.2 release, when main reaches the mirror, and criterion 5 is unticked until it does.
