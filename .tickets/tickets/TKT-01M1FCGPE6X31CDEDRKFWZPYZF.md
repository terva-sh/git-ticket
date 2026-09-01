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
updated_at: 2026-09-01T21:48:44Z
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

## Notes

**agent:terva/mieli** at 2026-09-01T21:48:44Z

Both blockers are gone. The mirroring question is settled in TKT-01M1FCJ8Y: terva ticket mirrors all 24 commands by embedding the command surface, so this slice is one call to cli.Run rather than 24 reimplementations. The description above still says it is blocked on that question; it is not.

**agent:terva/mieli** at 2026-09-01T21:48:44Z

v0.2.0 is tagged and published, on the internal Forgejo and the public mirror at identical SHAs, and it resolves from the module proxy. It is the first tag carrying the cli package, which was internal/cli and unreachable through v0.1.0. Require v0.2.0 in terva go.mod. Verified from a module outside the repository with no replace directive: cli.Run returned 0 for ready and 1 for a missing ticket, and ticket.Store.Ready returned structured tickets. The ticket package is byte-identical to v0.1.0, so the bump is additive.
