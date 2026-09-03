---
schema: 1
id: TKT-01K3ZZY1E0A2Y6DP5AWPQ90872
title: Draft sitting in the working directory
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-08-31T12:00:00Z
updated_at: 2026-09-12T09:00:00Z
created_by:
  id: human:sothr
  name: Drew Short
updated_by:
  id: human:sothr
  name: Drew Short
extensions: {}
---

## Description

This ticket is a draft and the file is in `tickets/`, which is where a store
written before section 4 grew past two directories would have put it.

It is here because the archived case alone cannot tell a generalized rule from
a hardcoded one. A check that still asked "is this in `archive/` and is the
status archived" would pass the archived fixture and stay silent on this file,
so this is the fixture that fails when the rule is only half rewritten.

`tickets/` is the working set. A draft is not startable, so it does not belong
in the directory somebody reads to find what to work on.
