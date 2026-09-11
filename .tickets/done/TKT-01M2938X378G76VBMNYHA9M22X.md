---
schema: 2
id: TKT-01M2938X378G76VBMNYHA9M22X
title: cross-branch reads nothing when the store path holds a symlink
type: bug
status: done
status_reason: null
priority: high
due_on: null
labels: []
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references:
  - ref: plan:7.4
    path: docs/plan.md
  - ref: handoff:flywheel-export-import
    path: null
claim: null
archive: null
created_at: 2026-09-11T20:41:46Z
updated_at: 2026-09-11T20:46:05Z
created_by:
  id: agent:terva/mieli
  name: Mieli
updated_by:
  id: agent:terva/mieli
  name: Mieli
extensions: {}
---

## Description

A cross-branch query against a store reached through a symlinked path silently
reports only the tickets on the current branch. Exit 0, no warning, no finding.
A wrong answer that looks like a right one.

### Repro, on Linux

```sh
mkdir -p /tmp/gt-real/repo && ln -s /tmp/gt-real /tmp/gt-link
cd /tmp/gt-link/repo && git init -q && git ticket init
git ticket create --title "on main"; git add -A && git commit -qm tickets
git switch -qc other
git ticket create --title "only on the other branch"
git add -A && git commit -qm "one more"
git switch -q main
git ticket list --all --cross-branch     # reports 1 ticket
cd /tmp/gt-real/repo
git ticket list --all --cross-branch     # reports 2
```

Measured on this machine at v0.14.3+4: one ticket through the symlinked
spelling, two through the resolved one. The shell resolves a symlink out of the
working directory on its own, so the trigger is the store path arriving as a
string that still contains the link. `PWD` carrying the symlinked spelling is
one way to produce that, and `t.TempDir()` on macOS is another.

### Cause

`storePathspec` in `ticket/crossbranch.go` builds the `ls-tree` pathspec with
`filepath.Rel(s.Root(), s.path)`. `Root()` comes from `git rev-parse
--show-toplevel`, which resolves symlinks, and `s.path` does not. The two are
then in different name spaces, `Rel` returns a path full of `../..`, `ls-tree`
matches nothing, `treeAt` returns no entries, and the scan reports the working
tree alone.

This is the same split `displayPath` already handles, and it handles it by
retrying through `EvalSymlinks`. The fix is to give `storePathspec` the same
treatment rather than to invent a second one.

### Where it came from

Reported by an agent working git-ticket in another organization, as a macOS test
failure: four tests in `cli/crossbranch_test.go` fail on an unmodified checkout
because `/var/folders` is a symlink to `/private/var/folders`. Their workaround
was `TMPDIR=/private/tmp`. The report undersold it. It is not a test problem and
not macOS-only, and the Linux repro above is the reason to fix it rather than to
pin `TMPDIR`.

### Why it is worth the priority

Every other failure in this tool is loud. `check` names a finding, a mutation
refuses, a stale revision refuses by name. This one answers confidently with
half the store, and the half it drops is the half a cross-branch query exists to
find.

## Acceptance criteria

- [x] storePathspec resolves the store path and the repository root into one name space before taking their relative path
- [x] A regression test fails without the fix and passes with it, and runs on every platform rather than needing a macOS runner
- [x] cli/crossbranch_test.go passes with TMPDIR pointing at a symlinked directory
- [ ] The four macOS test failures the original report described are gone without setting TMPDIR

## Implementation plan

1. Reproduce in a test first. Build the symlinked-path case the way
   `TestAStoreSurvivesACRLFClone` builds the CRLF case: with the real mechanism,
   not a stand-in, so it runs everywhere. A symlinked temporary directory is
   reachable from any platform, so no runner is needed.

2. Give `storePathspec` the `EvalSymlinks` retry that `displayPath` already has.
   Look at `evalExisting` before writing a second one: a path that does not
   exist must not break the call, which is the trap that helper already covers.

3. Check whether `Root()` and `s.path` disagree anywhere else. `storePathspec`
   is the call this ticket measured, and a grep for `filepath.Rel` will say
   whether it is the only one. If there are others, name them here rather than
   fixing them silently.

4. Confirm the original report: run `cli/crossbranch_test.go` with `TMPDIR`
   pointed at a symlinked directory and see the four failures, then see them go.

## Notes

**agent:terva/mieli** at 2026-09-11T20:41:51Z

draft to ready: Promoted on the user's instruction to fix this ahead of the export/import handoff work.

**agent:terva/mieli** at 2026-09-11T20:45:53Z

Fixed in `storePathspec` by resolving both operands through `EvalSymlinks`
before `filepath.Rel`, with a small `resolveSymlinks` helper beside it that
answers with what it was given when resolution fails.

Not `evalExisting`, and the comment says why rather than leaving the next reader
to wonder about two helpers doing one job. That one walks up to the first
existing parent because a mutation reports a path it has just deleted. Neither
path here can be missing: `OpenWith` stats the store before returning one, and
the root came from git a moment earlier. Copying the walk would have bought
nothing and left two versions of one rule to drift, which is the lesson PR #135
already paid for.

Plan step 3, the grep for other places `Root()` and `s.path` could disagree:
`ticket/store.go:400`, `s.rel`, is the only other `filepath.Rel` in the package
and it is not affected. Both of its operands descend from `s.path` itself, so
they are always spelled the same way and no name-space split can reach it. Left
alone deliberately.

The measurement, on Linux with `/tmp/gt-linktmp` symlinked to `/tmp/gt-realtmp`:

    TMPDIR=/tmp/gt-linktmp go test ./cli/ -run TestCrossBranch
      without the fix   4 failures
      with the fix      ok, 0.474s

The four are `TestCrossBranchListFindsATicketOnAnotherRef`,
`TestCrossBranchReadyHonoursAClaimOnAnotherRef`,
`TestCrossBranchReadsRemoteTrackingRefs` and
`TestCrossBranchRowsAbbreviateUnambiguously`, by name the same four the original
report named.

`TestCrossBranchReadsAStoreReachedThroughASymlink` in `ticket/crossbranch_test.go`
is the regression, and it was written before the fix and watched to fail. Its
control subtest reads the same repository through the resolved spelling and
passed both before and after, which is what says the repository was built
correctly and the failure was the symlink rather than the fixture. It needs no
macOS runner, because a symlinked temporary directory is reachable anywhere. On
a platform that refuses to make a symlink it skips and says so, rather than
passing quietly, so it cannot become a test that no longer tests anything.

Criterion 4 stays unticked on purpose. It asks for the four macOS failures to be
gone on macOS, and there is no Mac here. What was proven is the identical
mechanism: `/var` to `/private/var` is a symlink in front of `TMPDIR`, which is
exactly what the run above builds, and it produced the same four failures by
name and then cleared them. That is strong evidence and it is still not the run
the box asks for. Whoever next has this branch on macOS can tick it with one
`go test ./cli/ -run TestCrossBranch` and no `TMPDIR` set.
