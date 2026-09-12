---
schema: 2
id: TKT-01M29PRW7BGJHTNFQZCWV657A5
title: Give git am of an export a path to a usable store
type: bug
status: draft
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
updated_at: 2026-09-12T02:23:49Z
created_by:
  id: agent:terva/mieli
  name: Mieli
updated_by:
  id: agent:terva/mieli
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

Or state in 12.8 that a receiver runs `git ticket init` before `git am`, which costs nothing but makes the promise conditional on reading the plan.</description>
<parameter name="acceptance_criteria">["A receiver can go from git am of an export to a store that list and check open, using documented commands only", "The store_exists and store_not_found disagreement is resolved, so two commands cannot contradict each other about the same directory", "A test lands an export by git am into a repository with no store and then opens it, so the whole 12.8 promise is exercised rather than its first half"]

## Acceptance criteria

- [ ] A receiver can go from git am of an export to a store that list and check open, using documented commands only
- [ ] The store_exists and store_not_found disagreement is resolved, so two commands cannot contradict each other about the same directory
- [ ] A test lands an export by git am into a repository with no store and then opens it, so the whole 12.8 promise is exercised rather than its first half
