---
schema: 1
id: TKT-01K3ZZQ7X0T4N2VJ8B6HM5WD9E
title: Move the fleet onto short-lived credentials
type: epic
status: ready
status_reason: null
priority: high
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
updated_at: 2026-09-12T09:00:00Z
created_by:
  id: human:sothr
  name: Drew Short
updated_by:
  id: human:sothr
  name: Drew Short
extensions: {}
---

## Description

This epic exists and `epics.md` beside it still says the store has none, which
is the ordinary way the index goes stale: somebody filed an epic and nobody ran
`git ticket check --fix` afterwards.

The index is derived, so nothing here is wrong. `check` compares the file
against the tickets rather than trusting it, reports `epics_index_stale`, and
`check --fix` rewrites it.

It is a warning rather than an error because a derived file falling behind is
not a malformed store. `--strict` promotes it, and CI runs `--strict`, so it is
enforced exactly as hard as an error everywhere enforcement happens.

`blocks_on` is `none` deliberately. An epic gating on children it does not have
would raise `blocks_on_no_children` too, and this fixture isolates one code.
