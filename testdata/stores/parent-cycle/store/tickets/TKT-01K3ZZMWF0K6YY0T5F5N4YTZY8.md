---
schema: 1
id: TKT-01K3ZZMWF0K6YY0T5F5N4YTZY8
title: Parent cycle member, child of the other
type: epic
status: ready
status_reason: null
priority: normal
labels: []
assignees: []
milestone: null
parent: TKT-01K3ZZPQ20RR5ARKYED1FCV2HV
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

Each of these two tickets names the other as its parent.

Section 11 checks the parent graph separately from the dependency graph, and
this store is why. `dependencies` is empty on both files, so a checker that
walked one combined graph would still find a cycle and report the wrong code.
Only a separate walk can report `parent_cycle` here.
