---
schema: 1
id: TKT-01M1PCY3NMQH8QYF3FMMKTC6ZG
title: schema does not publish the label and milestone allowlists
type: task
status: draft
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
created_at: 2026-09-04T14:25:04Z
updated_at: 2026-09-04T14:25:04Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

git ticket schema --json publishes thirteen keys: blocksOn, errorCodes, findingCodes, kind, kinds, openStatuses, priorities, schemaVersion, statuses, ticketSchema, transitions, types, unreadyReasons. Neither labels nor milestones is among them.

Both are enforced. A label outside the allowlist is label_unknown and a milestone outside it is milestone_unknown, so a write fails on a value the tool will not name. The only way to learn the legal set is to open .tickets/config.yml and read it.

Found while filing a ticket in this repository. A docs label seemed obvious, was rejected, and the allowlist had to be read out of the store file to discover that the sanctioned set is ci, claims, format, integration, mcp, policy, question and release.

The asymmetry is the argument. schema exists so a consumer never hard-codes a value this binary enforces, and 10.4 makes that its job. It already publishes the status enum for that reason, and unreadyReasons was added so nobody would hard-code eight values. Labels and milestones are enforced the same way and published nowhere.

One thing to settle. The other schema keys describe the binary and are identical in every store, while these two are per-store configuration. Putting them in the same envelope mixes those two things, so this ticket should decide whether they belong in the schema kind, in a separate config kind, or on a store envelope.

## Acceptance criteria

- [ ] The legal labels and milestones are discoverable from a command rather than from config.yml
- [ ] docs/plan.md section 10.4 records where they sit and why
- [ ] A store with no allowlist is distinguishable from one with an empty allowlist
