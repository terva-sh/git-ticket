---
schema: 1
id: TKT-01M1SMR8D337KEP9KNWEC9A3TT
title: Add just install-release, a source build into install.sh's location
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
created_at: 2026-09-05T20:39:24Z
updated_at: 2026-09-05T20:48:03Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Add `just install-release [TAG]`, which builds a tagged release from
source and installs it where `install.sh` would have put a downloaded
one. It is the build-it-yourself alternative to the README's curl
one-liner, for a machine that has Go and would rather compile than
trust a prebuilt archive.

The existing `just install` stays exactly as it is. It runs `go
install` into GOBIN, which is the right thing for dogfooding an
untagged working tree, and it is not what this ticket is about.

### Why the destination matters

Installing to GOBIN and installing to ~/.local/bin are not the same
act, because PATH order decides which binary `git ticket` means.
TKT-01M1SMAPV96V9SPB0BF7E78ZFQ records the confusion that causes,
observed the same day. Sending `install-release` to the same place
`install.sh` sends its download means the two cannot leave two
binaries of different ages behind.

### The decisions, settled with the user on 2026-09-05

The destination is the first writable of ~/.local/bin and ~/bin, and
the recipe never sudos. That is `install.sh`'s own resolution and its
stated no-sudo stance, so agreeing with it is the whole point. A real
system path such as /usr/local/bin was offered and declined.

The name is `install-release`.

TAG is an optional argument. The build happens in a temporary git
worktree checked out at that tag, so the current checkout is never
touched: no branch switch, no stash, nothing to put back. With no
argument the recipe uses the tag at HEAD and refuses if there is not
one.

Nothing that is not a tag is installed. That is the requirement the
whole recipe exists to hold, because `go build` derives the version
from what the VCS says, so an untagged commit produces a
pseudo-version and a shallow checkout produces `devel`. Plan 12.1
records that mechanism.

### The dirty-tree question, and why the answer moved

The user asked that a dirty working tree be refused, reasoning that a
binary stamped `modified: true` is not the release it names. That
reasoning is right and the check turned out to be the wrong way to
enforce it, because of the worktree decision above. A temporary
worktree at a tag is clean by construction, so the state of the
current checkout cannot reach the binary, and refusing on it would
block a legitimate install for no mechanical reason.

So the recipe checks the artifact rather than the proxy. It reads
`--version --json` from the binary it just built and refuses to
install unless the version equals the requested tag and `modified` is
false. That enforces the user's intent strictly, and it also catches
failures a dirty-tree check never could, such as a build with no VCS
information at all.

It reads the JSON and not the human line on purpose. Plan 12.4 says
human-readable output may be reworded or reordered in any release and
is not a covered surface, so a recipe that grepped the pretty version
string would be parsing an interface this project does not offer.

## Acceptance criteria

- [x] just install-release with HEAD at a tag builds that tag and installs it to the first writable of ~/.local/bin and ~/bin, never sudoing.
- [x] just install-release TAG installs that tag from wherever HEAD is, and the current checkout is unchanged afterwards: same branch, no new files, and git worktree list shows only the main tree.
- [x] The recipe refuses with exit 1 and a message naming the problem when HEAD carries no tag and no TAG is given, when TAG names a branch, and when TAG names a tag that does not exist locally.
- [x] The recipe verifies the built binary by reading --version --json, not the human line, and refuses to install unless version equals the requested tag and modified is false. Nothing reaches the destination when that check fails.
- [x] just --list describes the recipe, and a justfile comment says why its destination differs from just install.

## Notes

**agent:terva/mieli** at 2026-09-05T20:46:42Z

Built, and one of the four settled decisions changed during the
build because the mechanism it named does not work.

### The temporary worktree does not work

`go build` inside a linked git worktree fails outright:

    error obtaining VCS status: exit status 128
    Use -buildvcs=false to disable VCS stamping.

The cause is that Go finds a VCS root by looking for a `.git`
directory, and a linked worktree has a `.git` file pointing into the
main repository instead. `go build -x` names the failing command as
`git status --porcelain`, and every git command Go could run there
succeeds when run by hand in that same directory, including `status
--porcelain`, `rev-parse --show-toplevel`, `tag --points-at HEAD` and
`describe --tags --dirty`. So the failure is in Go's detection rather
than in git.

`-buildvcs=false` is not an answer here. It disables the very
stamping this recipe exists to check.

### What replaced it

A throwaway local clone at the tag, `git clone --local --branch
$tag`, which has a real `.git` directory. Built from a deliberately
dirty checkout it stamped version v0.11.0 with modified false, which
is the outcome the worktree was chosen to produce.

The decision the user made was that the current checkout must not be
touched, and the worktree was the mechanism named for it. The clone
keeps that outcome and improves on it in one way worth recording:
`git worktree add` writes into this repository's .git and needs
cleaning up afterwards, and `git clone` writes nothing here at all.
The recipe's cleanup is now one `rm -rf` on a temp directory.

### The dirty-tree refusal, resolved rather than dropped

The user asked that a dirty tree be refused, on the reasoning that a
binary stamped modified true is not the release it names. Because the
build happens elsewhere, the checkout's state cannot reach the
binary, so refusing on it would block a legitimate install for no
mechanical reason. The recipe checks the artifact instead: it reads
`--version --json` from the binary it just built and refuses unless
version equals the requested tag and modified is false. That is
strictly stronger, since it also catches a build with no VCS
information at all.

Proven from a dirty checkout, which is how the end-to-end run above
was made.

### What was tested

Five refusals, each exiting 1 with a message naming the problem: HEAD
at no tag with no argument, a branch name, an unknown tag, a bare
commit SHA, and an unwritable HOME. After all five,
~/.local/bin/git-ticket was unchanged by sha256 and by mtime, so no
refusal path writes.

The success path installed v0.11.0 and left the checkout on the same
branch at the same commit with the same two dirty files, and `git
worktree list` showed only the main tree.

The PATH shadow warning was exercised with a decoy git-ticket earlier
on PATH. It named the decoy and reported its version, which is the
warning TKT-01M1SMAPV96V9SPB0BF7E78ZFQ is about.

## Summary

Shipped. `just install-release [TAG]` builds a release tag from
source and installs it to the first writable of ~/.local/bin and
~/bin, which is install.sh's own resolution, and it never sudos. With
no argument it uses the tag at HEAD and refuses when there is not
one. `just install` keeps its GOBIN behaviour untouched.

Nothing that is not a tag is installed. The tag is verified through
refs/tags before anything is built, so a branch, an unknown tag and a
bare commit SHA are all refused with exit 1.

The build runs in a throwaway local clone at the tag rather than in a
temporary worktree, which was the mechanism first chosen and does not
work: Go looks for a `.git` directory to find the VCS root, a linked
worktree has a `.git` file, and `go build` there fails with "error
obtaining VCS status: exit status 128" instead of stamping. The clone
has a real `.git` directory, and unlike `git worktree add` it writes
nothing into this repository.

The result is proven on the artifact rather than on the tree. The
recipe reads `--version --json` from the binary it just built and
refuses to install unless the version equals the requested tag and
modified is false, so a dirty checkout cannot produce a binary that
lies about what it is. It reads the JSON because plan 12.4 does not
cover the human line.

After a successful install it warns if the destination is not on
PATH, and warns if some other git-ticket comes first on PATH and
still owns the `git ticket` spelling.

The README gained a paragraph beside the existing install paths.
