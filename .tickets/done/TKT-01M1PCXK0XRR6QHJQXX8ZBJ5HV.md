---
schema: 1
id: TKT-01M1PCXK0XRR6QHJQXX8ZBJ5HV
title: git ticket --help exits 1 while every subcommand answers it
type: bug
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
created_at: 2026-09-04T14:24:47Z
updated_at: 2026-09-04T14:43:19Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

The bare binary refuses the flag every other level accepts.

git ticket --help exits 1 and prints "git-ticket: flag: help requested". git ticket help exits 0 and prints the full command list. Every subcommand answers --help correctly and exits 0, verified against create, list and check.

That asymmetry shipped in v0.5.0. Commit 954245e, answer --help on a subcommand instead of refusing it, fixed the subcommand layer and left the top level alone.

It matters more than its size because --help is the first thing somebody types against an unfamiliar binary, and a tool whose pitch is legibility failing on it reads as broken rather than as a papercut. It was the first command run while reviewing ergonomics for adoption by another project, and it failed.

The note in cli.go about flag.Parse stopping at the first non-flag word explains why parseFlags loops. The top level is where a help request becomes an error rather than the text that the help command already prints.

## Acceptance criteria

- [x] git ticket --help exits 0 and prints the command list
- [x] git ticket -h behaves the same as git ticket help
- [x] A test covers both spellings so the asymmetry cannot return

## Summary

Fixed in Run. The top-level parse ran before the command name was read, and the standard library reports -h and --help from that parse as flag.ErrHelp, which fell straight into fail() as "flag: help requested" with exit 1. Run now catches flag.ErrHelp there, writes the usage to stdout, and returns exitOK, which mirrors what parseFlags already did one level down.

The name check at cli.go:167 was unreachable from the leading position all along, because Parse consumes the flag before a command name exists. It stays, because `git ticket -- --help` does reach it, and TestTopLevelHelp now exercises that form so it cannot rot unnoticed.

Both spellings match `git ticket help` byte for byte. That is asserted rather than approximated, because two spellings of one question answering differently is a second thing to learn.

The test was proved to catch the bug rather than assumed to. Stashing the cli.go hunk and rerunning it failed on all three counts: exit 1, the stderr text, and the empty stdout.

Two guards keep the blast radius honest. A bare invocation with no arguments still exits nonzero with usage on stderr, because that is a mistake rather than a question. And --nope is still refused, so this did not turn every parse error into a help page.
