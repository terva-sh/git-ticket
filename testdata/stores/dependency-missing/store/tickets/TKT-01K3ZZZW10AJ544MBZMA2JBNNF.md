---
schema: 1
id: TKT-01K3ZZZW10AJ544MBZMA2JBNNF
title: Waits on a ticket that is not in the store
type: task
status: ready
priority: normal
labels: []
assignees: []
milestone: null
parent: null
dependencies:
  - TKT-01K4001PM01GKMCECKCWBQJFBP
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

The dependency names a well-formed ID that no file in this store defines.

A dangling dependency is an error rather than a warning because the graph
cannot be evaluated at all: `ready` has no way to decide whether this ticket is
blocked, and guessing either answer would be wrong.

The usual cause is a ticket deleted outright instead of archived, which is
exactly why 6.3 makes archive the supported way to retire work.
