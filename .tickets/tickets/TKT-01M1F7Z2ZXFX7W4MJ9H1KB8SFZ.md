---
schema: 1
id: TKT-01M1F7Z2ZXFX7W4MJ9H1KB8SFZ
title: Decide what tool discovery the stdio adapter should expose
type: spike
status: draft
status_reason: null
priority: low
labels:
  - mcp
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
updated_at: 2026-09-02T16:56:19Z
created_by:
  id: human:sothr
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

A deferred question in plan section 15, and part of Phase 4. The MCP adapter has to name its tools and say what each takes. git ticket schema now publishes the enums and codes, which is most of what such a description needs.

## Notes

**agent:terva/mieli** at 2026-09-02T16:56:19Z

Still parked, but the trigger in plan section 15 is now precise. It fires when a host wants to drive git-ticket and cannot embed the cli package. Terva is not that host, because 12.2 exports cli for it to embed. What remains undecided is whether every command becomes a tool or only the ones an agent should reach for.
