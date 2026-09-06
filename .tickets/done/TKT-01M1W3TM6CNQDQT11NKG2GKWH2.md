---
schema: 2
id: TKT-01M1W3TM6CNQDQT11NKG2GKWH2
title: Ship shell completion for bash, zsh, fish, and PowerShell
type: task
status: done
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
created_at: 2026-09-06T19:41:19Z
updated_at: 2026-09-06T21:00:26Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Ship shell completion, from section D of `docs/review-backlog-md.md`.

Backlog.md has `completion install` for bash, zsh, fish, and PowerShell,
completing command names, live task IDs, and enum values. We have nothing:
there is no `completion` command among the 33 this binary carries, verified
against `git ticket --help`.

Most of the work is static. The enums are fixed and `git ticket schema`
already publishes them, the statuses, types, priorities, error codes and
finding codes, so a generator can read that rather than restating the lists
and drifting from them. Plan 10.4 requires `schema` to read no store, so the
generator works before `init` and outside a repository.

The dynamic half is ticket IDs, and it pairs with work that exists.
`ticket.ShortestUnique` shortens an ID to what actually resolves across the
store, and plan 5.5 says any command taking an ID accepts a unique prefix, so
completion should offer the same abbreviations the CLI accepts rather than
full ULIDs a person cannot read.

This is an idea lift and not a code lift. Backlog.md's implementation is
TypeScript and nothing here would be adapted from it, so under the ruling on
this work it takes a NOTICE entry and no file header.

Worth settling before building: whether completing IDs shells out to the
binary on every tab, which is a store read per keystroke, and what that costs
on a store larger than this one's 92 tickets.

## Acceptance criteria

- [x] git ticket completion SHELL prints a working script for bash and zsh
- [x] Command names, flags, and every enum value come from completion --dump rather than being restated in the script
- [x] Completing a ticket ID offers the abbreviation ShortestUnique produces, not the full ULID
- [x] Installing it is documented in the README for each shell
- [x] Both git-ticket TAB and git ticket TAB complete, in bash and in zsh
- [x] Install refuses when a completion file already exists, and overwrites only under --force

## Definition of done

- [x] NOTICE credits Backlog.md for the idea, with no file header because no code was adapted
- [x] The script is exercised in at least one real shell, not only unit-tested

## Notes

**agent:terva/mieli** at 2026-09-06T20:42:12Z

Acceptance criteria reworded before the build, after the grilling session
settled the design. The originals are recorded here verbatim, because a reader
of this ticket should not have to run `git log` to learn what was asked.

### The two that changed

Criterion 1 read, verbatim:

"git ticket completion SHELL prints a working script for bash, zsh, fish, and
PowerShell"

It now names bash and zsh alone. This is a scope decision taken before the
branch, not a retreat during it: shipping four scripts means four things to
keep correct, and fish and PowerShell have no user asking for them yet. The
deferred half is filed as its own ticket rather than left as a sentence here,
so it is tracked. PowerShell is not idle speculation, since the release does
build windows/amd64.

Criterion 2 read, verbatim:

"Command names and every enum value are read from git ticket schema rather
than restated in the script"

It was unsatisfiable as written. `git ticket schema` publishes the enums, the
statuses, types, priorities, error codes and finding codes, but it does not
publish command names and it does not publish flags. Satisfying it literally
would have meant enlarging `schema`, which plan 10.4 defines as what the
binary enforces and which 12.4 then covers forever. The design instead puts
the command table, the flags and the enums behind `completion --dump`, which
stays private. The criterion now names that.

### Two that were added

Both invocation forms, and the install refusal. Neither was in the ticket as
filed, and both are things the grilling session decided rather than things
the build chose for itself, so they belong in the criteria as evidence of
what was asked.

### Two that did not change

Criterion 3, the `ShortestUnique` abbreviation, stands exactly as written. It
was flagged as at risk during the interview on the assumption that ID
completion might be dropped. It was not dropped, so the criterion is
satisfiable and stays.

Criterion 4, the README documenting installation per shell, stands as
written.

**agent:terva/mieli** at 2026-09-06T21:00:26Z

Built and verified in both shells. All six criteria and both definition-of-done
items are ticked.

### The seam that made it cheap

The design question was how a script learns the commands and flags without
restating them. The interview settled on a private introspection surface, and
the cost estimate given at the time was wrong: enumerating flags looked like it
needed registration split from execution across 33 commands.

It did not. Each command already passes a register closure to
`ctx.parseFlags`, and `--help` prints a full flag list with no store. Measured
rather than assumed: `git ticket update --help --store /nonexistent` exits 0 and
lists all 19 flags, while `list` against the same store exits 1. So a
`captureFlags` hook on `cmdContext` receives each command's registered FlagSet
and stops it before it opens anything. `completion --dump` walks all 36 commands
that way, needs no store and no repository, and walked `ui` without launching a
TUI.

That also moved the introspection surface from a hidden command to
`completion --dump`, a flag on the command whose job is generating scripts.
Nothing is invisible in help and nothing new is published.

### What the research changed

A research pass established the two dispatch mechanisms from git's and zsh's
own source, and driving them turned up things worth recording. bash computes `_git_${command//-/_}` and
autoloads a file named after the binary; zsh calls `_git-ticket` with a hyphen
and scans `$fpath` for `_git-*`. The two function names cannot be shared. git
registers its wrapper with `-o nospace`, so the script adds trailing spaces
itself, and the two bash entry points see different variables, which is why
`_git_ticket` normalizes through `$__git_cmd_idx`.

zsh was not installed when the research ran, so its single-file claim was
inferred. It is now executed: `words=(ticket show "")` and
`words=(git-ticket show "")` return identical candidates.

### One decision whose mechanism was substituted

The interview settled that install should warn when it can see a git-shipped
`_git` shadowing zsh's. Reading the real `$fpath` means executing zsh, and
`TestGitCommandsAreReadOnly` refused it: plan 7.4 allows exactly one non-git
exec, the clipboard tools of 12.7. A warning is not worth a second exception to
a deliberately tight rule.

The outcome was kept and the mechanism changed. `shadowingGit` reads the
locations such a file actually occupies, including the Homebrew path the
conflict is famous for, and detects git's version by the `git-completion.bash`
it sources. It cannot see `$fpath` ordering, so the warning says "if it comes
before" rather than asserting. The plan is unchanged, which was the point.

### Derived rather than written down twice

Which commands offer ID completion comes from the usage line beginning with
`ID`, so a new command inherits it. Positional enums come from the same line:
`status ID STATUS` offers the statuses because `STATUS` lowercases to an enum
name. Nothing hardcodes a command list.

### Scope

bash and zsh ship. fish and PowerShell are TKT-01M1W7AC8EW6TZXR6A8NYFFYYK, and
criterion 1 was reworded before the branch to say so, with the original in an
earlier note.

## Summary

Shipped. `git ticket completion bash|zsh` prints a completion script, and
`--install` writes it where the shell looks: for bash the XDG bash-completion
directory, for zsh an explicit `--dir` it refuses to guess, because zsh has no
standard location and the directory must be on `$fpath` before compinit. It
refuses an existing file and overwrites only under `--force`.

Both invocation forms complete in both shells, `git-ticket <TAB>` and
`git ticket <TAB>`, from one file per shell. The file names are load-bearing:
bash's loader asks for a file named after the binary and zsh's `_git` scans
`$fpath` for `_git-*`, so `--install` chooses them.

Completion offers command names, the flags of the command in hand, enum values
for flags and positionals, and ticket IDs as the same shortest-unique
abbreviations `show` accepts, so it can never offer a candidate the CLI would
reject. zsh shows titles beside IDs; bash shows IDs alone.

Everything derives from `completion --dump`, a private surface that is not a
published JSON kind. It needed no refactor: a `captureFlags` hook stops each
command inside `parseFlags` once its flags exist, so the dump reads real flag
definitions with no store and no repository.

Verified by running both shells, not only by unit tests. bash-completion's
loader registered the file and completed all four candidate kinds; zsh 5.9
autoloaded `_git-ticket`, its `_git` fpath glob found the file and read its
description, and both word shapes returned identical candidates.

One mechanism was substituted mid-build. The install-time warning about a
git-shipped `_git` shadowing zsh's was to read `$fpath` by running zsh, which
plan 7.4 forbids: it allows one non-git exec and that is the clipboard. The
warning now checks the locations such a file occupies, including Homebrew's,
and says "if it comes before" rather than claiming an order it cannot see. The
plan is unchanged.

fish and PowerShell are TKT-01M1W7AC8EW6TZXR6A8NYFFYYK.
