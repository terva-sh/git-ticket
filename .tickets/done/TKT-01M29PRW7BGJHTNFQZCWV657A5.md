---
schema: 2
id: TKT-01M29PRW7BGJHTNFQZCWV657A5
title: Give git am of an export a path to a usable store
type: bug
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - integration
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references:
  - ref: plan:12.8
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-12T02:22:32Z
updated_at: 2026-09-15T05:08:16Z
created_by:
  id: agent:terva/mieli
  name: Mieli
updated_by:
  id: agent:terva/init-adopts
  name: ""
extensions: {}
---

## Description

Plan 12.8 promises a receiver types `git am DIR/*.patch` and needs no git-ticket at all. Following that promise into a repository that has never heard of git-ticket produces a store no command can open, and no CLI path repairs it.

An export carries the ticket files and nothing else. `exportTicketFiles` writes one hunk per ticket and no `config.yml`, so `git am` creates `.tickets/draft/TKT-....md` and stops there.

From that state the two halves of the CLI disagree about whether a store exists:

    $ git ticket list --all
    git-ticket: store_not_found: no ticket store at .../.tickets: no config.yml

    $ git ticket init
    git-ticket: store_exists: a ticket store already exists at .../.tickets

`init` has no `--force` or equivalent. Every read refuses because `config.yml` is missing, and the one command that would write `config.yml` refuses because the directory is there. The receiver is left holding valid ticket files and no way to use them, short of hand-writing a `config.yml` or moving the files aside and running `init` first.

### Measured, not reasoned

Reproduced on Linux with `core.autocrlf=false` and pure LF throughout, so this is nothing to do with line endings or platform. `git am` applied cleanly, the landed file carried 0 CR bytes, and the two commands above answered exactly as shown. Run against the released v0.17.1 binary.

### Why it was not noticed

The v0.16.0 verification landed an export with `git am` into a third repository and confirmed the patch applied and left readable Markdown. That is true and is what was checked. It did not go on to ask whether git-ticket could then open the result.

`import --adopt` is unaffected, because it re-renders through the library and never needs the arriving files to form a store. The promise that breaks is the `git am` one, which is the half aimed at a receiver who does not have git-ticket and then wants it.

### Candidate repairs

Let `init` adopt a `.tickets` directory that has no `config.yml`, which is the smallest change and turns the dead end into one command. The refusal exists to stop a second store clobbering a first, and a directory with no `config.yml` is not a store by the definition every read already uses.

Or have export carry a `config.yml` hunk, which collides with whatever the receiver may already have and makes an export assert a store's configuration rather than its tickets.

Or state in 12.8 that a receiver runs `git ticket init` before `git am`, which costs nothing but makes the promise conditional on reading the plan.

## Acceptance criteria

- [x] A receiver can go from git am of an export to a store that list and check open, using documented commands only
- [x] The store_exists and store_not_found disagreement is resolved, so two commands cannot contradict each other about the same directory
- [x] A test lands an export by git am into a repository with no store and then opens it, so the whole 12.8 promise is exercised rather than its first half

## Implementation plan

Take the first candidate repair, and take it as a definition rather than as a special case. config.yml is what every read keys on, so Init keys on it too instead of on the directory. That resolves the contradiction at its source: both commands then answer the same question about the same directory, and adoption falls out rather than being handled.

Two things ride along because the repair is not finished without them. Init builds the epics index from the tickets it finds rather than writing an empty one, since the stated reason that file is written at all is that a store should not be born reporting a warning it did nothing to earn, and an adopted store earns one. And the CLI names each undeclared series with the series add that declares it.

Init does not declare the series itself. A store's series list says what this project mints, and inferring it from a file that happens to be present would have init decide something belonging to whoever owns the store. Advice, not action, which is what export and import already do about the half of a move this tool will not perform.

## Notes

**agent:terva/init-adopts** at 2026-09-15T05:07:49Z

Cleaned two lines of leaked tool-call syntax off the end of the description, a stray </description> and a <parameter name="acceptance_criteria"> block repeating the criteria as JSON. It came from whatever filed the ticket. check --strict passes over it, so nothing would have reported it; the criteria themselves were already parsed correctly into their own section and are unchanged.

**agent:terva/init-adopts** at 2026-09-15T05:08:03Z

Measured before choosing. Letting init adopt is necessary but not sufficient: with a custom series the receiver still gets unknown_series, and check --fix does not repair it because the prefix sits inside an immutable ID. The full documented route is git am, init, series add, and that is what the test walks. For the default TKT series it is git am and init, and the store is clean.

**agent:terva/init-adopts** at 2026-09-15T05:08:03Z

Rejected the other two candidates the description named. Having export carry a config.yml hunk makes an export assert the receiver's configuration and collides with any config already there. Stating in 12.8 that a receiver runs init before git am leaves the two commands still contradicting each other for anyone who does it in the other order, and the promise is then conditional on having read the plan.

**agent:terva/init-adopts** at 2026-09-15T05:08:03Z

Falsification caught a vacuous assertion of mine. The epics-index check in TestInitAdoptsADirectoryWithNoConfig passed with the rebuild reverted, because the arrived fixture was a task and the index lists epics only, so both renderings were the empty index. The fixture is an epic now and the assertion fails when the rebuild is removed.

## Summary

git am of an export now reaches a usable store. The route is git am, then git ticket init, then git ticket series add NAME when the export carries a series this store has never declared, and the tickets keep their original IDs throughout. Verified end to end in the built binary for both a custom series and the default TKT one, where init alone is enough.

The repair is a definition rather than a special case. config.yml is the file every read keys on, so Init keys on it too instead of on the directory existing. The store_exists and store_not_found contradiction goes away because both commands now answer the same question about the same directory, and adoption falls out of that rather than being handled separately. Init still refuses a real store, which is the clobbering the check was put there to stop, and a test holds that.

Init also builds the epics index from the tickets it finds rather than writing an empty one. The stated reason that file is written at all is that a store should not be born reporting a warning it did nothing to earn, and an adopted store earned one.

Init names each undeclared series and the series add that declares it, but does not declare it. A store's series list says what this project mints, so inferring it from a file that happens to be present would have init decide something that belongs to whoever owns the store.

Three tests. TestInitAdoptsADirectoryWithNoConfig and TestInitStillRefusesARealStore hold the two halves at the library, and TestGitAmOfAnExportReachesAUsableStore walks the whole 12.8 promise through the CLI, which is what the v0.16.0 verification did not do: it confirmed the patch applied and left readable Markdown, and never asked whether git-ticket could open the result. All four new assertions were falsified before being trusted, and one of them was vacuous until the fixture became an epic.

Plan 12.8 records the ruling and the route. newGitStore is now newGitRepo plus an init, so a test can watch a store come into being.
