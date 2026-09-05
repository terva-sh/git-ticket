---
schema: 1
id: TKT-01M1RVH857KM4SPQ3SNAQ4CE2P
title: Filter the TUI list by type and priority tokens
type: task
status: done
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
claim:
  actor: agent:terva/mieli
  branch: feat/tui-filter-tokens
  worktree: /home/sothr/workspace/git.local.sothr.com/terva-sh/git-ticket
  commit: 4661b8ce0ee908a8da6bcf4d717ca4d91ae31fc6
  claimed_at: 2026-09-05T13:24:05Z
  expires_at: null
archive: null
created_at: 2026-09-05T13:18:40Z
updated_at: 2026-09-05T13:25:01Z
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

- [x] type:T and priority:P filter the list with the same alternatives-within, AND-across semantics as status:
- [x] the ? help page documents the filter grammar, all five tokens and bare words
- [x] tests cover both new tokens

## Notes

**agent:terva/mieli** at 2026-09-05T13:25:00Z

Built and all three criteria are ticked.

The change is as small as the draft predicted: two cases in
parseFilter, two clauses in match, and the help page. The struct field
for the type token is named kind, because Go took the word, with a
comment saying so. Both tokens use oneOf, the scalar match status
uses, not the set-intersection match of labels and assignees, because
type and priority are one value per ticket.

The help page's Filter section already existed, so the second
criterion was one example line, now showing all five tokens across two
lines. TestFilterByTypeAndPriority covers each token alone and the
combination, with two priorities as alternatives and the type ANDed
across, the same grammar sentence the other filter tests assert.

The leading-colon spelling from the original ask was not added. The
draft's reasoning held: :type:epic falls through to a title word today
and matches nothing, which is honest, and one documented grammar beats
two accepted ones.

## Summary

Shipped. The TUI filter line takes type:T and priority:P beside the
existing status:, label:, and assignee: tokens, same spelling, same
semantics: alternatives within a field, conjunction across fields. The
? help page shows all five in its example. TestFilterByTypeAndPriority
holds each token and the combination.
