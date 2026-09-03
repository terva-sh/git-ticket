---
schema: 1
id: TKT-01M1H9X166M1ATNK9S7ET26BVQ
title: Decide what an explicit schema migration looks like
type: spike
status: done
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
updated_at: 2026-09-02T16:48:17Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

A deferred question in plan section 15. Plan 12.4 settles that a store never upgrades itself: reading never rewrites, and a mutation writes back the schema the file already declared, so upgrading a binary cannot make a repository unreadable to a colleague who has not. A store moves only through an explicit migration a person runs, and that operation is undesigned. It has to decide whether it is a CLI command or a library call, whether it converts a whole store or one ticket, what it does about a store other clones cannot read yet, and how check reports a store caught halfway. Nothing needs it until there is a schema 2, and nothing should bump the schema before it exists.

## Summary

Answered in plan 12.5. config.yml declares the store's schema, migrate is the only thing that changes it, it writes config first because a config an old reader rejects fails loudly while a ticket it rejects fails quietly, it converts the whole store under the lock and is idempotent, it never commits, and there is no downgrade. Designing it early found the reason to do so. config.schema was read only to refuse a too-new store and to write itself back, and create stamped the binary's maximum, so a newer binary would have drifted an older store upward with no migration. That rule ships now. The command, Store.Migrate and the check warning land with schema 2, because no fixture can express a merely mixed store while the only parseable level is 1.
