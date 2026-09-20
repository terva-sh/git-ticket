---
schema: 2
id: TKT-01M301NEWXVAF8VJYTR6QPCH4G
title: Widen a pen's rule from requiredLabels to a match record
type: task
status: in-progress
status_reason: null
priority: normal
due_on: null
labels:
  - area/format
  - area/cli
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
claim:
  actor: agent:claude/t3code-a6d0ff31
  branch: t3code/match-record
  worktree: /home/sothr/.t3/worktrees/git-ticket/t3code-a6d0ff31-match
  commit: 23ee2ac82fc86cc6654d07415300479211d2d757
  claimed_at: 2026-09-20T18:36:12Z
  expires_at: null
archive: null
created_at: 2026-09-20T18:36:12Z
updated_at: 2026-09-20T18:36:12Z
created_by:
  id: agent:claude/t3code-a6d0ff31
  name: ""
updated_by:
  id: agent:claude/t3code-a6d0ff31
  name: ""
extensions: {}
---

## Description

Schema 4 of the canvas board file, per plan 12.10 and the canvas repository's TKT-01M2Y91C17YTE0W0P3RBHTF50Y, which holds the design: a pen's requiredLabels becomes a match record with labels, status, type and parent, every field optional, an absent field matching everything, present fields conjoined, and a list within a field a disjunction. Schema 3 keeps opening and requiredLabels reads as match.labels; a write renders schema 4. layout.Route reads every field, git ticket canvas explain reports which fields a candidate failed, canvas show prints the rule, and pen add takes --status, --type and --parent. The canvas normaliser and resolver follow in that repository once this is released. A minor under 12.4: a new schema the reader also understands.

## Acceptance criteria

- [ ] A schema 4 board carries match with labels, status, type and parent, every field optional, and a schema 3 board opens with requiredLabels read as match.labels
- [ ] layout.Route matches on every field, and Candidate says which fields a ticket failed and with what
- [ ] git ticket canvas explain names the failed fields, show and pens print the whole rule, and pen add takes --status, --type and --parent
- [ ] A write renders schema 4, and check --fix rewrites a schema 3 board to it as layout_not_canonical
- [ ] Plan 10.10 and 12.10 describe the record and the envelope fields
