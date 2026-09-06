---
schema: 2
id: TKT-01M1W3RMKSHKK7Q21T7PCE885H
title: Review the Backlog.md interface for ergonomics worth lifting
type: spike
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - question
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-06T19:40:14Z
updated_at: 2026-09-06T19:40:14Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Re-read Backlog.md's interface and decide what else is worth lifting, now
that this project draws its own view.

`docs/review-backlog-md.md` is the 2026-09-02 read, against
`MrLesk/Backlog.md@main`. It produced thirteen tickets and most of them
shipped. This is not a re-run of that review. It is the part that review
deliberately did not do.

That review opened by stating its premise: "terva draws the board and we do
not. So a feature only counts as ours if a UI cannot supply it from what
`--json` already returns." Section C, "Display: terva's job", follows from
that premise and defers every display question. The premise died when v0.7.0
shipped `git ticket ui`. This project now draws a view, so section C is the
stale part and our TUI is the thing that would consume what it declined.

### Scope

No web UI is in scope. terva builds the web surface as a consumer of the
`ticket` and `cli` packages, because that is where the harness already has
one. A future git-ticket-web might host a standalone version of the same
idea. Neither is this ticket, and neither changes what the TUI should do.

### What already shipped, so the review does not re-propose it

From section D, "Operations worth lifting": the refreshable instruction block
landed, and cross-branch visibility landed as `git ticket refs`, reading each
surviving ref with `ls-tree` per plan 7.4.

Three of section D's items are still unbuilt and are filed separately, so
this spike should not re-derive them: shell completion, bulk archive by age,
and splitting the instruction block into named workflow guides.

### Attribution

Per the ruling on this work, a lift gets a NOTICE entry whether it is code or
an idea, and a file header only where actual code was adapted. So the review
must say, for each candidate, which of the two it is. Backlog.md is MIT,
Copyright (c) 2025 Backlog.md.

## Acceptance criteria

- [ ] Section C of docs/review-backlog-md.md is re-read with git ticket ui as the consumer, and its verdict recorded rather than left deferred to terva
- [ ] Backlog.md's own view is compared against our TUI, naming what it does for a person that we do not
- [ ] Each candidate lift is either filed as its own ticket or declined in the review with a reason
- [ ] Each candidate says whether it is a code lift or an idea lift, because the attribution regime differs
- [ ] The review names the Backlog.md revision and the date it was read, as the 2026-09-02 one does

## Definition of done

- [ ] The review lands as a dated document and docs/plan.md says which document is current
- [ ] Follow-up tickets are filed and named in the review, so no candidate survives only as prose
