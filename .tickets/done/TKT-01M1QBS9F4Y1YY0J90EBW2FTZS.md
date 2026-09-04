---
schema: 1
id: TKT-01M1QBS9F4Y1YY0J90EBW2FTZS
title: Decide what the library must add to serve an interactive TUI host
type: spike
status: done
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
claim:
  actor: agent:terva/mieli
  branch: chore/tui-library-spike
  worktree: /home/sothr/workspace/git.local.sothr.com/terva-sh/git-ticket
  commit: c772c22f77fdac761d16effa99f612c0efae4ffb
  claimed_at: 2026-09-04T23:33:43Z
  expires_at: null
archive: null
created_at: 2026-09-04T23:24:12Z
updated_at: 2026-09-04T23:40:55Z
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

- [x] each candidate in the description is answered with code references
- [x] plan section 12 is amended, or a note records that no change is needed

## Notes

**agent:terva/mieli** at 2026-09-04T23:35:04Z

Spike findings, 2026-09-04, against main at c772c22. Verdict: the existing
surface suffices, and no plan section 12 amendment is needed. Each candidate,
with references:

- Change detection: no watch API exists and none is needed. `Store` holds no
  cache (`ticket/store.go`), so `List` and `Get` re-read the directory on every
  call, and `Ticket.Revision` is the SHA-256 of the bytes on disk at the moment
  of the read, computed and never stored, per plan 7.1 (`ticket/ticket.go`).
  The TUI re-runs `List` on a keypress or a timer and diffs by (ID, Revision).
  At this store's scale, tens of tickets, a full re-list is trivially cheap.
- Conflict surface: typed and complete. A stale precondition returns
  `*ticket.Error` with `Code: CodeStaleRevision`, the ticket ID, the title when
  the bytes still parse, and `Details["expected"]`/`Details["actual"]`
  (`ticket/apply.go`, the block after the lock is held). `CodeOf(err)` extracts
  the code. The check runs after the flock is acquired, so nothing slips
  between check and write. The host catches the code, re-reads, re-presents.
- Section access: exported in full. `Body` carries Preamble, Description,
  AcceptanceCriteria, DefinitionOfDone, ImplementationPlan, Notes, Comments,
  Summary, and `Extra []Section` for unknown sections (`ticket/`, `Body`).
  `Checklist(text)` and `Entries(text)` parse the checklist and dated-entry
  sections into structs.
- Query completeness: `Filter` covers the list view and more: Status, Type,
  Priority, Labels, Assignees, Milestone, Parent (with the empty string for
  top-level), DueBy, All, and CrossBranch (`ticket/`, `Filter`). Nothing the
  TUI list needs is missing.
- Lock behaviour: per-write and bounded. `s.lock()` is taken in exactly four
  places, `Apply`, `Create`, `Fix`, and `Remove`, and released on return.
  Reads take no lock. The flock blocks up to the configured timeout, default
  10s. An interactive host holds no lock while a person thinks, by
  construction.

One observation short of a requirement: `Result.PathsChanged` is absolute and
`Ticket.Path` is where the read happened, so the TUI needs `displayPath`-style
handling from `cli` if it ever prints them. That is a consumer concern, not a
library gap.

## Summary

Spike complete: the library surface suffices for an interactive TUI host, no plan 12 amendment. Change detection is re-list plus (ID, Revision) diff, stale_revision is a typed error with expected/actual details, Body exports every section, Filter covers the list view, and the store lock is per-write with lock-free reads. Details in the note.
