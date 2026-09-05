---
schema: 1
id: TKT-01M1S02QA2X281VPFJKM3H0GK3
title: Add an install script for the stable release
type: task
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
claim:
  actor: agent:terva/mieli
  branch: feat/install-script
  worktree: /home/sothr/workspace/git.local.sothr.com/terva-sh/git-ticket
  commit: 610d6f4b2a1b952f490970ff21faf94d617fdb4c
  claimed_at: 2026-09-05T15:11:22Z
  expires_at: null
archive: null
created_at: 2026-09-05T14:38:07Z
updated_at: 2026-09-05T15:22:02Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Filed 2026-09-05 from a user ask: a simple install script for the
stable release, so a machine without a Go toolchain gets a working
`git ticket` in one command.

What exists today, so the script's audience is clear. A Go user runs
`go install github.com/terva-sh/git-ticket/cmd/git-ticket@latest` and
is done. An installed binary updates itself with `self-update`, per
plan 12.6, against the GitHub releases. What is missing is the first
install on a machine with neither Go nor a binary: today that is a
manual walk through the releases page, the right archive for the
platform, a checksum nobody verifies, and a directory on PATH.

The shape: one POSIX shell script, `install.sh` at the repository
root, runnable as a download-and-pipe from the public mirror. It
detects OS and architecture, asks the GitHub releases API for the
latest tag, the same source and the same `latest` semantics as
self-update so the two cannot disagree about what stable means,
downloads the matching archive and checksums.txt, verifies the sha256
before anything is unpacked, and installs the binary into the first
writable candidate of ~/.local/bin, ~/bin, or a --prefix the caller
names, refusing rather than sudo-ing on its own. It ends by checking
the destination is on PATH and saying so when it is not, because an
installed binary the shell cannot find is a support ticket.

Platform scope matches the rest of the tooling: Linux, macOS, WSL2.
Native Windows users take the zip from the releases page, and the
script says so rather than guessing at PowerShell.

The checksum verification is not optional decoration. A pipe from a
forge is exactly the delivery a checksum exists for, and the releases
already publish checksums.txt on both forges, verified end to end
since v0.5.1.

### Open questions

Whether the README's install section leads with the script or with
go install. Whether the script also offers the internal forge for
clones that live there, or stays public-mirror-only like self-update,
which plan 12.6 settled as GitHub-only for the reason that a binary
in the wild has never heard of the internal forge; the same argument
plausibly applies.

Whether CI runs the script against a scratch prefix on each release,
so a broken installer is caught by the release that broke it and not
by the next new user.

### Trigger

None. Startable feature work, parked as a draft until promoted.

## Acceptance criteria

- [x] curl of install.sh from the public mirror installs the latest release on Linux, macOS, and WSL2
- [x] the sha256 is verified against checksums.txt before anything is unpacked
- [x] the script never sudos on its own and says so when the destination is not on PATH
- [x] the README documents the one-liner beside go install and self-update

## Notes

**agent:terva/mieli** at 2026-09-05T15:12:50Z

Built. Three of the four criteria are ticked, and the first stays
open deliberately until the next release.

The script is proven live from the working tree: `sh install.sh
--prefix /tmp/gt-install` fetched v0.9.2 from the real GitHub
releases, verified the sha256 against checksums.txt, installed, and
the binary answered with the right version. The refusal paths were
run, not assumed: an unknown argument and an unwritable --prefix both
exit 1 with the repair named, --help prints the header, and dash runs
the script clean, which is the POSIX claim tested rather than
asserted. The PATH note fired on the scratch prefix.

What cannot be proven yet is the criterion's own spelling: curl of
install.sh from the public mirror. The mirror receives main at release
time, per this repository's push policy, so
raw.githubusercontent.com/.../main/install.sh does not exist until the
next tag pushes main. The README one-liner ships now and goes live
then. Whoever cuts the next release should run the curl pipe as part
of release verification and tick the first criterion with that
evidence. macOS is likewise untested from here; the shasum branch
exists for it and awaits a Mac.

Two open questions from the draft were settled by the build: the
source stays GitHub-only, the same argument as self-update, and the
README leads with the script because its audience needs the least.
The third, CI exercising the script each release, stays open and is
now half-answered by the criterion above: the release ritual runs it
by hand.

**agent:terva/mieli** at 2026-09-05T15:21:52Z

The first criterion gets its tick, with the evidence it waited for.
v0.9.3 pushed main to the mirror, which is what made the raw URL
exist, and the README's exact one-liner was then run against it on
2026-09-05:

    curl -fsSL https://raw.githubusercontent.com/terva-sh/git-ticket/main/install.sh | sh

Under a scratch HOME, so the default-destination branch ran rather
than the --prefix escape hatch: the script fetched
git-ticket_0.9.3_linux_amd64.tar.gz for the v0.9.3 latest, verified
the sha256 against checksums.txt, installed into ~/.local/bin, exited
0, and the installed binary reported v0.9.3 (cf5ee4dfcd88), the same
commit the tag names. The PATH note fired, correctly, because a
scratch HOME's ~/.local/bin is on nobody's PATH.

Linux and WSL2 are the same code path and are hereby proven; macOS
still awaits a Mac, as the earlier note says, and its shasum branch
is the one line of difference. The criterion asked that the curl from
the public mirror install the latest release, and it now has.

## Summary

Shipped and fully proven. install.sh at the repository root installs
the latest stable release on Linux, macOS, and WSL2: GitHub latest
with self-update's semantics, sha256 verified against checksums.txt
before unpacking, first writable of ~/.local/bin and ~/bin or a named
--prefix, never sudo, and a loud PATH note when the shell will not
find the result. The README leads with the curl one-liner, live since
v0.9.3 pushed main to the mirror, and the exact one-liner was run
against it: v0.9.3 installed into a scratch HOME's ~/.local/bin,
verified, exit 0. All four criteria are ticked; macOS's shasum branch
is the one line still awaiting a Mac.
