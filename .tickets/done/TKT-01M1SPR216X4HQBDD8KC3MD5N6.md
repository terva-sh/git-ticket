---
schema: 1
id: TKT-01M1SPR216X4HQBDD8KC3MD5N6
title: Make just build detect a worktree and set -buildvcs=false
type: chore
status: done
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
updated_at: 2026-09-05T22:48:23Z
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

- [x] just build succeeds inside a linked git worktree, and so do check, fix, ready, and ci, with no environment variable set by the caller.
- [x] The detection compares git rev-parse --git-dir against --git-common-dir, so the flag is passed in a linked worktree and not in the main tree.
- [x] A build in the main tree still stamps a real version: git ticket --version reports the tag rather than devel, proving the flag was not passed there.
- [x] When the flag is passed, just build says so, because the binary it produced reports devel and the existing short-SHA line reads like reassurance.
- [x] install-release and .forgejo/workflows/ci.yml are unchanged, and the justfile records why the workflow does not need the same treatment.

## Notes

**agent:terva/mieli** at 2026-09-05T22:48:23Z

Built. One correction to the description above, found before writing
any code.

### The scope claim in the description was wrong

It says "build is the only recipe that runs `go build` on the main
package". That is true of `go build` and false of the question it was
answering. `install` runs `go install ./cmd/git-ticket`, which links
the main package too and fails in a worktree the same way. Verified on
2026-09-05: `just install` in a linked worktree exits 1 with the same
`error obtaining VCS status: exit status 128`.

So the change covers `build` and `install`, and through `build` it
covers `check`, `fix`, `ready` and `ci`. No acceptance criterion had
to move, because none of them said install was out of scope. The one
that names what stays untouched names `install-release` and the CI
workflow, and both are untouched.

### How it is done

One `buildvcs` variable at the top of the justfile, so the rule exists
once rather than in each recipe:

    buildvcs := `if [ "$(git rev-parse --git-dir 2>/dev/null)" != "$(git rev-parse --git-common-dir 2>/dev/null)" ]; then echo "-buildvcs=false"; fi`

`build` and `install` interpolate it, and each prints a note on stderr
when it is non-empty. Both `git rev-parse` calls are silenced so that a
directory which is no git repository yields two empty strings, which
compare equal and add no flag.

### What was run

In the main tree: `just --evaluate buildvcs` is empty, `just build`
runs `go build  -o git-ticket` with no flag, and the binary reports
`v0.11.1-0.20260905211440-cf06b173c88f (cf06b173c88f, go1.26.2,
modified)`.

That is worth stating precisely, because criterion 3 says the main
tree reports "the tag". It reports a pseudo-version rather than a bare
tag, since HEAD is past v0.11.0 and the tree was dirty with this very
change. What the criterion is for is proving the flag was not passed,
and a pseudo-version carrying a commit and a modified marker proves the
VCS stamping ran. A bare tag would need HEAD to be at one, which a
working branch never is.

In a linked worktree, with nothing set by the caller: `just --evaluate
buildvcs` is `-buildvcs=false`, `just --list` still parses, and
`build`, `check`, `fix`, `ready`, `ci` and `install` all exit 0. Each
build prints the note, and the binary reports `devel (unknown)`.

### AGENTS.md

The entry added by the retrospective told the reader to run
`GOFLAGS=-buildvcs=false just ci` by hand. That is now unnecessary and
would have sat there as advice nobody needs, so it is rewritten to
describe the automatic behaviour. It keeps the part that is still
true: a bare `go build` in a worktree fails, because nothing
intercepts it.

## Summary

Shipped. One `buildvcs` variable in the justfile compares `git
rev-parse --git-dir` with `--git-common-dir`. They agree in the main
tree and differ in a linked worktree, so `build` and `install` pass
`-buildvcs=false` only where Go cannot stamp a version, and `check`,
`fix`, `ready` and `ci` inherit it through `build`. Nothing is set by
the caller.

The scope grew by one recipe during the build. The description claimed
`build` was the only recipe linking the main package, and `install`
does too through `go install`, failing in a worktree identically. Both
are fixed.

Each recipe says when it passed the flag, because the binary then
reports `devel (unknown)` and the `built ./git-ticket (sha)` line on
its own reads like reassurance. A devel binary is fine for checking a
store and never fine for a release.

`install-release` and `.forgejo/workflows/ci.yml` are untouched.
`install-release` builds in a clone with a real `.git` directory, and
the CI runner does an ordinary checkout, which the justfile now
records as the one place it and the workflow may differ.

AGENTS.md is updated in the same change, because its entry told the
reader to set `GOFLAGS=-buildvcs=false` by hand and that is now advice
nobody needs. It keeps the part still true: a bare `go build` in a
worktree fails, since nothing intercepts it.
