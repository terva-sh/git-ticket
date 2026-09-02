---
schema: 1
id: TKT-01K3ZYNRC014P7KT7R1SASWG2B
title: Ticket carrying a status outside the set
type: task
status: frobnicate
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

`frobnicate` is not one of the seven statuses in 6.1, but the YAML is
well-formed, so this is not a parse failure. The file parses, `check` reports
`invalid_status`, and the bytes survive a round trip untouched.

Preserving the value matters more than rejecting it. A tool that normalized an
unrecognized status to `draft` would destroy the evidence a person needs to fix
the mistake, and it would do so in a file that Git tracks.

`invalid_type` and `invalid_priority` follow the same rule on their own fields.
