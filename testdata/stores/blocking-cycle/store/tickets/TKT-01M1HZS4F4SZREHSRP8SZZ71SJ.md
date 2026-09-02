---
schema: 1
id: TKT-01M1HZS4F4SZREHSRP8SZZ71SJ
title: Child depending on the epic that waits for it
type: task
status: ready
status_reason: null
priority: normal
labels: []
assignees: []
milestone: null
parent: TKT-01M1HZS4EQFS2102Y7TV75JKJ1
dependencies:
  - TKT-01M1HZS4EQFS2102Y7TV75JKJ1
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

This child names its epic as both parent and dependency. Naming a parent is
ordinary. Depending on it is the mistake, and on its own it is a legal, acyclic
edge: plenty of tickets depend on something they are also filed under.

It only becomes a deadlock because the epic gates on its children. This ticket
carries `blocks_on: none` to make the point that the cycle is not something
both ends opted into. One end did.
