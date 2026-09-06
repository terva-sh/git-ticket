---
schema: 1
id: TKT-01M1VKV53GZK2A7PMD011H2KKM
title: Raise the format to schema 2 and build migrate, per plan 12.5
type: task
status: done
status_reason: null
priority: high
due_on: null
labels:
  - format
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-06T15:01:59Z
updated_at: 2026-09-06T17:46:01Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

The first of the two tickets that ship v0.12.0, and it lands first so
that `migrate` exists before anything asks a store to adopt a series.
It builds the machinery that makes a second schema real, with no series
in it yet.

### What schema 2 adds here

`ticket.SchemaVersion` goes to 2, and `origin` becomes the first field
that exists at schema 2 and not at schema 1. The field is inert in this
ticket: nothing sets it and `create --from` belongs to the series
ticket. It is here because a schema needs a field to differ by, and
because the renderer rule below cannot be tested without one.

### The renderer rule, which is the load-bearing part

Per plan 5.3 and 5.6, a field introduced at schema 2 renders only at
schema 2. `renderFrontmatter` emits the field set for the ticket's own
schema rather than every known field. Without it a v0.12.0 binary
writing into a schema-1 store turns every ticket it touches into an
`unknown_field` error for a colleague who has not upgraded, which is
exactly the drift 12.5 exists to prevent.

Adding a frontmatter field means editing every fixture that carries
one, per 5.3, which cost `status_reason` the same. Here the gate should
mean the schema-1 fixtures do not change at all, and that is worth
checking rather than assuming.

### migrate

`git ticket migrate [--to N] [--dry-run]` and `Store.Migrate`, per 12.5.
Both exist because a person needs the command and a host embedding the
library needs the method, and neither can drive the other. It converts
the whole store in one pass under the store lock, is idempotent, skips
a ticket already at the target, and writes `config.yml` before any
ticket so an interrupted run leaves a store an old reader refuses
outright rather than one it reads with tickets missing. There is no
downgrade.

### The warning

`check` reports `migration_incomplete` when a ticket declares a lower
schema than `config.yml` does. It is a warning, because such a store is
correct for a reader that understands both levels, and `check --fix`
does not repair it: the repair is `migrate`, and a store moves only
through a migration a person runs while `check --fix` is what CI runs.

Section 11's tables carry none of the three schema-2 codes yet, because
`TestCorpusCoversEveryPlanCode` requires a fixture for every code in
them. This ticket adds the `migration_incomplete` row and its fixture
together. `unknown_series` and `origin_missing` belong to the series
ticket.

## Acceptance criteria

- [x] ticket.SchemaVersion is 2, and a schema-1 store still parses, renders, and round-trips with no fixture byte changed
- [x] renderFrontmatter emits origin at schema 2 and omits the key entirely at schema 1
- [x] The 5.3 round trip holds on a schema-1 fixture and on a schema-2 fixture
- [x] git ticket migrate and Store.Migrate convert a store in one pass under the lock, and a second run reports nothing to do
- [x] migrate writes config.yml before any ticket, proven by an interrupted run leaving a store an old reader refuses outright
- [x] check reports migration_incomplete on a store whose files are behind its config, and check --fix declines to repair it
- [x] Section 11 carries the migration_incomplete row with a fixture, so TestCorpusCoversEveryPlanCode passes
- [x] create in a schema-1 store still stamps schema 1, per the rule 12.5 already ships

## Definition of done

- [x] just ci is green
- [x] docs/plan.md 12.5 is amended in the same commit if the build deviates from it

## Notes

**agent:terva/mieli** at 2026-09-06T17:46:01Z

Built and verified. Schema 2 exists, `migrate` ships, and `check`
reports `migration_incomplete`. Three decisions were taken with the
user before the CLI half, and two things surfaced that the ticket did
not predict.

### The three decisions, settled 2026-09-06

1. `migrate --json` gets its own envelope kind, `migrate-result`,
   specified as plan 10.8, rather than reusing `mutation-result` with a
   null ticket stub.
2. `migrate --dry-run` exits 0 with a migration pending. `check
   --strict` already fails CI on `migration_incomplete`, and 12.5 makes
   a store move only through a migration a person runs, so a second
   gate would be a duplicate that can disagree with the first.
3. `init` keeps writing the current level, so a store created by
   v0.12.0 is refused outright by a v0.11 binary. That is the designed
   loud refusal rather than a regression.

### The design point the ticket did not state

Below schema 2, `origin` is not a struct field at all, it is an unknown
field. The smaller-looking design is wrong: parsing `origin` at every
level and letting the renderer decline to emit it below 2 silently
deletes the line on the next write. A field a level does not define has
to be unknown to it, or the round trip is not a round trip.
`testdata/parse/roundtrip/origin-below-schema.md` pins that, beside
`origin.md` which is the same key at the level that defines it.

That has a consequence the ticket also did not predict. A schema-1
ticket carrying `origin` as an unknown field would, on migration,
render the key twice: once as the known null and once as the preserved
copy. `promoteUnknown` moves it into the field instead, and a value the
field cannot hold stops the pass rather than writing something nobody
meant. That refusal is also how the config-before-tickets test
interrupts a run deterministically, without a chmod trick that a root
CI container would skip.

### A contract bug this surfaced

The binary emitted ten envelope kinds and published nine. `self-update`
shipped in v0.8.0 absent from the plan's list, from the `kinds` array
10.4 publishes, and from `envelopeKinds`, so a consumer validating an
envelope against the published list would have rejected a legitimate
answer for four releases.

It is the second occurrence. `TestEveryEmittedKindIsPublished` warns in
its own doc comment that `version` shipped the same way in v0.4.0,
because that guard is only as complete as its hand-written table, so
both lists agreed with each other while disagreeing with the plan.
`TestEnvelopeKindsMatchTheSource` now reads the kind literals out of
the source, which is the technique `TestGitCommandsAreReadOnly` uses on
`exec.Command`: the property is about what the code contains, not about
what one test happens to run.

Repaired here because this change edits all three lists anyway, and
adding a kind beside a known gap is worse than closing it.

### One corpus adjustment

`migration_incomplete` is store-scoped the way `label_unknown` is.
`TestCheckParseFixtures` passed `DefaultConfig()` under a comment
saying "no config", and that default now declares a level, so every
schema-1 parse fixture reported a migration nobody started. The harness
zeroes the declared level, which is what the comment always meant.

### Proven by running it

Every criterion has a run behind it, and the command was exercised on a
real scratch store rather than only through tests: dry run exit 0
writing nothing, the real run moving config.yml and both tickets with
`updated_at` byte-identical afterwards, an idempotent second run, a
downgrade refused at exit 1, and `check --strict` clean.

## Summary

Shipped. Schema 2 exists and nothing moves to it on its own.

`ticket.SchemaVersion` is 2. It adds `origin` to frontmatter and
removes nothing, so reading a schema-1 file is unchanged, which is what
makes this additive under 12.4. `hasOrigin(schema)` is the one function
parse and render both call, so the two cannot come to disagree and drop
the field on a round trip. Below that level `origin` is an unknown
field rather than a struct field, which is what keeps a hand-edited
schema-1 file from losing it.

`Store.Migrate` and `git ticket migrate [--to N] [--dry-run]` convert a
store in one pass under the store lock. `config.yml` is written before
any ticket, so an interrupted run leaves a store an old reader refuses
outright rather than one it reads with tickets missing. It is
idempotent, there is no downgrade, and it leaves `updated_at` and
`updated_by` alone, because a migration is not a mutation by an actor.

`check` reports `migration_incomplete` when a ticket is behind its
store's declaration. It is a warning, and `check --fix` declines it,
because the repair is `migrate` and a store moves only through a
migration a person runs.

Three surfaces were settled with the user first: the `migrate-result`
kind of plan 10.8, exit 0 for a pending `--dry-run`, and `init`
continuing to write the current level.

Two things arrived that the ticket did not predict. Migrating a
schema-1 ticket that already carried `origin` would have written the
key twice, which `promoteUnknown` prevents. And the binary turned out
to publish nine envelope kinds while emitting ten, with `self-update`
undeclared since v0.8.0; both lists now carry it and a source-scanning
test closes the hole that let it and `version` through.

TKT-01M1VKVH (Build ID series and origin provenance, per plan 5.6) is
next and depends on this. Both ship as v0.12.0.
