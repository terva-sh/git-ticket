---
schema: 1
id: TKT-01M1SPR216X4HQBDD8KC3MD5N6
title: Make just build detect a worktree and set -buildvcs=false
type: chore
status: draft
status_reason: null
priority: normal
due_on: null
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-05T21:14:14Z
updated_at: 2026-09-05T21:14:21Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

`just build` fails inside a linked git worktree, and so does everything
that depends on it: `check`, `fix`, `ready`, and `ci`. Make `build`
detect the worktree and pass `-buildvcs=false` itself.

This is not a corner case here. Sub-agents work in worktrees, so a
sub-agent's first `just ci` fails at the build step for a reason that
has nothing to do with its change. AGENTS.md records the workaround as
of d376381, but nothing makes a sub-agent read it before the failure.

### What actually happens

`go build` of the main package stops before compiling anything:

    error obtaining VCS status: exit status 128
    Use -buildvcs=false to disable VCS stamping.

Go finds the VCS root by looking for a `.git` directory, and a linked
worktree has a `.git` file pointing into the main repository instead.

The shape of the failure is what makes it expensive. `go vet` and `go
test` both pass in a worktree, because only linking a binary stamps a
version, so the suite is green while the build is red. Verified in a
worktree at de6db3f: `just ci` exits 1, and
`GOFLAGS=-buildvcs=false just ci` exits 0 with the full gate green.

### The detection

One command tells a linked worktree from the main tree, verified on
2026-09-05:

    git rev-parse --git-dir          # .git                  in the main tree
    git rev-parse --git-common-dir   # .git                  in the main tree
    git rev-parse --git-dir          # <repo>/.git/worktrees/NAME  in a worktree
    git rev-parse --git-common-dir   # <repo>/.git                 in a worktree

They are equal in the main tree and differ in a linked worktree. A
submodule does not trip this, because its git-dir and common-dir are
the same, though this repository has no submodules to care about.

### Only `build` needs it

`build` is the only recipe that runs `go build` on the main package.
`check`, `fix`, `ready` and `ci` reach it through their dependency on
`build`, so one change covers all five.

`install-release` needs nothing. It builds in a throwaway clone with a
real `.git` directory, which is exactly why it clones rather than
adding a worktree.

`.forgejo/workflows/ci.yml` needs nothing either. The runner does an
ordinary checkout, not a worktree. This is the one case where the
justfile and the workflow are allowed to differ, and the reason should
be stated wherever that mirror rule is repeated.

### The constraint worth holding

A binary built this way reports `devel (unknown)` rather than a
version, because disabling the stamping is the whole point. That is
fine for `check`, which only reads the store, and it is never fine for
a release or for answering what version something is.

So the recipe must say what it did. `just build` already prints the
short SHA it built from, which reads like reassurance; printing that
line while silently producing a binary that cannot report its own
version is the same class of trap as the stale `./git-ticket` this
file already warns about. A person who sees `devel` later should be
able to remember why.

### The alternative, named so it is a decision and not an oversight

`build` could refuse in a worktree with a message naming the flag,
rather than setting it. That is more honest and more annoying, and it
leaves every sub-agent to hit the failure once. Automatic was asked
for. If the implementer changes their mind while building it, that is
a decision to record, not to take quietly.

## Acceptance criteria

- [ ] just build succeeds inside a linked git worktree, and so do check, fix, ready, and ci, with no environment variable set by the caller.
- [ ] The detection compares git rev-parse --git-dir against --git-common-dir, so the flag is passed in a linked worktree and not in the main tree.
- [ ] A build in the main tree still stamps a real version: git ticket --version reports the tag rather than devel, proving the flag was not passed there.
- [ ] When the flag is passed, just build says so, because the binary it produced reports devel and the existing short-SHA line reads like reassurance.
- [ ] install-release and .forgejo/workflows/ci.yml are unchanged, and the justfile records why the workflow does not need the same treatment.
