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
updated_at: 2026-09-02T17:17:33Z
created_by:
  id: human:sothr
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

A deferred question in plan section 15. Schema 1 is what the library reads and writes. Decide what a schema 2 means for a schema 1 reader, and what the Go module promises across it, before anything outside this repository depends on either.

## Notes

**agent:terva/mieli** at 2026-09-02T14:58:17Z

Settled by interview, not by argument. Four decisions: v1.0.0 waits for Phase 3; a file schema bump is independent of the Go module version; a store never upgrades itself; and the promise covers every machine-readable surface. Two follow-ups corrected the first pass. cli.Run and Env are covered after all, because we exported them in v0.2.0 so terva could embed them and a command surface that may break on a minor upgrade is not one a host can build on. The error and finding codes are fully covered, the same as the envelope they travel in.

**agent:terva/mieli** at 2026-09-02T14:58:17Z

The load-bearing rule is that a store never upgrades itself. Go semver only covers the Go API, so without this rule a minor upgrade could rewrite a colleague's tickets to a schema their binary refuses with schema_unsupported. That is a break no version number would have announced, because no Go API moved. Reading never rewrites, and a mutation writes back the schema the file already declared.

**agent:terva/mieli** at 2026-09-02T14:58:17Z

Settling this raised the explicit schema migration question, TKT-01M1H9X166M1ATNK9S7ET26BVQ: if a store only moves through an explicit migration, that operation has to exist and is undesigned. Per AGENTS.md it went to section 15 rather than being invented here.

**agent:terva/mieli** at 2026-09-02T17:17:33Z

The stated reason for staying on v0.x has changed. This ticket settled v1.0.0 waits for Phase 3 on the grounds that the parent hierarchy was a break already in sight. Running that question shrank it to an additive parent filter on list, and 10.1 did not move, so the break never arrived. Plan 12.4 and the summary here now rest v0.x on a different reason: no consumer has exercised a covered surface yet, so the promise v1.0.0 makes about those surfaces is a guess until one has. The decision is unchanged.

## Summary

Answered in plan 12.4. The module version, the file schema, and the envelope schemaVersion move independently. Every machine-readable surface is covered, the ticket API, cli.Run and Env, the JSON envelope and its kinds, the exit statuses, the error and finding codes, and the on-disk format; human CLI output is not. A covered surface breaks only when something that worked stops working or changes meaning, so adding beside the old thing is always minor. The module version tracks the Go API alone, which makes a schema bump an ordinary minor release rather than a /v2, and a consumer gates at runtime on the exported ticket.SchemaVersion. A store never upgrades itself. v1.0.0 waits for Phase 3, because no consumer has exercised a covered surface yet and the promise v1.0.0 makes about those surfaces is a guess until one has. The break that was in sight when this was settled, the parent hierarchy TKT-01M1FCMN7QEWM584N192NBC7TD, never arrived, because the answer was an additive parent filter on list and 10.1 did not move.
