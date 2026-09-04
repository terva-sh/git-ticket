---
schema: 1
id: TKT-01M1PRBT2GBWC82AW05TW6X0JH
title: Give ticket titles a length limit and name them beside every ID
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
claim: null
archive: null
created_at: 2026-09-04T17:44:47Z
updated_at: 2026-09-04T17:58:14Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

A ticket ID is a ULID and carries no meaning a person can hold in their head.
An operator reading `TKT-01M1PQ7T` in prose, in a check finding, or in an error
has to go and look it up before they know what is being talked about.

`title` already exists and `list` and `ready` already print it. The gap is
everywhere else: `check` findings carry only code, file, ticket and field, an
error naming a ticket names its ID alone, and agent prose quotes bare IDs.

### The rule

Titles get a length limit so they stay readable inline. A warning above 72
characters and a refusal above 120. The corpus is the reason for two thresholds
rather than one: across 62 tickets the median title is 57 characters and 23 of
them are over 60, so a single cap tight enough to be useful would make a third
of the store non-conforming, and `check --strict` gates CI here.

Severity belongs to the code in this codebase rather than to the call site, so
that is two check codes and not one whose severity varies.

The 120 cap is also enforced on the write path, the way an invalid priority is
refused by `create` rather than only reported by `check`.

No `short_title` field. 120 characters is generous enough that a second field
carrying the same meaning would be redundant, and the title's whole job is
telling an operator what the ticket is about.

### Where titles appear

`check` findings gain a title in the human rendering. Not in the JSON: findings
pin exactly four keys and every fixture sidecar records them, so adding one
there rewrites the corpus for something no machine asked for. `Message` is
already `json:"-"` and is the precedent. Adding it to the JSON later is additive
under 12.4; removing it would be a break.

An error that names a ticket names its title too.

### Agent prose

An agent naming a ticket in prose includes the title on first mention, as
`TKT-01M1PQ7T (Build git ticket remove, per plan 9.1)`. Later mentions in the
same piece of writing may use the bare ID. This is what makes a summary readable
without a second window open.

## Acceptance criteria

- [x] check warns above 72 characters and errors above 120
- [x] create and update refuse a title over 120
- [x] check findings name the ticket title in the human rendering
- [x] an error naming a ticket names its title
- [x] schema publishes both limits
- [x] the agent instructions require the title on first mention of a ticket

## Summary

Shipped. A title is now bounded and appears wherever an ID does.

`title` already existed and `list` already printed it, so nothing was added to
the format. What was missing was the bound and the other three surfaces.

Warn over 72, refuse over 120, measured in characters rather than bytes. Two
thresholds because one useful cap would have been retroactive: across the 62
tickets here the median title is 57 and 23 were over 60, so a cap at 60 would
have made a third of the store non-conforming while `check --strict` gates CI.

The 72 threshold still cost four renames in this repository, which is the
honest price of the rule and was paid rather than avoided by loosening it. The
plan names three of those four IDs but quotes none of their titles, so nothing
desynced.

Severity belongs to the code here rather than to the call site, enforced by
`TestEveryFindingMatchesItsPublishedSeverity`, so this is two codes:
`title_long` warns and `title_too_long` errors. They are exclusive, so one
title raises one finding rather than an error with a weaker restatement beside
it. Both have a corpus fixture.

`title_too_long` is reachable only from a hand edit or a merge, since writes
refuse past 120. That is the same shape as `invalid_priority`, and the fixture
says so.

Titles reach `check` findings and error messages through the human rendering
only. `Finding` pins exactly four JSON keys and every sidecar records them, so a
fifth would have rewritten the corpus for something no machine asked for.
`Message` was already `json:"-"` and is the precedent. Adding it to the JSON
later is additive under 12.4; removing it would be a break.

The finding column abbreviates at 48 characters. The first cut printed the whole
title and produced a 200-character row, which defeats the reason it is there.

Two regressions were introduced and caught here rather than later. Prefixing
`Error()` with the ID made `ticket_not_found` print it twice in one sentence,
since the message already carried it; `ResolveRef` and the `parse_error` path
were both reworded. `TestTicketNotFoundNamesTheIDOnce` holds that. And
`stale_revision` returns before the parse, so it named a ticket with no title
until it started reading the title from bytes already in hand. Losing a race is
the failure an agent hits most, so it was worth the extra parse.

`mutation.go` was left alone in two places that look like the same bug: the
messages for a missing parent and a missing dependency name a different ticket
than the subject, so there is no duplication to remove.

The prose convention is in `cli/instructions.md`, which carries it to every
project, and in AGENTS.md for this one. A ULID is unreadable on purpose, so a
summary naming three bare IDs asks its reader to look up three tickets before it
says anything.
