---
schema: 1
id: TKT-01K400J5Z097ND5C86DFD1W4KZ
title: In progress with nobody holding it
type: task
status: in-progress
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
created_at: 2026-08-31T12:00:00Z
updated_at: 2026-09-05T14:00:00Z
created_by:
  id: human:sothr
  name: Drew Short
updated_by:
  id: human:sothr
  name: Drew Short
extensions: {}
---

## Description

The status says someone is working on this and `claim` says nobody is.

This is the state that makes a multi-agent board lie. A second agent reading
status alone concludes the work is taken, while `ready` correctly reports the
ticket as available, and the two answers disagree. Either could be right, which
is why the tool says so instead of picking.

It is a warning and not an error because the file is well formed and the
situation is recoverable: a human moving a ticket by hand produces it routinely,
and the fix is to claim the ticket or move it back to `ready`.

`claim: null` and not an expired claim, which is the separate `claim_expired`
case in `parse/roundtrip/claim-expired.md`.
