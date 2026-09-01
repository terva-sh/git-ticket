---
schema: 1
id: TKT-01M1FCGPFB787F6MQ37QYJY4PA
title: "Slice 2: add the five read tools"
type: task
status: archived
status_reason: null
priority: normal
labels:
  - integration
assignees: []
milestone: null
parent: TKT-01M1FCHH3QN04AYHZAP1M8DNQK
dependencies:
  - TKT-01M1FCGPE6X31CDEDRKFWZPYZF
references: []
claim: null
archive:
  archived_at: 2026-09-01T22:03:57Z
  from_status: ready
  reason: "Terva owns this work and terva has no store here, so tracking it in git-ticket's ledger would go stale the moment terva starts. The content moved to docs/handoff-terva-phase-3.md, which is the artifact a terva agent works from. Archived rather than cancelled because none of it was rejected: it is being done, elsewhere. Unarchive if the split changes."
created_at: 2026-09-01T21:03:03Z
updated_at: 2026-09-01T22:03:57Z
created_by:
  id: human:sothr
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

ticket_list, ticket_search, ticket_get, ticket_ready, ticket_check. All local-read, which IsReadOnlyAuthority auto-allows, so they carry no mutation authority and are cheap to review.

Each maps onto a library call that already exists: Store.List, Store.Search, Store.Get, Store.Ready, Store.Check. That mapping is the reason Phase 3 comes after Phase 2 rather than alongside it.

Two requirements from terva standard-tools.md. Results must be capped and pageable, so list and search take offset and limit and report whether more remain: a store with a thousand tickets must not blow a context window. And the tools register only when a store exists, probed with Store.Discover, so a repository without a ticket store pays no schema cost. Visibility is not authority.

## Notes

**agent:terva/mieli** at 2026-09-01T22:03:57Z

archived from ready: Terva owns this work and terva has no store here, so tracking it in git-ticket's ledger would go stale the moment terva starts. The content moved to docs/handoff-terva-phase-3.md, which is the artifact a terva agent works from. Archived rather than cancelled because none of it was rejected: it is being done, elsewhere. Unarchive if the split changes.
