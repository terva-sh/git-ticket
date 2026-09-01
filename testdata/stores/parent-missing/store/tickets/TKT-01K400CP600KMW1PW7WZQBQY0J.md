---
schema: 1
id: TKT-01K400CP600KMW1PW7WZQBQY0J
title: Child of an epic that is not in the store
type: task
status: ready
priority: normal
labels: []
assignees: []
milestone: null
parent: TKT-01K400EGS02Y2068ZAB6TY3G9R
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

`parent` names a well-formed ID that no file in this store defines. The epic was
deleted, or this ticket arrived through a merge that did not bring its parent.

`dependencies` is empty, so `dependency_missing` cannot fire and the two codes
stay distinguishable. A checker that reported a dangling parent as
`dependency_missing` would send a reader to the wrong field.
