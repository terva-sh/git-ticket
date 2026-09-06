---
schema: 2
id: TKT-01K3ZYQ7M0R4TBFC5XN2JHD8SW
title: Rotate the signing key without downtime
type: task
status: ready
status_reason: null
priority: normal
due_on: null
labels: []
assignees: []
milestone: null
parent: null
origin: TKT-01K3ZYRB40N8QVD2M5T7XCFH3J
dependencies: []
blocks_on: none
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

A schema 2 ticket, which is the level that adds `origin`, per plan 5.6.

The key sits directly after `parent` because both name a single ticket and 5.1
orders them together. It is a known field here rather than an unknown one, so it
renders in that position instead of after `extensions`, which is what
distinguishes this file from `origin-below-schema.md`.

`origin` names the ticket this one was created from. Nothing resolves it in a
parse fixture, because a single file is not a store and `origin_missing` is a
store-scoped finding.
