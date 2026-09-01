---
schema: 1
id: TKT-01M1FCGPH5M9EMBHTG27JFRFKV
title: "Slice 4: add the kanban board"
type: task
status: archived
status_reason: null
priority: normal
labels:
  - integration
assignees: []
milestone: null
parent: TKT-01M1FCHH3QN04AYHZAP1M8DNQK
dependencies:
  - TKT-01M1FCGPG9Z1TMEX28AKAAFXB1
references: []
claim: null
archive:
  archived_at: 2026-09-01T22:03:57Z
  from_status: ready
  reason: "Terva owns this work and terva has no store here, so tracking it in git-ticket's ledger would go stale the moment terva starts. The content moved to docs/handoff-terva-phase-3.md, which is the artifact a terva agent works from. Archived rather than cancelled because none of it was rejected: it is being done, elsewhere. Unarchive if the split changes."
created_at: 2026-09-01T21:03:03Z
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

A kanban view over the store: columns by status, cards for tickets, pointer interaction. In-tree Preact under terva packages/agent/web/client/src, over ctrlproto, alongside the control panel.

This slice is why the whole surface goes core. An extension can own a panel and that panel does render in the web client, but the panel model is lines of text in both frontends: PanelBody joins ANSI-stripped lines. An extension gets a monospace blob plus key events and cannot emit columns, cards, or pointer targets.

The board reads through the same library as the tools and must not parse a ticket file itself. Moving a card is a status transition through Store.Apply with the same revision precondition and the same permission class as ticket_transition. A drag is a mutation, and being a mouse gesture does not change its authority.

Deliberately last: what a card should show is cheaper to answer after using the tools than by guessing now.

## Notes

**agent:terva/mieli** at 2026-09-01T22:03:57Z

archived from ready: Terva owns this work and terva has no store here, so tracking it in git-ticket's ledger would go stale the moment terva starts. The content moved to docs/handoff-terva-phase-3.md, which is the artifact a terva agent works from. Archived rather than cancelled because none of it was rejected: it is being done, elsewhere. Unarchive if the split changes.
