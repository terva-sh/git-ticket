---
schema: 1
id: TKT-01M1FCGPFB787F6MQ37QYJY4PA
title: "Slice 2: add the five read tools"
type: task
status: ready
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
archive: null
created_at: 2026-09-01T21:03:03Z
updated_at: 2026-09-01T21:04:21Z
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
