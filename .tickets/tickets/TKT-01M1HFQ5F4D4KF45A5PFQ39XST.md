---
schema: 1
id: TKT-01M1HFQ5F4D4KF45A5PFQ39XST
title: Decide how git-ticket integrates with an external tracker
type: spike
status: draft
status_reason: null
priority: normal
labels:
  - format
assignees: []
milestone: null
parent: null
dependencies: []
references: []
claim: null
archive: null
created_at: 2026-09-02T16:37:30Z
updated_at: 2026-09-02T16:37:30Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Deferred question 10 of plan section 15. Storing an external identifier already works. jira:PROJ-1234, gh:owner/repo#88 and url:... all link cleanly, check --strict stays green, and decodeReferences takes the ref verbatim, so the namespace is open by design per 5.1. Three gaps sit above that storage. Nothing enforces a namespace, so an untyped PROJ-1234 is accepted and JIRA:proj-1234 and jira:PROJ-1234 are unrelated references, which sits badly with 5.1 calling a reference a typed stable identifier. There is no lookup by ref, so which ticket is PROJ-1234 falls to search, a substring match across title, description, notes and comments that also matches a ticket merely mentioning the number, and that is the wrong instrument for a sync wanting one ticket per issue. files PATH exists for the file: namespace and has no equivalent for the others. And extensions, which 5.1 calls the only place a consumer may write fields the core does not define, has no mutation at all, so it round-trips through parse and render but can only be written by hand-editing the file.
