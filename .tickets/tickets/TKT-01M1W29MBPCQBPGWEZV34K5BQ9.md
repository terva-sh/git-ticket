---
schema: 1
id: TKT-01M1W29MBPCQBPGWEZV34K5BQ9
title: Migrate this repository's own store to schema 2
type: chore
status: in-progress
status_reason: null
priority: normal
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
  branch: chore/migrate-store-schema-2
  worktree: /home/sothr/workspace/git.local.sothr.com/terva-sh/git-ticket
  commit: 58169e5f37bcc4ffeb11157ae0ee92136b7f424a
  claimed_at: 2026-09-06T19:15:06Z
  expires_at: null
archive: null
created_at: 2026-09-06T19:14:33Z
updated_at: 2026-09-06T19:15:06Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

This repository's own store is at schema 1. v0.12.0 shipped schema 2 and
`git ticket migrate`, and nothing converts a store on its own: per plan
12.5 a store moves only through a migration a person runs, which is why
`check` reports `migration_incomplete` rather than repairing it.

Migrating buys two things. Frontmatter gains `origin`, so `create --from`
works here and provenance can be recorded. `config.yml` gains `series`, so
this store could adopt a second ID prefix if it ever wants one. Neither is
usable at schema 1: `create --from` is refused outright, naming this
command, and `series add` is too.

The cost is that it rewrites every ticket file in the store, 92 of them
plus `config.yml`, so the diff is wide and shallow. Each file gains one
`origin: null` line and moves `schema: 1` to `schema: 2`. Nothing else
changes, and in particular `updated_at` is not stamped, because a
migration is not an edit anybody made to the work.

That width is the real risk rather than the mechanism. Several agents work
this repository at once in separate worktrees, and a change touching every
ticket file conflicts with anything in flight that edits one. This wants to
land on its own, promptly, and not sit open behind other work.

Once landed, the installed binary must be v0.12.0 or later to read this
store at all. An older one refuses every command with `schema_unsupported`,
which is the designed loud refusal of 5.6 and not a bug.

## Acceptance criteria

- [ ] config.yml declares schema 2 and every ticket file declares schema 2
- [ ] Every ticket gained an origin: null line and changed nothing else
- [ ] No updated_at moved: a migration is not an edit anybody made
- [ ] check --strict is clean and CI is green on the migrated store
- [ ] create --from works here afterwards, which it could not before

## Definition of done

- [ ] The PR lands promptly rather than sitting open, because it touches every ticket file
