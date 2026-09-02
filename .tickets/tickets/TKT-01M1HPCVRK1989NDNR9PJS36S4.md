---
schema: 1
id: TKT-01M1HPCVRK1989NDNR9PJS36S4
title: Add a deadline field to tickets and let list and ready review by it
type: task
status: draft
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
references:
  - ref: review:backlog-md
    path: docs/review-backlog-md.md
claim: null
archive: null
created_at: 2026-09-02T18:34:12Z
updated_at: 2026-09-02T20:42:53Z
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

### What is settled: the date goes on the ticket

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

### The field is the small half

`list` sorts by ID and has no `--sort` at all. A deadline nobody can sort or filter by is a deadline read out by eye, which is what prose in a description already is, so shipping the field alone does not meet the trigger above.

Ordering `ready` is where it pays most: of the things I can start, what is closest to late.

### What check must not do

`check` must not report an overdue ticket. It validates store integrity, and being behind schedule is not a malformed store. Mixing a calendar verdict into the command CI gates on means CI goes red for a reason no commit caused, which is how `--strict` gets turned off.

A missing or unparseable value is a finding. A passed date is not.

### Decision 1: the name

`due_at` fits the convention. Every timestamp in 5.1 ends `_at`, and `expires_at` on the claim block is precedent for an `_at` that points at the future rather than recording the past.

`complete_by` reads better in prose and would be the only field of its shape in the file.

### Decision 2: a date or an instant

A deadline is naturally a date, but end of day needs a timezone.

Option A, store an RFC3339 instant like every other timestamp in the format, and have the CLI accept a bare `2026-10-14` by expanding it.

Option B, store date-only and define it in the plan as end of day UTC.

Backlog.md stores an instant at minute precision and rejects date-only input, which is the worse half of each.

### Decision 3: does ready sort by it without a flag

Sorting by default makes the field useful without anybody learning a flag, and quietly changes output for every existing caller of `ready`.

A flag leaves the default alone and gets used by nobody.

### Cost

Plan 5.3 renders an absent scalar as `null` rather than omitting it, so this means editing every fixture that carries frontmatter. `status_reason` cost exactly that, and the AGENTS.md note records it as all 32 of them.

## Acceptance criteria

- [ ] All three decisions in the description are settled and recorded in docs/plan.md before any code lands.
- [ ] The plan states the field holds an external constraint and is not a second priority.
- [ ] list can filter and sort on the field.
- [ ] ready orders by it, with the default settled by decision 3.
- [ ] check reports an unparseable value and never reports an overdue ticket.
- [ ] Every fixture carrying frontmatter gains the field, per 5.3.
