---
schema: 1
id: TKT-01M1QBS9F4Y1YY0J90EBW2FTZS
title: Decide what the library must add to serve an interactive TUI host
type: spike
status: draft
status_reason: null
priority: low
due_on: null
labels:
  - question
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-04T23:24:12Z
updated_at: 2026-09-04T23:24:12Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Inventory what the ticket and cli packages must add before an in-process
interactive host can be built on them, per the 2026-09-04 TUI review. The
answer may well be "nothing", and that finding closes this ticket too.

Candidates to answer, each with code references:

- Change detection. The TUI holds a view open while other agents write to the
  store. It needs a cheap way to notice the store changed under it: an mtime
  walk, a revision sweep, or reload on a keypress. Nothing is exposed today.
- Conflict surface. A stale `--if-revision` write must arrive as a typed error
  the host can catch, re-read, and re-present, not as prose on stderr. Check
  what the library mutations return today.
- Section access. The detail view renders Description, Implementation plan,
  Notes, and Comments as separate panes. Confirm the parse surface exports
  section-level access where the TUI can reach it.
- Query completeness. `list`, `search`, the `parent` filter, and cross-branch
  reads exist. Verify the filter set covers the list view: status, label,
  assignee, milestone, type.
- Lock behaviour. An interactive host must hold no lock while a person thinks.
  Confirm the store lock is per-write and bounded, and that the TUI never needs
  a longer hold.

The decision output is either a plan section 12 amendment naming what the TUI
consumes, or a note here that the existing surface suffices.

## Acceptance criteria

- [ ] each candidate in the description is answered with code references
- [ ] plan section 12 is amended, or a note records that no change is needed
