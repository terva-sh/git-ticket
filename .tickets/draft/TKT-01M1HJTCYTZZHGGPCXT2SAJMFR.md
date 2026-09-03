---
schema: 1
id: TKT-01M1HJTCYTZZHGGPCXT2SAJMFR
title: Decide what a directory must contain to count as a ticket store
type: spike
status: draft
status_reason: null
priority: normal
labels:
  - format
  - question
assignees: []
milestone: null
parent: null
dependencies: []
references:
  - ref: proposal:git-ticket
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-02T17:31:41Z
updated_at: 2026-09-02T18:06:04Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/dev-loop
  name: ""
extensions: {}
---

## Description

A deferred question in plan section 15. Nothing decides what makes a directory a store, so any existing directory is one.

ticket.Open documents that its path is the .tickets directory itself. It returns store_not_found only when the path is missing or is not a directory. A missing config.yml then falls through to DefaultConfig rather than failing, so Open on any directory that exists returns a Store reporting zero tickets and zero unreadable files.

The CLI inherits it. git ticket --store DIR list --json against an empty directory answers kind ticket-list with an empty tickets array and exit 0, and GIT_TICKET_STORE does the same. store_not_found fires only for a path that does not exist. So a typo in a path answers "no tickets" where it should answer "no store", and that is the reading an agent acts on. A tool that lists and sees an empty array concludes there is no work, which is the one wrong answer that looks exactly like a right one.

The markers already exist. init writes config.yml, README.md, tickets/ and archive/ under .tickets, and Discover walks up looking for a directory named .tickets, so the layout in plan 4 is already the definition in everything except the check at open time.

What has to be decided is which marker is the contract, because that is a format promise and not an implementation detail. Requiring config.yml is the cheapest test and the strictest, and it makes a store whose config a user deleted stop opening rather than opening empty. Requiring tickets/ describes the store better and survives a missing config, which 4.1 already treats as defaults. Requiring the directory be named .tickets would break --store pointing at a fixture directory under another name, which the corpus relies on.

Also open is whether the answer belongs to Open alone or to the CLI's store resolution, since a caller passing an explicit path in Go has arguably said what it means, where a person typing --store has not.

Nothing forces this today. It is worth settling before an external consumer builds a tool path on an empty listing, which is Phase 3.
