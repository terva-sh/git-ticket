---
schema: 1
id: TKT-01M1PCXW46GVCKAZ6HEKNNSB21
title: Let a body section be read from a file or stdin
type: task
status: draft
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
updated_at: 2026-09-04T14:24:56Z
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

- [ ] A description longer than a line can be written without shell quoting hazards
- [ ] The chosen shape covers create, update, plan, summary, note and comment
- [ ] docs/plan.md section 12.1 records the input convention
