---
schema: 1
id: TKT-01M1QP27SDCV8SYC045C0SQJ62
title: Build git ticket self-update against the GitHub releases
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - release
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-05T02:23:51Z
updated_at: 2026-09-05T02:27:25Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Releases exist on both forges since v0.5.1, and the only upgrade path
is knowing that and doing it by hand. Build `git ticket self-update`,
per plan 12.6: replace the running binary with the latest GitHub
release, with no-op modes that answer whether an update exists without
touching anything.

The design is settled in the plan, decided with the user on
2026-09-04:

`self-update`, not `update`, which has meant "change a ticket's
fields" since Phase 2. `--check` reports the version gap and exits by
the graded bucket of 10.2: 10 patch, 11 minor, 12 major, 0 up to date,
so `self-update --check || notify` works in cron and tooling that
auto-applies patches but waits on minors compares one number.
`--dry-run` additionally names the asset for this OS and architecture
and the path it would replace. Both write nothing. The bare command
downloads, verifies sha256 against checksums.txt before anything
moves, and replaces the executable atomically.

Refusals with exit 1: a devel binary (someone's development tree, the
repair is `just install`), a failed checksum, an unwritable target, a
release with no asset for this platform.

The JSON envelope is the self-update kind of 10.7. The network
exception is scoped to this one command; the library and every other
command stay offline. The command reads no store and answers outside a
repository, like schema.

Tests inject the release API base URL and the target executable path
through Env, serve a fake release from httptest, and never reach the
real network.

## Acceptance criteria

- [x] self-update --check reports the gap and exits 0, 10, 11, or 12 by the graded bucket of 10.2
- [x] self-update --dry-run also names the asset and target path, writing nothing
- [x] The bare command verifies sha256 before replacing the executable atomically, and a bad checksum refuses
- [x] A devel binary refuses with the just install repair named
- [x] The self-update JSON kind of 10.7 has a test, and no test reaches the real network

## Notes

**agent:terva/mieli** at 2026-09-05T02:27:25Z

Built and all five criteria are ticked.

The plan moved first, in the same branch: 10.2 gains the graded exit
bucket, 10.7 specifies the self-update envelope, 12.6 is the design,
and the 12.1 command list gains the entry, taking the binary to 32
commands.

The exit-code question the user raised is settled in 10.2 with the
precedent named: the reserved graded bucket of zypper and terraform's
-detailed-exitcode, not fsck's bitmask, because a version gap is one
category rather than independent conditions. 10 patch, 11 minor, 12
major, 0 up to date. Under v0.x this composes with 12.4: a minor is
where breaks land, so tooling that waits on 11 gets the caution the
numbering promises.

Implementation notes for the next reader:

Run maps a typed exitStatusErr to the process exit code without an
error envelope, because the graded status is an answer, not a failure.
The devel refusal fires before any network call. The sha256 gate holds
before anything on disk moves, and the checksum-mismatch test proves
the target survives. The replace is write-beside-then-rename; Windows
renames the running binary aside first, and that path is written but
untested here, which the apply test says out loud with its skip.

runningVersion is a package variable over buildVersion because an
in-tree test binary reports devel and a suite that can only exercise
the refusal is no suite. Env gains ReleaseAPI and Executable, both
additive, both existing so the tests inject httptest and a file they
own. No test reaches the real network.

Ahead-of-latest exits 0 as up to date rather than offering a
downgrade. It is a real state: between a tag push and the mirror's
release job finishing, the tagging machine is ahead.

## Summary

Shipped. `git ticket self-update` replaces the running binary with the
latest GitHub release, per plan 12.6. `--check` reports the version gap
and exits by the graded bucket of 10.2 (10 patch, 11 minor, 12 major, 0
up to date), so `self-update --check || notify` works in cron and
tooling can auto-apply patches while waiting on minors. `--dry-run`
also names the asset and the target path. The bare command verifies
sha256 against checksums.txt before anything moves and replaces the
executable atomically. A devel binary refuses, naming `just install` as
the repair. The JSON envelope is the self-update kind of 10.7. The
network exception is scoped to this one command; every test runs
against httptest and none reaches the real network.
