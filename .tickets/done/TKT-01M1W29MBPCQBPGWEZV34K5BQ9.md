---
schema: 2
id: TKT-01M1W29MBPCQBPGWEZV34K5BQ9
title: Migrate this repository's own store to schema 2
type: chore
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - format
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-06T19:14:33Z
updated_at: 2026-09-06T19:19:05Z
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

- [x] config.yml declares schema 2 and every ticket file declares schema 2
- [ ] Every ticket gained an origin: null line and changed nothing else
- [x] No updated_at moved: a migration is not an edit anybody made
- [x] check --strict is clean and CI is green on the migrated store
- [x] create --from works here afterwards, which it could not before

## Definition of done

- [x] The PR lands promptly rather than sitting open, because it touches every ticket file

## Notes

**agent:terva/mieli** at 2026-09-06T19:19:05Z

Migrated. The store is at schema 2 and the diff was measured rather than
trusted.

93 files changed, 255 insertions, 93 deletions. The deletions are exactly
93 occurrences of `schema: 1` and nothing else. The insertions are 93 of
`schema: 2`, 92 of `origin: null`, one `series: []` in config.yml, and then
34 of `due_on: null`, 33 of `blocks_on: none`, one `milestones: []`, and one
`  actor: null`.

Those last four shapes are why acceptance criterion 2 ships unticked. It
reads, verbatim: "Every ticket gained an origin: null line and changed
nothing else". The first half holds, all 92 tickets gained the line. The
second half could not hold for any migration of this store. `migrate`
re-renders each file, and plan 5.3 renders every known field rather than
omitting an absent scalar, so a file last written before a field existed
gains that field the next time anything writes it. The criterion was
unsatisfiable as written, in the TKT-01M1PCY3 sense, so it stays as it was
asked and goes unticked.

That explanation was measured, not assumed. Every file that gained `due_on`
carries an `updated_at` at or before 2026-09-02T21:24:32Z, and every file
that already had it carries one at or after 2026-09-02T23:06:26Z. The two
sets do not overlap, so the split is by last write and not by anything else.

Nothing a person wrote moved. `git diff -U0 | grep -E
'^[+-](updated_at|created_at|title|status|id):' | wc -l` returns 0, which is
criterion 3.

`check --strict` reports no problems and exits 0, and `just ci` is green,
which is criterion 4. `create --from` then produced
TKT-01M1W2CZ1WFZBD13DDW10VFHQS carrying `origin:
TKT-01M1W29MBPCQBPGWEZV34K5BQ9`, and `remove` took it away again with no
residue in `git status`. That is criterion 5, and it is the capability the
migration was for: at schema 1 the same command refused with
`validation_failed` on field `origin`, naming `git ticket migrate` as the
fix.

This ticket was filed before the migration ran, deliberately, so it took part
in it and its own file carries the before-and-after. Its filing commit is
2a8120f and stands alone for that reason.

## Summary

Done. This repository's own store runs at schema 2.

`git ticket migrate` rewrote `.tickets/config.yml` and all 92 ticket files:
config gained `series: []`, every ticket gained `origin: null`, and every
`schema: 1` became `schema: 2`. 34 older files also gained `due_on` and 33
gained `blocks_on`, because plan 5.3 renders every known field and those
fields were introduced after those files were last written. No `updated_at`,
`created_at`, `title`, `status`, or `id` changed.

Four of five acceptance criteria are ticked. The second one asked that
nothing beyond `origin` change, which no migration of this store could
satisfy for the render reason above, so it stays unticked with the
measurement in a note.

`check --strict` is clean, `just ci` is green, and `create --from` now works
here, which it refused to do at schema 1.
