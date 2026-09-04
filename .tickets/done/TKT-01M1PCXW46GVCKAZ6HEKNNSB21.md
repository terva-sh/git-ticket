---
schema: 1
id: TKT-01M1PCXW46GVCKAZ6HEKNNSB21
title: Let a body section be read from a file or stdin
type: task
status: done
status_reason: null
priority: high
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
created_at: 2026-09-04T14:24:56Z
updated_at: 2026-09-04T16:50:24Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Every command taking prose takes it only as a command-line argument. There is no --description-file, no --plan-file, and no support for a bare dash meaning stdin. Confirmed by grep over the cli package: the only file words in it are the files command and JSON keys.

This is the sharpest daily friction in a tool whose format is Markdown prose. A description of any length has to be passed as one shell argument, so the shell becomes part of the authoring surface. A single apostrophe ends a single-quoted string and a backtick inside a double-quoted string runs a command. Writing the tickets in this session meant deliberately avoiding contractions to keep the quoting intact, which is not a constraint the format asks for and not one a person will guess.

It compounds the heading trap. parseBody splits on any line starting with two hashes and a space, so long prose needs care, and fighting the shell is what prevents care.

The commands that take prose are create with --description and --plan, update with --description, and plan, summary, note and comment.

Two shapes, and this ticket should settle which. A per-flag file variant such as --description-file PATH is explicit and scales badly as flags multiply. A convention where a bare dash means stdin is smaller and matches what people expect of Unix tools, but a command taking one positional TEXT argument has to decide what a bare dash means when it is also a legal ticket body.

Either way the JSON contract does not move. This is input ergonomics and changes no output.

## Acceptance criteria

- [x] A description longer than a line can be written without shell quoting hazards
- [x] The chosen shape covers create, update, plan, summary, note and comment
- [x] docs/plan.md section 12.1 records the input convention

## Summary

Shipped. Every command that takes prose now reads it from a file or stdin, and
this summary was written with the feature: `summary ID --file -`.

The shape, settled in plan 12.1: a named flag gets a named sibling, so
`--description` is joined by `--description-file` and `--plan` by `--plan-file`.
The four commands that take the text as a positional get one `--file`, since
there is only one section it could fill. A path of `-` reads stdin.

The ticket named two candidates and asked which. The bare-dash convention alone
does not work, because a command taking one positional TEXT cannot tell a dash
meaning stdin from a dash that is the body. A sigil such as `@notes.md` has the
same defect one level down: it reads a legal body opening with `@` as a path, so
it needs an escape for its own escape. A sibling flag has neither problem, and
there are only three of them, so the scaling worry the ticket raised does not
bite.

Two refusals rather than a precedence rule. A section handed both an inline
value and a file is a usage error naming both, matching what `--depends-on` with
`--ref` already does. Two sections both naming `-` is refused before the first
read, because stdin is a stream and the second read returns nothing.

Trailing whitespace is stripped, since a final newline terminates a text file's
last line rather than being content. Worth recording that two layers do this:
`parseBody` already drops trailing blank lines, so removing the `TrimRight` in
`readProse` fails only the trailing-spaces case. The test covers both, and the
comment says which layer carries which.

The heading warning now names the flag that carried the text, so a reader is
sent to `--description-file` and not to a flag they did not type. That matters
more here than it did before: a file is where somebody writes enough prose to
want subheadings, so the `## ` trap is likelier from a file, not less.

`cli.Env` gained a `Stdin` field, defaulted to an empty reader. An embedding
host that supplies none gets an empty section rather than a panic.
