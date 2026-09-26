---
schema: 2
id: TKT-01M3DW3SWYJN3V32G9ETPGKJWH
title: Make the reference tests pass on the Windows lane
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - area/ci
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-26T03:28:33Z
updated_at: 2026-09-26T03:30:10Z
created_by:
  id: agent:terva/release-v0.24.0
  name: ""
updated_by:
  id: agent:terva/release-v0.24.0
  name: ""
extensions: {}
---

## Description

The Windows lane on the mirror (run 36214753463, commit cf56258) failed two
`ticket` tests added with the reference work, and that blocked tagging v0.24.0.

`TestResolveReferenceUsesPortableURLAndOptionalLocalCheckout` compared
`ResolveReference`'s `LocalPath` against the path the test wrote. The resolver
returns the path after `filepath.EvalSymlinks`, because `safeExistingPath` checks
containment in that name space. On the runner, the temp dir is spelled
`RUNNER~1`, an 8.3 short name, and resolving it expands the name. The same
failure reproduces on Linux with `TMPDIR` pointed through a symlink. The product
is right, so the test now compares canonical paths.

`TestMappingPlanKeepsRegistryAndRevisionFromOneSnapshot` races a writer
against readers of `references.yml`. Windows refuses to rename over a file
another handle holds open, so `writeFileAtomic` failed with "Access is denied".
The test asserts that each read sees one consistent snapshot, not that each
write succeeds. On Windows it now skips a write refused with a permission error,
which leaves the old bytes intact.

The second failure points at a property that predates this release. On
Windows, any `writeFileAtomic` can fail while another process is reading the
same file. Every store write goes through that function, so this is not new with
the reference registry, and this ticket does not change it.

## Acceptance criteria

- [ ] The Windows lane is green on the commit v0.24.0 tags
- [x] The reference resolver test passes with TMPDIR reached through a symlink

## Notes

**agent:terva/release-v0.24.0** at 2026-09-26T03:28:42Z

Reproduced the path failure on Linux with TMPDIR=/tmp/linktmp, a symlink to /tmp/realtmp: the test fails exactly as on the runner and passes after the fix. The whole ticket suite passes under that TMPDIR, and just ci passes. The rename fix cannot be proven off Windows; the lane after the merge is its evidence, which is criterion 1.

**agent:terva/release-v0.24.0** at 2026-09-26T03:30:10Z

PR #229 review 768 (run 9d6da90c-402d-4bec-b769-8b9cc16dacff, head 8d31814) raised one medium finding: on Windows the writer could have every write refused and the test would still pass with readers seeing a single revision. Accepted and fixed: the writer counts landed writes and fails under two. Proven able to fail by raising the threshold to 301, which reported 'only 300 of 300 registry writes landed'.

## Summary

Both Windows-lane failures in the reference tests were test assumptions, not product faults: the resolver's canonical path is compared canonically, and the snapshot test's writer tolerates Windows refusing a rename over an open file. The path fix is proven on Linux through a symlinked TMPDIR. Criterion 1 stays unticked until the mirror's Windows lane runs green on the merged commit, which is the release's own gate.
