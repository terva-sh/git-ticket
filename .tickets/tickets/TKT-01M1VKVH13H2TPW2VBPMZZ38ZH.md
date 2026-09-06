---
schema: 1
id: TKT-01M1VKVH13H2TPW2VBPMZZ38ZH
title: Build ID series and origin provenance, per plan 5.6
type: task
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - format
assignees: []
milestone: null
parent: null
dependencies:
  - TKT-01M1VKV53GZK2A7PMD011H2KKM
blocks_on: none
references: []
claim:
  actor: agent:terva/mieli
  branch: feat/id-series
  worktree: /home/sothr/workspace/git.local.sothr.com/terva-sh/git-ticket
  commit: 51a8ca7f7ddb45f6650144448e7047bc566d0252
  claimed_at: 2026-09-06T18:14:25Z
  expires_at: null
archive: null
created_at: 2026-09-06T15:02:11Z
updated_at: 2026-09-06T18:16:23Z
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

- [ ] config.yml carries an enforced series key defaulting to [TKT], and create refuses an undeclared prefix with unknown_series
- [ ] A bare ULID fragment resolves across every series, and a prefixed fragment matches both halves or nothing
- [ ] show FRAGMENT never returns a ticket from another series, which is the exit-0 wrong answer measured on 2026-09-05
- [ ] ShortestUnique abbreviates within a series, keeps its signature, and never prints TKT-IDEA-01M
- [ ] --ids series|store|full works on every command that prints more than one ticket, with series the default
- [ ] git ticket series lists, series add declares, and series remove refuses while tickets carry the prefix
- [ ] create --from seeds title, description, and acceptance criteria, records origin, and is refused alongside --template
- [ ] list --origin reads the reverse direction, and check reports origin_missing
- [ ] Section 11 carries the unknown_series and origin_missing rows with fixtures, so TestCorpusCoversEveryPlanCode passes
- [ ] A schema-1 store refuses create --from and series add, naming migrate

## Definition of done

- [ ] just ci is green
- [ ] A real store adopts a second series end to end: migrate, series add, create --series, list, show by prefix
- [ ] docs/plan.md 5.6 is amended in the same commit if the build deviates from it

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
