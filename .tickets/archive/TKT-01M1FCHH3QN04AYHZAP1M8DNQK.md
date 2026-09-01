---
schema: 1
id: TKT-01M1FCHH3QN04AYHZAP1M8DNQK
title: "Phase 3: integrate git-ticket into terva"
type: epic
status: archived
status_reason: null
priority: high
labels:
  - integration
assignees: []
milestone: null
parent: null
dependencies: []
references:
  - ref: proposal:git-ticket
    path: docs/plan.md
claim: null
archive:
  archived_at: 2026-09-01T22:03:57Z
  from_status: draft
  reason: "Terva owns this work and terva has no store here, so tracking it in git-ticket's ledger would go stale the moment terva starts. The content moved to docs/handoff-terva-phase-3.md, which is the artifact a terva agent works from. Archived rather than cancelled because none of it was rejected: it is being done, elsewhere. Unarchive if the split changes."
created_at: 2026-09-01T21:03:30Z
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

Phase 3 of plan section 13. The work lands in terva, not here, and is specified in terva docs/plans/git-ticket.md. It is tracked in this store because this is where the ledger lives, and because dogfooding the format on real cross-repository work is the point.

The layer is decided: the whole surface goes core, not extension-first. terva standard-tools.md defaults to extension-first to avoid committing core surface to an unproven design, which is why terva-tasks went that way. git-ticket is not in that position: format, CLI, corpus, green build, tagged release, and a store this repository runs its own work through. Two of the exceptions standard-tools.md lists also apply. The ticket tools replace a bash pattern, and the board needs harness-level UI an extension cannot reach, because the panel API renders ANSI-stripped lines of text in both the TUI and the web client.

Four slices, each useful alone: the dependency and the command, the read tools, the write tools, the board.

One format limitation found while filing this. Plan 5.5 resolves a references path against this repository root, so these tickets cannot point at terva docs/plans/git-ticket.md by path. They name it in prose instead. Whether a cross-repository reference is worth a scheme is unsettled and not urgent.

## Notes

**agent:terva/mieli** at 2026-09-01T22:03:57Z

archived from draft: Terva owns this work and terva has no store here, so tracking it in git-ticket's ledger would go stale the moment terva starts. The content moved to docs/handoff-terva-phase-3.md, which is the artifact a terva agent works from. Archived rather than cancelled because none of it was rejected: it is being done, elsewhere. Unarchive if the split changes.
