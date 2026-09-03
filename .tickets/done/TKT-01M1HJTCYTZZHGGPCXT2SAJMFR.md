---
schema: 1
id: TKT-01M1HJTCYTZZHGGPCXT2SAJMFR
title: Decide what a directory must contain to count as a ticket store
type: spike
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - format
  - question
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references:
  - ref: proposal:git-ticket
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-02T17:31:41Z
updated_at: 2026-09-03T03:22:01Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
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

## Summary

Settled and shipped. A directory is a ticket store when it holds `config.yml`, enforced in `Open` so the library carries the rule and the CLI inherits it. `store_not_found` now names the file it wanted, so a reader learns what would make the directory a store.

The decision was made twice. It was first settled as `config.yml` and `tickets/` together, the strictest option and the one matching what `init` writes and what all 17 fixtures carry. Git overruled it within minutes. Git tracks no empty directory, so `init`, commit, clone produces a store of three files and no subdirectories at all, and requiring `tickets/` rejected it. `TestWorktreesShareOneLock` failed on exactly that. The rule would also have rejected any store that finished its open work, which is to say it punished a store for being up to date. A directory cannot be a marker in a format kept in Git unless something keeps it non-empty, and nothing here does.

That is the finding worth carrying past this ticket. The choice looked like a trade between strictness and tolerance, and it was really a question about what Git stores.

### What the investigation added

The premise held on probing, which the last two tickets did not. `--store` at an existing empty directory answered `ticket-list` with an empty array at exit 0, and `ready` printed that nothing was ready to pick up.

One thing the ticket did not know: `check --strict` already refused an empty directory, but through `epics_index_stale`, whose message says `epics.md` "does not match the epics in this store" while failing to be a store. Plain `check` exited 0 on the same directory, because that finding is a warning. It was right that the file was missing and wrong about what that meant. Both forms now stop at the store.

### Verification

Three mutations, each confirmed red and reverted: the guard disabled, the reverted `tickets/` rule, and the message without the filename. The reverted rule is the one that matters, because it proves `TestAClonedStoreOpens` encodes the discovery rather than restating it. That test also asserts the clone genuinely lacks `tickets/` before opening it, so it cannot pass for the wrong reason if Git ever changes.
