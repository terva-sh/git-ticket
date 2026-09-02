---
schema: 1
id: TKT-01K40091005G7KY9JBW0WB8543
title: Ticket carrying a type outside the set
type: incident
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

`incident` is not one of the five types in 5.1. It is a plausible thing to want,
which is what makes it the realistic mistake: someone reached for a vocabulary
the format does not have.

The file parses and round-trips, and `check` reports `invalid_type`. Adding a
type is a `config.yml` question, and 4.1 is explicit that configuration sets
vocabulary but cannot add a status. Type is the same kind of decision.
