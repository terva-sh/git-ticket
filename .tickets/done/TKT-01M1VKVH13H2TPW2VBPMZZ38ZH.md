---
schema: 2
id: TKT-01M1VKVH13H2TPW2VBPMZZ38ZH
title: Build ID series and origin provenance, per plan 5.6
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
origin: null
dependencies:
  - TKT-01M1VKV53GZK2A7PMD011H2KKM
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-06T15:02:11Z
updated_at: 2026-09-06T18:57:59Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Plan 5.6 is the design and this builds it: ID series, the enforced
`series` config key, series-aware resolution and abbreviation, `origin`
provenance, and the `series` command. It is the second of the two
tickets that ship v0.12.0 and it sits on the schema-2 machinery.

### Check the site count against the tree first

`IDPrefix` is one constant, and beyond `ValidID`, `NormalizeRef`, and
`ResolveRef` in ticket/id.go two things read it: the filename fallback
in `file.id()` in ticket/store.go, which decides whether an unparseable
file is a ticket, and `ShortestUnique` in ticket/id.go. That count was
four until PR #135 collapsed two copies of the abbreviation rule into
one, so verify it rather than trusting this paragraph.

### Resolution is where the measured bugs are

These were demonstrated against a schema-1 binary on 2026-09-05 and are
what the build has to fix, not hypotheses:

`NormalizeRef` strips `TKT-` and nothing else, so a prefixed ID never
normalizes to a bare ULID, drops out of fragment matching entirely, and
a bare fragment of it returns a different ticket at exit 0 where two
ordinary tickets would answer `ambiguous_id` at exit 1.

`ShortestUnique` normalizes, strips nothing, and pastes `IDPrefix` back
onto a string that may already carry a prefix, so a listing printed
`TKT-IDEA-01M`, an ID that exists nowhere and resolves to nothing.

Per 5.6 a bare ULID fragment must resolve across every series, a
prefixed fragment must match both halves, and the four-character floor
applies to the ULID half alone.

### Abbreviation

`ShortestUnique` shortens within a series and keeps its signature,
because an ID carries its own series and the function can group by
prefix from its argument alone. The store-wide mode arrives as a second
exported function rather than a changed one, which is what keeps this
additive under 12.4. `--ids series|store|full` is the flag, and it
belongs to every command that prints more than one ticket.

### Enforcement and provenance

The `series` config key is enforced, unlike `labels` and `milestones`:
`create` refuses an undeclared prefix and `check` errors on a ticket
carrying one, both as `unknown_series`. `config` publishes the
effective list with `enforced` always true, which is the one allowlist
whose `enforced` is not derived from its length.

`origin` gains its behaviour here. `create --from ID` seeds title,
description, and acceptance criteria and records the source, `--from`
and `--template` are refused together, `list --origin` reads the
reverse, and `check` reports `origin_missing`. There is no
`origin_cycle`, per 5.6.

### The section 11 rows

This ticket adds the `unknown_series` and `origin_missing` rows to
section 11 together with the fixtures that cover them, because
`TestCorpusCoversEveryPlanCode` fails on a row without one.

## Acceptance criteria

- [x] config.yml carries an enforced series key defaulting to [TKT], and create refuses an undeclared prefix with unknown_series
- [x] A bare ULID fragment resolves across every series, and a prefixed fragment matches both halves or nothing
- [x] show FRAGMENT never returns a ticket from another series, which is the exit-0 wrong answer measured on 2026-09-05
- [x] ShortestUnique abbreviates within a series, keeps its signature, and never prints TKT-IDEA-01M
- [x] --ids series|store|full works on every command that prints more than one ticket, with series the default
- [x] git ticket series lists, series add declares, and series remove refuses while tickets carry the prefix
- [x] create --from seeds title, description, and acceptance criteria, records origin, and is refused alongside --template
- [x] list --origin reads the reverse direction, and check reports origin_missing
- [x] Section 11 carries the unknown_series and origin_missing rows with fixtures, so TestCorpusCoversEveryPlanCode passes
- [x] A schema-1 store refuses create --from and series add, naming migrate

## Definition of done

- [x] just ci is green
- [x] A real store adopts a second series end to end: migrate, series add, create --series, list, show by prefix
- [x] docs/plan.md 5.6 is amended in the same commit if the build deviates from it

## Implementation plan

### Survey, verified against 51a8ca7 on 2026-09-06

The description's count holds. `IDPrefix` is read in exactly two files.

`ticket/id.go`: the constant, `NewID` (mints), `ValidID`, `NormalizeRef`,
`ResolveRef`, `ShortestUnique`.
`ticket/store.go`, in `file.id()`: the filename fallback that decides
whether an unparseable file is a ticket.

Callers of the abbreviation: `storeAbbreviations` in `cli/commands.go`
and `abbreviateIDs` in `tui/view/list.go`, both deferring to the one
`ticket.ShortestUnique` since PR #135.

### The shape

An ID becomes `SERIES-ULID`. A new `SplitID` returns the two halves, and
everything above stops reasoning about a fixed-width constant.

`Config.Series` holds the declared list. The effective list is `["TKT"]`
when it is empty, per 5.6, and it is enforced rather than advisory, so
`config` publishes it with `enforced` always true.

Resolution splits at the first hyphen, per 5.6:

- a bare ULID fragment matches across every series, since ULIDs cannot
  collide
- a prefixed fragment matches the series exactly and the ULID as a
  prefix, because the prefix is identity and not decoration
- the four-character floor applies to the ULID half alone
- a reference naming an undeclared series is `unknown_series`, not
  `ticket_not_found`, because the two send a reader to different places

`ShortestUnique` groups by series and abbreviates within each, keeping
its signature: an ID carries its own series, so the function can group
from its argument alone. The store-wide mode of `--ids store` arrives as
a second exported function rather than a changed one, which is what
keeps this additive under 12.4.

### Order of work

Config and the code first, then minting and validation, then
resolution, then abbreviation, then the surfaces that read them:
`--ids`, the `series` command, `create --series`, then `origin`. The two
findings and their fixtures land last, because
`TestCorpusCoversEveryPlanCode` fails on a section 11 row with no
fixture behind it.

### Traps already known

A schema-1 store refuses `create --from` and `series add`, naming
`migrate`, because neither can be honoured at a level whose field set
lacks `origin`.

`file.id()` must recognize any declared series, or a broken file in a
new series stops being seen as a ticket at all.

Adding `origin_missing` means `check` walks a new edge; `list --origin`
matches direct origin only, the way `--parent` does, and there is no
`origin_cycle` because nothing walks it transitively.

## Notes

**agent:terva/mieli** at 2026-09-06T18:57:55Z

Built and verified. Series and origin both ship.

Two interface contracts that 12.4 covers were settled with the user
before shipping, and the plan carries both. `--json series` answers with
a new `series` kind for all three forms, the bare read and both writes,
because they answer the same question and a caller that ran `add` to be
sure a series exists wants exactly what the read returns; `changed` is
what separates them. That is section 10.9. And `schema` publishes
`seriesLimits` with `minLength`, `maxLength` and `pattern`, because the
bounds are a fact about the binary rather than about one store.

The build deviated from 5.6 once and the plan moved in the same commit.
The filename fallback was to recognize any series the store declares; it
recognizes any well-formed ID instead. Checking the declaration there
would make an unparseable file in an undeclared series yield no ID at
all, which is the store failing to see a file sitting in it, and that is
the quiet wrong answer 5.6 argues against everywhere else.

Two things are worth knowing beyond what the criteria say.

`AddSeries` appends to the effective list and not the literal one. A
store that has written no `series` key has `[TKT]` in effect and an empty
literal, so appending to the literal would write `[IDEA]` alone and turn
every ticket already in the store into an `unknown_series` error. That is
the single sharpest edge in this feature.

`create --from` on a schema-1 store was missing its refusal, and the
suite could not have caught it because every test store is at the current
level. Against a real store built by the released v0.11.3 binary it
exited 0, reported `origin: null`, and wrote no `origin` line: `origin`
renders only at the level that introduced it, so the provenance the
caller asked to record was discarded while nothing said so. Now refused,
naming `migrate`, with a test on `schema1Store`.

The end-to-end run used a store made by v0.11.3 rather than by this
build, so the starting point was authentic. It refused `series add` at
schema 1 naming migrate, migrated, adopted IDEA while keeping TKT,
minted `IDEA-` IDs, resolved a bare fragment across series, refused
`TKT-` in front of an IDEA ULID with `ticket_not_found`, answered
`unknown_series` for an undeclared prefix, seeded and linked through
`--from`, refused `series remove` naming the count of 2, and finished
`check --strict` clean.

The compatibility claim of 5.6 was measured against that same released
binary rather than reasoned about. v0.11.3 meeting the migrated store
refuses `list`, `ready`, `show` and `check --strict`, each exit 1 with
`schema_unsupported`. The four quiet wrong answers 5.6 recorded on
2026-09-05 are now one loud refusal.

## Summary

Shipped. A store can now partition its IDs by prefix, and a ticket can
record which ticket it came out of.

`config.yml` gains `series`. It is the one allowlist 4.1's advisory rule
does not cover: an empty list means `[TKT]` rather than "no opinion", and
an undeclared prefix is refused rather than warned about, because a
series lives inside an immutable ID and the only repair is `remove` plus
a second `create`. `config` publishes it with `enforced` always true, per
10.6.

An ID is `SERIES-ULID`. `SplitID` separates the halves, `ValidSeries` is
5.6's grammar, and `ValidID` checks grammar alone: a well-formed ID whose
series the store does not declare is `unknown_series`, which is a
different repair from a malformed one.

The prefix is identity. `ResolveRef` compares the series exactly and the
ULID as a prefix, so `IDEA-01M1SH` never resolves to a TKT ticket, while
a bare fragment still resolves across every series so the commands people
already type keep working. `ShortestUnique` abbreviates within a series,
and `ShortestUniqueAcrossSeries` is what `--ids store` selects.

Six listings take `--ids series|store|full`. `git ticket series` lists,
adds, and removes under the store lock, answering with the new `series`
kind of 10.9. `create --series` files into a declared prefix.

`origin` records provenance. `create --from ID` seeds the title,
description, and acceptance criteria and nothing else, unticked, and
never writes the source. `update --origin` sets and clears, `list
--origin` reads the reverse direction without anything storing it, and
`check` reports `origin_missing`. There is no `origin_cycle`, because
nothing walks the field transitively.

`check` also reports `unknown_series`. Both codes have a section 11 row
and a store fixture, landed together because the tables and the corpus
are one artifact.

A store that declares no series is untouched by all of it. `RenderConfig`
omits the key below schema 2, so a schema-1 `config.yml` is
byte-identical after a rewrite, and no existing fixture or test needed
editing.
