---
schema: 1
id: TKT-01K3ZYX2R0ZDAEJ1H456MKF1AN
title: Ticket left mid-merge
type: task
<<<<<<< HEAD
status: in-progress
priority: high
=======
status: review
priority: normal
>>>>>>> feat/other-branch
labels: []
assignees: []
milestone: null
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

Two branches moved the same ticket and the merge was left unresolved.

This is also invalid YAML, and that is the point of the fixture. A reader that
lets the YAML parser speak first reports a mapping error on line 6, which sends
the user looking for a syntax mistake they did not make. Detecting the conflict
markers first turns that into `merge_conflict`, which names what actually
happened and what to do about it.
