---
schema: 1
id: TKT-01M1S02QA2X281VPFJKM3H0GK3
title: Add an install script for the stable release
type: task
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
created_at: 2026-09-05T14:38:07Z
updated_at: 2026-09-05T14:38:07Z
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

- [ ] curl of install.sh from the public mirror installs the latest release on Linux, macOS, and WSL2
- [ ] the sha256 is verified against checksums.txt before anything is unpacked
- [ ] the script never sudos on its own and says so when the destination is not on PATH
- [ ] the README documents the one-liner beside go install and self-update
