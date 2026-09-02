---
schema: 1
id: TKT-01M1HPCVRK1989NDNR9PJS36S4
title: Add a deadline field to tickets and let list and ready review by it
type: task
status: ready
status_reason: null
priority: low
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
updated_at: 2026-09-02T22:15:45Z
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

### Cost

Plan 5.3 renders an absent scalar as `null` rather than omitting it, so this means editing every fixture that carries frontmatter. That is 44 files today. `status_reason` cost exactly that, and the AGENTS.md note records the migration pattern that did it: a throwaway test gated on an environment variable that re-renders each fixture through the package renderer.

## Acceptance criteria

- [x] All three decisions in the description are settled and recorded in docs/plan.md before any code lands.
- [ ] The plan states the field holds an external constraint and is not a second priority.
- [ ] list can filter and sort on the field.
- [ ] check reports an unparseable value and never reports an overdue ticket.
- [ ] Every fixture carrying frontmatter gains the field, per 5.3.
- [ ] The frontmatter field is named due_on and holds a YYYY-MM-DD date rather than an RFC3339 instant, per plan 15.
- [ ] ready orders by due_on ascending, undated last, ID as the tiebreak, with no flag to ask for it.
- [ ] A store where no ticket carries a due_on gets the same ready order it gets today, and a test holds that.
- [ ] The CLI refuses an instant passed where a date belongs rather than truncating it.

## Notes

**agent:terva/mieli** at 2026-09-02T22:15:45Z

Settled all three open decisions. The field is due_on holding a YYYY-MM-DD date, and ready orders by it ascending with undated last and no flag. The name and the precision were one question rather than two: _at means an RFC3339 instant everywhere else in 5.1, so a due_at holding a date would have made expires_at ambiguous. The ordering tradeoff dissolved against the ID tiebreak, because the date key is a no-op in a store where nobody has set a date. Recorded at the end of plan section 15. TKT-01M1J2YR9D5242F6H7TPEV4M8K carries the priority-ranking question this raised.
