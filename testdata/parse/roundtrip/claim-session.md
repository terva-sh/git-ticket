---
schema: 3
id: TKT-01K3ZYQ9X0M2VBNF7TRC4KDW8H
title: Seed the session task list from a claimed ticket
type: task
status: in-progress
status_reason: null
priority: normal
due_on: null
labels: []
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
claim:
  actor: agent:terva/session-4417
  branch: feat/tickets-to-tasks
  worktree: null
  commit: 4d3c2b1a0f9e
  session: 01M1VXVWZQ8K3PYRFN20HCTE5D
  claimed_at: 2026-09-01T09:00:00Z
  expires_at: null
archive: null
created_at: 2026-08-31T12:00:00Z
updated_at: 2026-09-01T09:00:00Z
created_by:
  id: human:sothr
  name: Drew Short
updated_by:
  id: agent:terva/session-4417
  name: Mieli
extensions: {}
---

## Description

A schema 3 ticket, which is the level that adds `claim.session`, per plan 6.4.

The key sits after `commit` because the three before it say where the work
happened in the repository and this one says where it happened in the record.
Given this ticket, an agent can find the session transcript that worked it and
read what was tried and what failed, none of which is in a diff.

It is the first field a schema level added inside a block rather than at the top
level, and that is why it needed a level at all. An unknown top-level key
round-trips, so a store could carry one and lose nothing. An unknown claim
sub-key does not: `decodeClaim` parses a fixed set and drops the rest, so a
reader that did not know this key would silently discard it on the next write.

The value here is free text and nothing parses it. Only a harness knows the
shape of its own session ids, so validating it would refuse ids that are correct
somewhere this project cannot see.
