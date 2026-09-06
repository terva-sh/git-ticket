---
schema: 2
id: TKT-01M1W3TM6CNQDQT11NKG2GKWH2
title: Ship shell completion for bash, zsh, fish, and PowerShell
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
created_at: 2026-09-06T19:41:19Z
updated_at: 2026-09-06T19:41:19Z
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

- [ ] git ticket completion SHELL prints a working script for bash, zsh, fish, and PowerShell
- [ ] Command names and every enum value are read from git ticket schema rather than restated in the script
- [ ] Completing a ticket ID offers the abbreviation ShortestUnique produces, not the full ULID
- [ ] Installing it is documented in the README for each shell

## Definition of done

- [ ] NOTICE credits Backlog.md for the idea, with no file header because no code was adapted
- [ ] The script is exercised in at least one real shell, not only unit-tested
