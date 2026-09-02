---
schema: 1
id: TKT-01M1F7Z31HR5GV06D6Y7WZWJK4
title: Set the compatibility policy for the module after the first stable schema
type: spike
status: done
status_reason: null
priority: normal
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
references:
  - ref: proposal:git-ticket
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-01T19:43:32Z
updated_at: 2026-09-02T14:58:23Z
created_by:
  id: human:sothr
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Deferred question 6 of plan section 15. Schema 1 is what the library reads and writes. Decide what a schema 2 means for a schema 1 reader, and what the Go module promises across it, before anything outside this repository depends on either.

## Notes

**agent:terva/mieli** at 2026-09-02T14:58:17Z

Settled by interview, not by argument. Four decisions: v1.0.0 waits for Phase 3; a file schema bump is independent of the Go module version; a store never upgrades itself; and the promise covers every machine-readable surface. Two follow-ups corrected the first pass. cli.Run and Env are covered after all, because we exported them in v0.2.0 so terva could embed them and a command surface that may break on a minor upgrade is not one a host can build on. The error and finding codes are fully covered, the same as the envelope they travel in.

**agent:terva/mieli** at 2026-09-02T14:58:17Z

The load-bearing rule is that a store never upgrades itself. Go semver only covers the Go API, so without this rule a minor upgrade could rewrite a colleague's tickets to a schema their binary refuses with schema_unsupported. That is a break no version number would have announced, because no Go API moved. Reading never rewrites, and a mutation writes back the schema the file already declared.

**agent:terva/mieli** at 2026-09-02T14:58:17Z

Settling this raised Q8, TKT-01M1H9X166M1ATNK9S7ET26BVQ: if a store only moves through an explicit migration, that operation has to exist and is undesigned. Per AGENTS.md it went to section 15 rather than being invented here.

## Summary

Answered in plan 12.4. The module version, the file schema, and the envelope schemaVersion move independently. Every machine-readable surface is covered, the ticket API, cli.Run and Env, the JSON envelope and its kinds, the exit statuses, the error and finding codes, and the on-disk format; human CLI output is not. A covered surface breaks only when something that worked stops working or changes meaning, so adding beside the old thing is always minor. The module version tracks the Go API alone, which makes a schema bump an ordinary minor release rather than a /v2, and a consumer gates at runtime on the exported ticket.SchemaVersion. A store never upgrades itself. v1.0.0 waits for Phase 3, because Q7 is a break already in sight and spending the major on it would teach a consumer that the number means nothing.
