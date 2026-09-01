---
schema: 1
id: TKT-01M1FCJ8XDS7ECVWQ2W7VS8X0G
title: Decide whether the terva ticket integration ships off by default
type: spike
status: ready
status_reason: null
priority: normal
labels:
  - integration
assignees: []
milestone: null
parent: TKT-01M1FCHH3QN04AYHZAP1M8DNQK
dependencies: []
references: []
claim: null
archive: null
created_at: 2026-09-01T21:03:55Z
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

terva AGENTS.md says a feature that changes authority or safety ships off, and is enabled from user-layer configuration only, never from project configuration, which a cloned repository controls.

Whether that rule bites here is genuinely unclear. Adding workspace-mutation tools is ordinary tool surface rather than a change to the authority model: write and edit are already in that class. Against that, ticket tools let an agent mutate files in the user repository on the strength of a store that the repository itself provides, and store-gated registration means a cloned repository decides whether the tools appear at all. That is discovery rather than authority, but the two sit close together here.

If a config key is wanted, decide whether it gates the write tools alone or the whole family. Read tools are local-read and auto-allowable, so gating them buys little.

Decide before slice 3. Slice 2 is unaffected either way.
