---
schema: 1
id: TKT-01M1VKV53GZK2A7PMD011H2KKM
title: Raise the format to schema 2 and build migrate, per plan 12.5
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
dependencies: []
blocks_on: none
references: []
claim:
  actor: agent:terva/mieli
  branch: feat/schema-2-migrate
  worktree: /home/sothr/workspace/git.local.sothr.com/terva-sh/git-ticket
  commit: 2328d6f3a274908e04d86040d68f1ff9b7fe8fac
  claimed_at: 2026-09-06T17:22:36Z
  expires_at: null
archive: null
created_at: 2026-09-06T15:01:59Z
updated_at: 2026-09-06T17:22:37Z
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

- [ ] ticket.SchemaVersion is 2, and a schema-1 store still parses, renders, and round-trips with no fixture byte changed
- [ ] renderFrontmatter emits origin at schema 2 and omits the key entirely at schema 1
- [ ] The 5.3 round trip holds on a schema-1 fixture and on a schema-2 fixture
- [ ] git ticket migrate and Store.Migrate convert a store in one pass under the lock, and a second run reports nothing to do
- [ ] migrate writes config.yml before any ticket, proven by an interrupted run leaving a store an old reader refuses outright
- [ ] check reports migration_incomplete on a store whose files are behind its config, and check --fix declines to repair it
- [ ] Section 11 carries the migration_incomplete row with a fixture, so TestCorpusCoversEveryPlanCode passes
- [ ] create in a schema-1 store still stamps schema 1, per the rule 12.5 already ships

## Definition of done

- [ ] just ci is green
- [ ] docs/plan.md 12.5 is amended in the same commit if the build deviates from it
