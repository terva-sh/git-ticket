---
schema: 1
id: TKT-01M1HPAFJENNESYCVQ7ZV43DT7
title: update cannot set the description, and create cannot set the milestone
type: bug
status: done
status_reason: null
priority: high
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
references:
  - ref: review:backlog-md
    path: docs/review-backlog-md.md
claim: null
archive: null
created_at: 2026-09-02T18:32:54Z
updated_at: 2026-09-02T19:01:14Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Two holes in the flag surface, found by reading Backlog.md against ours. See section A of docs/review-backlog-md.md.

SetDescription exists at ticket/mutation.go:472 and nothing calls it. create --description sets the field through CreateOptions instead, so from the CLI a description is write-once: a typo in the first sentence of a ticket cannot be fixed without hand-editing the file. The mutation is already written and tested; it needs a flag.

create has no --milestone and update has one. Plan section 15 records the argument that settled update --type and --parent, which is that every other field create sets already had an update flag. Nobody ran that argument in the other direction.

Scope: update --description and create --milestone. Both are additive under 12.4. --milestone on create resolves nothing and takes the string as given, the same as update does today.

## Summary

Added update --description and create --milestone.

SetDescription already existed in the library with no caller, so a description was write-once from the CLI: a typo in the first sentence of a ticket needed a hand edit. It now has a flag, and changedFields names it so the human line says what moved.

create --milestone closes the asymmetry the other way. Every field create sets now has an update flag, and every field update changes that create can also set.
