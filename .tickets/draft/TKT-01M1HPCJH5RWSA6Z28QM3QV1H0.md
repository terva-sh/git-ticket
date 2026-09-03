---
schema: 1
id: TKT-01M1HPCJH5RWSA6Z28QM3QV1H0
title: Decide whether the format stores a hand-set order
type: spike
status: draft
status_reason: null
priority: low
labels:
  - question
  - format
assignees: []
milestone: null
parent: null
dependencies: []
references:
  - ref: review:backlog-md
    path: docs/review-backlog-md.md
claim: null
archive: null
created_at: 2026-09-02T18:34:03Z
updated_at: 2026-09-02T18:34:29Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

A deferred question raised by reading Backlog.md. See section B6 of docs/review-backlog-md.md.

Store.List sorts by ID and nothing else, which plan 5.5 makes chronological because a ULID sorts by creation time. priority is a filter and never an order. So the only sequence the format can express is the order tickets were filed in.

Two different wants hide in that. Sorting a list by priority is display: every consumer already has priority on every row, and if human output ever wants it that is a --sort flag with no format change behind it. A hand-set sequence is not display, because no UI can persist a field the format does not have. Backlog.md stores an ordinal per task and maintains it in src/core/reorder.ts, which is what makes its drag-and-drop stick.

The trigger: terva ships a board with reorderable columns, or somebody asks for an ordering within one priority level and a fifth priority is not the answer.

The cost, if the trigger is met, is a frontmatter field on every ticket under plan 5.3, which is every fixture. Worse, an ordinal is the first field in the format whose value is meaningless on its own and only means something relative to its neighbours, so two agents inserting concurrently produce a merge Git cannot resolve sensibly. That interacts with the merge driver question, and whoever settles this should read that one first.
