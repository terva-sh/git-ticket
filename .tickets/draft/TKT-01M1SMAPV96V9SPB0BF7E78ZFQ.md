---
schema: 1
id: TKT-01M1SMAPV96V9SPB0BF7E78ZFQ
title: Warn that just install and install.sh write to different directories
type: chore
status: draft
status_reason: null
priority: low
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
created_at: 2026-09-05T20:32:00Z
updated_at: 2026-09-05T20:32:04Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

This repository has two ways to install its own binary, and they
write to different directories. Which one `git ticket` means is
decided by PATH order, silently.

`just install` runs `go install ./cmd/git-ticket`, so the binary
lands in GOBIN, which is ~/go/bin here. `install.sh` takes the first
writable of ~/.local/bin and ~/bin, per its own header, so the README
one-liner lands somewhere else. Run both, as anyone verifying a
release does, and there are two binaries with one name.

### What it looked like

Observed on 2026-09-05, immediately after tagging v0.11.0. `just
install` reported success and put v0.11.0 in ~/go/bin. `git ticket
--version` then answered v0.10.0, because ~/.local/bin came first on
PATH and still held the copy an earlier install.sh run had put there.
Nothing failed and nothing warned. `which -a git-ticket` was what
settled it.

### Why it is worth a line rather than a shrug

AGENTS.md now recommends using the installed binary as a free
before-image for a refactor that must change nothing a person sees:
`diff <(git ticket list --all) <(./git-ticket list --all)`. That
advice assumes `git ticket` is the binary installed before the
current branch. With a shadowing copy it can be any age at all, so
the diff compares the wrong pair and reports empty for the wrong
reason. An empty diff is exactly the answer the technique is looking
for, which is what makes this quiet failure expensive.

The same applies to `git ticket self-update`, which upgrades the
binary that ran it and leaves the other one where it was.

### The likely fix

A short entry in AGENTS.md beside the stale-binary gotcha, naming
both destinations and saying that PATH order picks the winner. It
does not need code. `just install` already prints its destination and
`install.sh` already prints its own, so both halves are visible; what
is missing is the warning that they can differ.

Worth checking at write time whether `just install` should print a
warning when a different git-ticket shadows GOBIN on PATH. That is a
justfile change rather than a Go one, and it may be more machinery
than the problem deserves.

## Acceptance criteria

- [ ] AGENTS.md names both install destinations, GOBIN for just install and the first writable of ~/.local/bin and ~/bin for install.sh, and says PATH order decides which one git ticket means.
- [ ] The before-image entry beside the stale-binary gotcha tells the reader to confirm with which -a git-ticket before trusting an empty diff.
- [ ] A decision is recorded either way on whether just install should warn when another git-ticket shadows GOBIN on PATH.
