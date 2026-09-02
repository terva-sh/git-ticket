---
schema: 1
id: TKT-01M1HPCVRK1989NDNR9PJS36S4
title: Add a deadline field to tickets and let list and ready review by it
type: task
status: done
status_reason: null
priority: low
due_on: null
labels:
  - question
  - format
assignees: []
milestone: null
parent: null
dependencies:
  - TKT-01M1HPB7ZT0FREYR04ZSHTMW3F
blocks_on: none
references:
  - ref: review:backlog-md
    path: docs/review-backlog-md.md
claim: null
archive: null
created_at: 2026-09-02T18:34:12Z
updated_at: 2026-09-02T23:06:26Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Raised by reading Backlog.md, section B5 of docs/review-backlog-md.md, and rescoped in design conversation once the milestone allowlist landed.

The trigger is a store where somebody has to answer what is late and is reduced to reading it out of prose in a description.

All three open decisions are settled and recorded at the end of docs/plan.md section 15. The field is `due_on`, it holds a date, and `ready` orders by it with no flag. What follows is why, then what is left to build.

### The date goes on the ticket

The original filing leaned the other way. Put the date on a milestone, buy the field once, avoid touching every fixture. That was written before TKT-01M1HPB7ZT0FREYR04ZSHTMW3F landed, and that ticket settled milestones as an advisory allowlist of bare strings in `config.yml`, with no milestone object and no milestone file. A dated milestone would mean turning the list into a mapping.

It also does not answer the question. Not every deadline is a release. A key that has to be rotated before it expires has a hard date and no milestone, and if a one-off deadline has to invent a milestone to hold its date, then the milestone vocabulary fills with entries that are not releases. That is the accumulation the allowlist warning in 5.1 exists to catch.

A milestone that wants a date becomes an epic ticket and carries the ticket field. That keeps one date field in the format and leaves `config.yml` a plain vocabulary. TKT-01M1HXHJWR806ETJQCE49AEZB3 holds the epic half.

### A deadline is not a priority

The field holds a constraint that comes from outside the ticket, not a preference. "It has to be done by the 14th" is a date. "I want this sooner" is `priority`, and `priority: urgent` with no date stays a complete statement.

The line matters because a field that accepts eagerness fills with dates nobody chose, and then every row reads as late and the field stops meaning anything.

### Why the cost is worth paying

A date is self-evaluating. `priority: high` means whatever it meant on the day somebody typed it and goes stale silently, because nothing about the passage of time changes the field. A date changes meaning on its own, and nobody has to revisit a ticket for it to become late.

That is the whole argument, and nothing else in the format has that property.

### Why not extensions

A consumer can write `extensions: {ns: {due: ...}}` today with no format change at all. It fails because 5.1 says the core never interprets `extensions`, so `check` cannot report a malformed date, `list` cannot filter on it, and `ready` cannot order by it.

That is the test for whether a field belongs in core: if the tool should act on the value, it is a core field, and if only a display layer reads it, it is an extension. Reviewing by deadline needs the tool to act.

### Settled: the name is due_on, holding a date

Neither offered name survives, because the name and the precision are one question rather than two.

`due_at` was the option that fit the convention, and the convention it fits is that `_at` holds an RFC3339 instant, which every timestamp in 5.1 does. Once the value is a date, `due_at` teaches a reader that `_at` sometimes means a day, and `expires_at` on the claim block is what goes ambiguous. `_on` costs one new suffix and carries a distinction a reader needs anyway. `complete_by` reads better in a sentence and says nothing about which of the two shapes it holds.

The value is `2026-10-14`, meaning the end of that day in UTC, stored as written.

Expanding a date to an instant throws away what somebody meant. A deadline is a claim about a calendar day and no deadline was ever 23:59:59Z. Expanding also has to pick a zone at write time: the writer's local zone stores two different instants for the same typed date depending on who typed it, and UTC is the end-of-day rule with fake precision spelled into the file. Either way the file stops recording that somebody said the 14th, and no reader can tell an expanded date from an instant somebody chose.

Backlog.md stores a minute-precision instant and rejects date-only input, which is both halves of this wrong at once.

### Settled: ready orders by it with no flag

`ready` orders by `due_on` ascending, undated last, ID as the tiebreak.

The tradeoff this looked like does not survive the tiebreak. A store where nothing carries a deadline sorts exactly as it sorts today, because the date key does nothing until somebody sets a date, and setting one is a deliberate act by the person who wants the order to change. So the default reorders no existing caller, and a flag would have been one more thing to learn before getting behaviour the reader already asked for by typing a date.

Undated last is "closest to late first" with never at the far end.

The order belongs to `ready` and not to `list`. `ready` recommends what to start next, so its ranking is part of the answer. `list` reports what exists, and stays chronological.

This settles one sort key and does not make `ready` rank by priority. TKT-01M1J2YR9D5242F6H7TPEV4M8K holds that question. The deadline key raised it rather than created it: `ready` sorts by ID today, so an urgent ticket already places below a low one filed before it.

### The field is the small half

`list` sorts by ID and has no `--sort` at all. A deadline nobody can sort or filter by is a deadline read out by eye, which is what prose in a description already is, so shipping the field alone does not meet the trigger above.

### What check must not do

`check` must not report an overdue ticket. It validates store integrity, and being behind schedule is not a malformed store. Mixing a calendar verdict into the command CI gates on means CI goes red for a reason no commit caused, which is how `--strict` gets turned off.

A missing or unparseable value is a finding. A passed date is not.

### Cost, as paid

Plan 5.3 renders an absent scalar as `null` rather than omitting it, so this meant editing every fixture that carries frontmatter. The estimate was 44, counted by grepping for `created_at:`. The bill was 42, migrated by the throwaway re-render test AGENTS.md describes.

The two that did not move are `conflict-markers.md` and `schema-2.md`, which carry frontmatter and exist to fail parsing. A field they never reach is a field they must not carry, so the grep counted two files the renderer correctly refused to touch.

### What shipped beyond the three decisions

Building it settled a fourth thing, which this ticket had carried as one line of acceptance criteria rather than as a question. `list` filters with `--due-by`, an inclusive bound, and sorts only when asked with `--sort due_on`, while `ready` sorts always. Plan 8 holds the asymmetry: `ready` ranks, and a list reports.

The value renders quoted, as `due_on: "2026-10-14"`. That is 5.3's existing rule and not a new one, because the renderer quotes any scalar a YAML reader would resolve to something other than a string, and a bare date is a YAML timestamp. It also makes the field portable, since a YAML 1.1 reader hands a bare date back as a date object. A hand-written bare date still parses, and the next write canonicalizes it.

## Acceptance criteria

- [x] All three decisions in the description are settled and recorded in docs/plan.md before any code lands.
- [x] The plan states the field holds an external constraint and is not a second priority.
- [x] list can filter and sort on the field.
- [x] check reports an unparseable value and never reports an overdue ticket.
- [x] Every fixture carrying frontmatter gains the field, per 5.3.
- [x] The frontmatter field is named due_on and holds a YYYY-MM-DD date rather than an RFC3339 instant, per plan 15.
- [x] ready orders by due_on ascending, undated last, ID as the tiebreak, with no flag to ask for it.
- [x] A store where no ticket carries a due_on gets the same ready order it gets today, and a test holds that.
- [x] The CLI refuses an instant passed where a date belongs rather than truncating it.

## Notes

**agent:terva/mieli** at 2026-09-02T22:15:45Z

Settled all three open decisions. The field is due_on holding a YYYY-MM-DD date, and ready orders by it ascending with undated last and no flag. The name and the precision were one question rather than two: _at means an RFC3339 instant everywhere else in 5.1, so a due_at holding a date would have made expires_at ambiguous. The ordering tradeoff dissolved against the ID tiebreak, because the date key is a no-op in a store where nobody has set a date. Recorded at the end of plan section 15. TKT-01M1J2YR9D5242F6H7TPEV4M8K carries the priority-ranking question this raised.

**agent:terva/mieli** at 2026-09-02T23:06:26Z

Shipped. due_on is a *string on the ticket, decoded as a string so a malformed value round-trips and check reports invalid_due_on rather than the whole file failing to parse. ValidDueOn demands the exact YYYY-MM-DD shape, so an instant and 2026-1-4 are both refused at create, at update, and in the library mutation. SortByDueOn is exported and shared, so ready and list cannot come to disagree about the order. Two surprises worth the record. The fixture bill was 42 and not 44: conflict-markers.md and schema-2.md carry frontmatter but exist to fail parsing, so the renderer correctly never reached them. And the value renders quoted, because 5.3 already quotes any scalar a YAML reader resolves to a non-string and a bare date is a YAML timestamp, which also keeps the field a string for a YAML 1.1 reader.
