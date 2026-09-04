---
schema: 1
id: TKT-01M1PCYC14VD3TJYDE1EWC671Y
title: Decide how a ticket filed by mistake is removed
type: spike
status: draft
status_reason: null
priority: normal
due_on: null
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-04T14:25:12Z
updated_at: 2026-09-04T14:25:12Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

There is no delete. archive is the soft close and it keeps the file, which is right for work that happened and wrong for a ticket that should never have existed.

The case is concrete and AGENTS.md already documents it. Text passed to a body section carrying a heading of two hashes is split into sections by parseBody, so a description written with Markdown subheadings lands partly in body.extra. The CLI warns on stderr and writes anyway, because passing several sections in one string is sometimes meant. The repair AGENTS.md records is to remove the file by hand and create the ticket again, because update --description replaces only the description and the stray sections survive. TKT-01M1HVMQQQE3K6VZG7793RXVXN had to be filed twice for exactly this.

So the tool documents a repair it cannot itself perform. That is the gap.

What to settle. Whether a removal exists at all, given that archive covers the honest close and a Git-native store can always be repaired with rm and a commit. If it does, whether it refuses anything carrying history, meaning notes, comments or a claim, so it can only take a ticket nobody has worked. And whether it is its own command or a flag on one that exists.

The counter-argument deserves a fair hearing. rm on the file plus a commit is the whole operation, the store is files by design, and a command wrapping one rm earns its place only by refusing the cases where rm is wrong. That refusal may be the actual product here.

This carries no question label on purpose. That label marks a question parked in plan section 15, and every ticket holding it has an entry there. This is queued work rather than a parked question, so labelling it would break the invariant that section 15 names every question ticket.

## Acceptance criteria

- [ ] docs/plan.md records whether a removal exists and what it refuses
- [ ] The mis-sectioned repair in AGENTS.md names a command, or is rewritten to say the format has none
