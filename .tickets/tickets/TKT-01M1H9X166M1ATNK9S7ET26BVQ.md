---
schema: 1
id: TKT-01M1H9X166M1ATNK9S7ET26BVQ
title: Decide what an explicit schema migration looks like
type: spike
status: draft
status_reason: null
priority: normal
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
references: []
claim: null
archive: null
created_at: 2026-09-02T14:55:50Z
updated_at: 2026-09-02T14:55:50Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Deferred question 8 of plan section 15. Plan 12.4 settles that a store never upgrades itself: reading never rewrites, and a mutation writes back the schema the file already declared, so upgrading a binary cannot make a repository unreadable to a colleague who has not. A store moves only through an explicit migration a person runs, and that operation is undesigned. It has to decide whether it is a CLI command or a library call, whether it converts a whole store or one ticket, what it does about a store other clones cannot read yet, and how check reports a store caught halfway. Nothing needs it until there is a schema 2, and nothing should bump the schema before it exists.
