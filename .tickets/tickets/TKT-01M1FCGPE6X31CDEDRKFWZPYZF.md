---
schema: 1
id: TKT-01M1FCGPE6X31CDEDRKFWZPYZF
title: "Slice 1: add the module dependency and terva ticket"
type: task
status: ready
status_reason: null
priority: high
labels:
  - integration
assignees: []
milestone: null
parent: TKT-01M1FCHH3QN04AYHZAP1M8DNQK
dependencies:
  - TKT-01M1FCJ8YGZN9MG9STWBSX308J
references: []
claim: null
archive: null
created_at: 2026-09-01T21:03:03Z
updated_at: 2026-09-01T21:04:21Z
created_by:
  id: human:sothr
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Add github.com/terva-sh/git-ticket to terva go.mod and build a terva ticket command that delegates to the library. No tools and no board in this slice.

This is useful on its own: it gives the agent a working ledger through bash, and it proves the import in terva build and CI before any tool surface is committed. The module is published and resolves from a clean environment, verified with go get from a throwaway module outside the repository.

The command must not grow a second parser, renderer, or schema. Where terva and git ticket disagree about the format, plan docs/plan.md is right and terva is the bug.

Blocked in practice on deciding how much of the 24-command CLI terva ticket mirrors.
