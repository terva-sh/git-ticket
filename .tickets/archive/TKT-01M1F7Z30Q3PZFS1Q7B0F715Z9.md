---
schema: 2
id: TKT-01M1F7Z30Q3PZFS1Q7B0F715Z9
title: Import from Backlog.md, and build a local view
type: task
status: archived
status_reason: null
priority: low
due_on: null
labels:
  - question
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references:
  - ref: proposal:git-ticket
    path: docs/plan.md
claim: null
archive:
  archived_at: 2026-09-06T19:43:07Z
  from_status: draft
  reason: "The importer will not be built: no Backlog.md project to migrate, and what this project wants from Backlog.md is its interface ideas rather than its data. The local-view half shipped in v0.7.0. Successors are TKT-01M1W3RMKSHKK7Q21T7PCE885H and four filed beside it."
created_at: 2026-09-01T19:43:32Z
updated_at: 2026-09-06T19:43:07Z
created_by:
  id: human:sothr
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

A deferred question in plan section 15, and the rest of Phase 4. Backlog.md is the source of the acceptance criteria and definition of done fields, so an importer is a real path in. Worth building when somebody has a backlog to move.

## Notes

**agent:terva/mieli** at 2026-09-02T16:56:19Z

Still parked, but plan section 15 now splits this into two triggers. Import waits for a real Backlog.md project somebody wants to move, with the ticket count as the evidence. The local view keeps the Phase 4 condition, that the file and agent contracts have held through one real project.

**agent:terva/mieli** at 2026-09-05T01:38:38Z

The local-view half is done, so this ticket is now the importer alone.

TKT-01M1QBSD (Build the Phase 4 TUI view for browsing and editing
tickets) shipped `git ticket ui` in v0.7.0: an alt-screen view over the
store with browsing, filtering, status transitions, claim and release,
and create and edit flows. That is the local view this ticket held a
trigger for, and the Phase 4 condition it waited on, the file and agent
contracts holding through one real project, was judged met when the
view was built. The carve-out and the gate decision are recorded in
plan section 13.

This supersedes the local-view half of the 2026-09-02 grooming note
above, which said the view keeps the Phase 4 condition. It kept it, and
then the condition was met.

What remains is the importer, and its trigger is unchanged: a real
Backlog.md project somebody wants to move, with the ticket count as the
evidence. Nothing here is blocked.

**agent:terva/mieli** at 2026-09-06T19:43:07Z

Closed by decision. The importer will not be built, and the trigger is
retired rather than left standing.

The user has no Backlog.md project to migrate, and the relationship between
the two projects is not the one this ticket assumed. We take inspiration from
how Backlog.md solves the problems we face. We do not take its data. An
importer moves data, so it answers a question nobody is asking.

This supersedes the 2026-09-05 note above, which said "What remains is the
importer, and its trigger is unchanged: a real Backlog.md project somebody
wants to move, with the ticket count as the evidence." That trigger is now
withdrawn. Nothing reopens this ticket. If a real Backlog.md project ever
needs moving, that is a new ticket with a live reason behind it, not this one
resurrected.

The local-view half was already recorded as shipped in that same note:
`git ticket ui` landed in v0.7.0 under TKT-01M1QBSD (Build the Phase 4 TUI
view for browsing and editing tickets). Nothing about that changes here.

### What replaces it

The interest in Backlog.md is real and continues, so it moves to work that
matches what we actually want from it. Five tickets, filed on the same branch
that closes this one.

TKT-01M1W3RMKSHKK7Q21T7PCE885H (Review the Backlog.md interface for
ergonomics worth lifting) is the successor question. The 2026-09-02 review
opened by declaring that terva draws the board and we do not, and section C
deferred every display question on that basis. v0.7.0 killed the premise, so
that spike re-reads section C with our own TUI as the consumer. No web UI is
in scope: terva builds that as a library consumer, and a future
git-ticket-web might host a standalone version.

Three lifts from section D were already identified and are still unbuilt, so
they are filed directly rather than waiting on the review to rediscover them:
TKT-01M1W3TM6CNQDQT11NKG2GKWH2 (Ship shell completion for bash, zsh, fish,
and PowerShell), TKT-01M1W3TM7PK7AMNVFBP6E6AXE2 (Archive finished work in
bulk by age), and TKT-01M1W3TM8BBJRSRC17WPFMDXXS (Split the agent instruction
block into named guides).

TKT-01M1W3WK41A9789XYMPD4KBMYX (Credit Backlog.md in NOTICE for the lifts
already shipped) closes a gap this session found. The ruling is that a lift
takes a NOTICE entry whether it is code or an idea, and a file header only
where code was adapted. Three lifts have already shipped, credited in
`docs/plan.md` section 16 and the README but not in NOTICE.

**agent:terva/mieli** at 2026-09-06T19:43:07Z

archived from draft: The importer will not be built: no Backlog.md project to migrate, and what this project wants from Backlog.md is its interface ideas rather than its data. The local-view half shipped in v0.7.0. Successors are TKT-01M1W3RMKSHKK7Q21T7PCE885H and four filed beside it.

## Summary

Archived by decision, not built.

This ticket held two halves. The local view shipped in v0.7.0 as
`git ticket ui`, under TKT-01M1QBSD. The Backlog.md importer will not be
built: there is no Backlog.md project to migrate, and what this project wants
from Backlog.md is its interface ideas rather than its data.

The trigger is retired rather than left standing, so nothing reopens this.

The interest continues as five tickets:
TKT-01M1W3RMKSHKK7Q21T7PCE885H reviews the interface with our own TUI as the
consumer, since the old review deferred every display question to terva and
v0.7.0 ended that premise. TKT-01M1W3TM6CNQDQT11NKG2GKWH2,
TKT-01M1W3TM7PK7AMNVFBP6E6AXE2 and TKT-01M1W3TM8BBJRSRC17WPFMDXXS are the
three section D lifts still unbuilt. TKT-01M1W3WK41A9789XYMPD4KBMYX credits
Backlog.md in NOTICE for what has already been taken.
