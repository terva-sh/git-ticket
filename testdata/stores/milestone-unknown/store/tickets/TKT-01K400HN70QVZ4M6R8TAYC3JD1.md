---
schema: 1
id: TKT-01K400HN70QVZ4M6R8TAYC3JD1
title: Filed against a milestone spelled a second way
type: task
status: ready
status_reason: null
priority: normal
labels:
  - auth
assignees: []
milestone: v1.2.0
parent: null
dependencies: []
references: []
claim: null
archive: null
created_at: 2026-08-31T12:00:00Z
updated_at: 2026-08-31T12:00:00Z
created_by:
  id: human:sothr
  name: Drew Short
updated_by:
  id: human:sothr
  name: Drew Short
extensions: {}
---

## Description

`config.yml` lists `v1.1` and `v1.2`. This ticket says `v1.2.0`, which is either
a third milestone or the second one typed a different way, and nothing in the
file can tell those apart.

That is the whole reason the key exists. A bare scalar with no registry means
`v1.2` and `v1.2.0` are two milestones, so a store accumulates near-duplicates
and the `list --milestone` filter quietly answers about the wrong one.

The strength matches `label_unknown` under 4.1, and for the same reason: a
milestone is how someone marks work before the vocabulary catches up, and
erroring would mean editing `config.yml` before a ticket could name a new
release once. So `check` warns, `--strict` turns it into an error for a
repository that wants the tighter rule, and nothing is ever rewritten.

The `auth` label is in the allowlist, so exactly one finding comes out of this
file rather than two.
