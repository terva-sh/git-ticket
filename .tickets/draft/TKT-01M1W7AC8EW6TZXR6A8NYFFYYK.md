---
schema: 2
id: TKT-01M1W7AC8EW6TZXR6A8NYFFYYK
title: Ship shell completion for fish and PowerShell
type: task
status: draft
status_reason: null
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
created_at: 2026-09-06T20:42:21Z
updated_at: 2026-09-06T20:42:21Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Ship completion for fish and PowerShell, the half deferred from
TKT-01M1W3TM6CNQDQT11NKG2GKWH2 (Ship shell completion for bash, zsh, fish,
and PowerShell). That ticket shipped bash and zsh and reworded its first
criterion to say so, because four scripts is four things to keep correct and
nobody had asked for these two.

PowerShell is not idle speculation. `.goreleaser.yaml` builds
`goos: [linux, darwin, windows]`, so a released Windows binary exists and its
users have no completion at all.

### What is already decided, so this ticket does not reopen it

The surface is settled by the parent ticket. `git ticket completion SHELL`
prints a script to stdout, `--install` writes it and refuses when a file is
already there unless `--force` is given, and `completion --dump` is the single
private source of truth for command names, flags and enum values, in plain
lines with `--json` for the same data. A new shell is a new generator reading
that same dump, not a new introspection path.

ID completion is store-scoped, prefix-filtered, capped at 200, and offers
`ShortestUnique` abbreviations. Whether these two shells can show titles
beside candidates the way zsh does is theirs to answer.

### What is genuinely open, and is why this is not a copy of the parent

The parent ships one file per shell serving both `git-ticket <TAB>` and
`git ticket <TAB>`, because bash and zsh each have a documented dispatch
convention for a Git subcommand: bash looks for `_git_ticket` and autoloads a
file named `git-ticket`, and zsh calls `_git-ticket` and scans `$fpath` for
`_git-*`. Neither fish nor PowerShell has an equivalent convention, so
whether `git ticket <TAB>` can work at all in them is the first thing to find
out, and the answer may be no.

fish uses `complete -c` with its own flag vocabulary rather than a completion
function returning candidates, so the generator is a different shape rather
than a port. PowerShell uses `Register-ArgumentCompleter`.

The parent's research also found that supporting the `git foo <TAB>` form is a
minority practice even in bash and zsh, and that the projects doing it warn
they lean on `_git` internals. Do not assume a fish or PowerShell equivalent
exists because bash has one. Establish it before promising it in a criterion.

### Trigger

Somebody asking for one of these shells. Naming which shell is the evidence,
because a generator written for a shell nobody runs is a script nobody tests.

## Acceptance criteria

- [ ] Whether git ticket TAB can work at all in fish and in PowerShell is established from source and recorded, before any criterion promises it
- [ ] git ticket completion fish and git ticket completion powershell each print a working script
- [ ] Both generators read completion --dump rather than restating commands, flags, or enum values
- [ ] Installing each is documented in the README, and each is exercised in that real shell

## Definition of done

- [ ] The parent ticket's first criterion is no longer the only record that these two were deferred
