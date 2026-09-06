---
schema: 2
id: TKT-01M1WE1JSC8GNR5BWM0RGJB3FT
title: Install from source where install.sh installs, with a DIR override
type: task
status: done
status_reason: Filed and worked on build/one-install-destination in one PR, the convention for a unit of work here.
priority: normal
due_on: null
labels: []
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-06T22:39:53Z
updated_at: 2026-09-06T22:43:48Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

`just install` writes to GOBIN, and `install.sh` and `just install-release`
write to the first writable of ~/.local/bin and ~/bin. Two binaries named
git-ticket then sit on PATH and their order decides what `git ticket` means.

On the machine this was filed from, `just install` was a no-op for the
subcommand. ~/.local/bin is at PATH position 6 and ~/go/bin at 8, so the
release binary answered and the freshly built one did not:

    /home/sothr/.local/bin/git-ticket   v0.12.0 (58169e5f, go1.25.0)
    /home/sothr/go/bin/git-ticket       v0.12.1-0.20260906191953-c0547c47

TKT-01M1SMAPV96V9SPB0BF7E78ZFQ (Warn that just install and install.sh write
to different directories) shipped the warning that reports this, and the
warning fired correctly. What it reports is that the recipe did not do what
the person ran it for, which is a defect rather than a hazard to document.

### The decision

Settled with the user 2026-09-06. All three installers resolve one
destination the way install.sh does: `--prefix` or DIR if given, else the
first writable of ~/.local/bin and ~/bin, never sudo. `just install` stops
calling `go install` and calls `go build -o` into that destination, so
whatever was installed last is what `git ticket` means.

`just install DIR` overrides it, so `just install ~/go/bin` keeps the Go
convention for anyone who wants it. An empty DIR is the default resolution.

Swapping `install` and `install-release` was considered and declined.
`install-release` exists so that a tag built from source lands in the same
file a downloaded release lands in, and sending it to GOBIN would remove the
only reason it is a separate recipe. The asymmetry is not the defect.

Cleaning up an existing stale copy is out of scope, by the same ruling. No
`just uninstall`, and no recipe deletes a binary it did not write. The
warning stays, and it now names a stale copy elsewhere on PATH rather than
an unreachable GOBIN.

## Acceptance criteria

- [x] just install writes to the first writable of ~/.local/bin and ~/bin, the same file install.sh and install-release write
- [x] just install DIR writes to the named directory, creating it, and refuses a directory it cannot write
- [x] The installed binary reports a version, so go build stamping survives the move off go install
- [x] The swap is atomic, so installing over a running git-ticket does not fail with ETXTBSY
- [x] A git-ticket elsewhere on PATH that shadows the install is named with its version and the just install DIR that would write there
- [x] No recipe deletes a binary it did not write

## Definition of done

- [x] just ci green
- [x] AGENTS.md and README.md describe one destination rather than three

## Notes

**agent:terva/mieli** at 2026-09-06T22:43:43Z

Built and verified. All six criteria and both definition-of-done items tick.

`just install` now takes an optional DIR, resolves the destination the way
install.sh does, and builds with `go build -o` into a temporary directory
before swapping the result into place through a dot-prefixed name in the
destination. The `{{buildvcs}}` handling carries over unchanged, so a linked
worktree still builds and still says it produced a devel binary.

What was actually run, rather than reasoned about:

The atomic swap was proven with its counterfactual, because a pass without
one proves nothing. `git-ticket ui` was held open under a pty with `setsid
script -qec`, which is what makes the file busy. A plain `cp` onto it then
failed with "Text file busy", exit 1. `just install` under identical
conditions exited 0 and left one file in the destination with no
`.git-ticket.new` beside it.

The default resolution was proven by running it: it wrote
~/.local/bin/git-ticket, and `which git-ticket` then answered that path with
the working-tree version. The GOBIN copy at ~/go/bin had been removed by the
user by then, so the shadow warning correctly stayed silent. Both earlier
runs into /tmp printed it in full, naming the shadowing path, its version,
and the `just install DIR` that would write there instead.

An unwritable destination was checked with /proc/nope, exit 1. A destination
that does not exist yet was checked with /tmp/gtdest, created and written.

One measurement error worth recording, since it is the pipeline gotcha in
AGENTS.md arriving in a new costume. The first ETXTBSY run reported exit 101
and looked like a failure of the recipe. It was `head -2` closing the pipe,
which sends SIGPIPE into a recipe running under `set -euo pipefail`. The
recipe had already succeeded. Reading a just recipe's exit status through
`head` measures the head.

**agent:terva/mieli** at 2026-09-06T22:43:48Z

draft to done: Filed and worked on build/one-install-destination in one PR, the convention for a unit of work here.

## Summary

Shipped. `just install` builds the working tree and writes it where
install.sh writes, so there is one file named git-ticket and whatever ran
last is what `git ticket` means. `just install DIR` names a destination
instead, which is how to keep GOBIN.

The defect was that `just install` ran `go install`, which writes to GOBIN.
Where ~/.local/bin comes first on PATH, the recipe built the tree, wrote the
binary where nothing looks, and left `git ticket` meaning the release
install. TKT-01M1SMAPV96V9SPB0BF7E78ZFQ's warning reported this correctly
every time, which is how a working warning sat on top of a broken recipe.

The warning stays and now means a copy an older convention left behind. It
names the copy, its version, and the `just install DIR` that would write
there. It deletes nothing, and no recipe here removes a binary it did not
write.

Swapping `install` and `install-release` was declined. `install-release`
exists so a tag built from source lands in the same file a downloaded release
lands in, and GOBIN would take away its only reason to be a separate recipe.

AGENTS.md and README.md describe one destination. `go install
...cmd/git-ticket@latest` still writes to GOBIN, because that is Go's rule
rather than this project's, so both files still say how two copies happen.
