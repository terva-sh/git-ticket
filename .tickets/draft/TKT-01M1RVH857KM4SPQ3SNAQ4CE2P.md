---
schema: 1
id: TKT-01M1RVH857KM4SPQ3SNAQ4CE2P
title: Filter the TUI list by type and priority tokens
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
created_at: 2026-09-05T13:18:40Z
updated_at: 2026-09-05T13:18:40Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Filed 2026-09-05 from a user ask: filter the TUI list by type and
priority, so a person can see just the epics, or just what is urgent.

Half of the ask already exists, which is worth stating so nobody
builds it twice. The filter line in tui/view/filter.go parses a token
grammar today: `status:S`, `label:L`, and `assignee:A` route to their
fields, a bare word matches titles without regard to case, values of
one field are alternatives, and fields AND together, deliberately
mirroring ticket.Filter so `git ticket list` and the filter line teach
one rule. `status:ready` works now.

What is missing is `type:T` and `priority:P`, and the fix is two more
cases in parseFilter plus the matching clauses where the filter is
applied. The type column just landed in the list view
(TKT-01M1RPSG), so type is now visible but not filterable, which is
the gap a person hits first.

The ask spelled the tokens with a leading colon, `:type:epic`. The
grammar this extends has no leading colon, and `type:epic` beside
`status:ready` is one rule while `:type:epic` beside `status:ready` is
two, so the proposal is the existing spelling. A token with a leading
colon costs nothing to also accept, since today it falls through to a
title word that matches nothing, but documenting one spelling is the
point.

The tokens belong in the help page with the rest of the filter
grammar, wherever `?` documents the filter line today. If it does not
yet document the grammar, this is the moment it starts, because three
findable tokens beat five secret ones.

### Trigger

None. Startable feature work, parked as a draft until promoted. Small
enough to ride along with any other TUI slice.

## Acceptance criteria

- [ ] type:T and priority:P filter the list with the same alternatives-within, AND-across semantics as status:
- [ ] the ? help page documents the filter grammar, all five tokens and bare words
- [ ] tests cover both new tokens
