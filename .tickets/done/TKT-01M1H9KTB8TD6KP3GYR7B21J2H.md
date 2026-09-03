---
schema: 1
id: TKT-01M1H9KTB8TD6KP3GYR7B21J2H
title: "Set up the local dev loop: a justfile and an install path"
type: chore
status: done
status_reason: null
priority: normal
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
references:
  - ref: plan:git-ticket
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-02T14:50:49Z
updated_at: 2026-09-02T14:53:13Z
created_by:
  id: agent:terva/dev-loop
  name: ""
updated_by:
  id: agent:terva/dev-loop
  name: ""
extensions: {}
---

## Description

The repository has a CI workflow and a README build line, and nothing in between. A person working here types the go invocations from memory, and the only way to get `git ticket` on PATH is to build into the repository root and copy it somewhere. Give the tree a justfile that names the same steps CI runs, plus an install recipe, so the local loop and the gate are the same commands.

## Acceptance criteria

- [x] A justfile names every step CI runs, and just ci is green locally
- [x] just install puts git-ticket on PATH so git ticket resolves as a Git subcommand
- [x] AGENTS.md and README name the recipes rather than the raw go invocations

## Notes

**agent:terva/dev-loop** at 2026-09-02T14:53:13Z

just --list shows only the last comment line above a recipe, so the one-line summary goes last and the prose above it. The first draft documented every recipe with the tail of a paragraph.

**agent:terva/dev-loop** at 2026-09-02T14:53:13Z

go install github.com/terva-sh/git-ticket/cmd/git-ticket@latest resolves against the public mirror and pulls v0.2.0, tested into a scratch GOBIN. The README says so because it was run, not because it should work.

## Summary

A justfile mirrors the four CI steps as lint, test-race, check, and ci, adds build, install, test, fmt, tidy, ready, and clean, and the README and AGENTS.md name the recipes. just ci is green in 4s. just install puts git-ticket in GOBIN, where Git resolves it as git ticket.
