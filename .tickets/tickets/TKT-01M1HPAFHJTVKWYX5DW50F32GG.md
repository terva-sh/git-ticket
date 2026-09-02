---
schema: 1
id: TKT-01M1HPAFHJTVKWYX5DW50F32GG
title: Add a mutation for the implementation plan
type: bug
status: done
status_reason: null
priority: high
labels:
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
created_at: 2026-09-02T18:32:54Z
updated_at: 2026-09-02T18:46:45Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Plan 5.2 lists "## Implementation plan" as a known body section. ticket/parse.go reads it, ticket/render.go writes it, ticket/query.go searches it, show prints it, and cli/json.go carries it as body.implementationPlan. There is no SetImplementationPlan mutation and no CLI flag, so neither the library nor the CLI can write the section.

The only way to fill it is to hand-edit the ticket file, which is the one thing the ledger exists to stop. cli/instructions.md cannot tell an agent to write a plan before it writes code, because there is no command to name.

Backlog.md is what made this visible. Its argument for the whole format is three review checkpoints, and the middle one is that the agent researches the codebase, writes its plan into the task, and waits for approval. See section A of docs/review-backlog-md.md.

Scope: a SetImplementationPlan mutation, a CLI path to it, and create --plan so a ticket can be seeded with one. Decide replace against append on the argument plan section 9 already uses for summary, which is that a plan is one statement of how the work will go and a log of those is what Notes is for. Replace is probably right, and an append variant can follow if the log turns out to be wanted.

## Implementation plan

1. Add SetImplementationPlan beside SetSummary, replacing rather than appending
2. Route the CLI through runTextEntry, which note, comment, and summary already share
3. Dispatch plan between dod and note so the commands follow plan 5.2 section order
4. Add create --plan through CreateOptions
5. Move plan.md 9 and 12.1, the README, and the instructions block in the same commit

## Summary

Added SetImplementationPlan, git ticket plan ID TEXT, and create --plan, so the section plan 5.2 defined is now writable by the tool rather than only by hand.

The command shares runTextEntry with note, comment, and summary, and is dispatched between dod and note so the four body-section commands appear in the order 5.2 gives the sections. It replaces rather than appends, on the argument section 9 already made for summary.

Adding it turned up a second defect. Parse trims section text and render writes it verbatim, so create --description with padded input wrote bytes that did not survive the round trip 5.3 requires. Create now trims the sections it seeds, which makes it agree with every Set* mutation. Where that guarantee belongs is filed as a question rather than settled here.

The instructions block now tells an agent to research and write a plan after claiming, and TestInstructionsWorkflowRuns runs that step in the position the block prints it.
