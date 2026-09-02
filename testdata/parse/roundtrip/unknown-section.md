---
schema: 1
id: TKT-01K3ZYKXS09X59847ZSQW9BH5D
title: Measure cold start before optimizing it
type: spike
status: ready
status_reason: null
priority: normal
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-08-31T12:00:00Z
updated_at: 2026-08-31T12:00:00Z
created_by:
  id: human:sothr
  name: Drew Short
updated_by:
  id: human:sothr
  name: Drew Short
extensions: {}
---

## Description

Two sections here are not in the known set of 5.2. They survive the round trip
and render after every known section, keeping their order relative to each
other.

## Acceptance criteria

- [ ] A cold start is measured on a machine with an empty cache

## Risks

Measuring on a warm cache would report a number that no user ever sees, which
is worse than not measuring at all.

## Open questions

Does the daemon count as cold when the binary is in the page cache but the
config has not been read?
