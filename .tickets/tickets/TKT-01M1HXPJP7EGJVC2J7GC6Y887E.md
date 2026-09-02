---
schema: 1
id: TKT-01M1HXPJP7EGJVC2J7GC6Y887E
title: Record that CI verifies generated artifacts and never commits them
type: task
status: draft
status_reason: null
priority: normal
labels:
  - ci
  - policy
assignees: []
milestone: null
parent: null
dependencies: []
references:
  - ref: plan:git-commands
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-02T20:41:51Z
updated_at: 2026-09-02T20:41:51Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

The verification mode already exists. `check --fix --dry-run` exits 1 when a repair is pending, prints what it would do, and writes nothing. Probed against a store holding a misnamed file: exit 1, `would move .tickets/tickets/WRONG-NAME.md -> ...`, and the file untouched afterward.

Once a stale epics index is a finding with a repair, per TKT-01M1HXHJXRFP7VMH7D35YNTG5H, the same command covers it with no new surface.

So what is missing is not a command. It is a written policy, because the obvious next step from "CI can detect this" is "CI should fix this", and that is the step to refuse in writing before somebody takes it.

### CI verifies, it does not commit

Plan 7.4 enumerates the Git commands this code runs. Both only read. `TestGitCommandsAreReadOnly` holds the source to that table, and the section already says no writing command joins it, so a change needing `fetch` or `push` is a change to the plan first.

A workflow could still commit with plain git, outside the tool, which means the rule does not settle the question on its own. The argument has to be won on merits, and it should lose:

- CI would need a write token where it needs none today. A leaked read token and a leaked write token are not the same incident.
- A push to a PR branch races the agent that opened it. AGENTS.md names concurrent agents in separate worktrees as the reason this repository uses pull requests at all, and a bot committing into a branch somebody is working in is that same collision with one more participant.
- A push to `main` from CI breaks the rule that nothing pushes to `main`, and desyncs every agent holding a local one.
- A push from CI retriggers CI, which then needs a skip marker, which is a second mechanism to maintain forever.

The gofmt comparison is the honest one. Nobody wants CI to reformat their code and push it. They want CI to say the code is unformatted, and a one-word local command that fixes it.

### The gap is discoverability, not capability

An agent that sees a red check needs to be told to run `git ticket check --fix`. That belongs in the agent workflow block, which is itself generated and already held to reality by `TestInstructionsNameRealCommands`, so the loop closes without a new mechanism.

### Decision 1: does one command cover artifacts outside the store

Generated artifacts split by scope, and the split is about authority rather than convenience.

Inside `.tickets/`: file placement today, and the epics index once it lands. `check --fix` owns these and already walks the store.

Outside the store: the agent workflow block, which `instructions --write` puts into a consumer's `AGENTS.md` at the repository root.

Option A, they stay separate. `check` is the store-integrity command and has no business rewriting a project's `AGENTS.md`. A tool that edits a prose file the project owns, as a side effect of a CI verification, will eventually clobber something somebody wrote by hand.

Option B, one command, something like `generate --check`, covering both. One thing for a workflow to run and one thing for a person to remember.

Option A is the conservative call and probably the right one. The difference between writing inside a store the tool owns and writing a file the project owns is real, and one flag covering both hides it at exactly the moment a reader should see it.

### Decision 2: is a stale generated artifact an error or a warning

This is decision 3 on TKT-01M1HXHJXRFP7VMH7D35YNTG5H restated, and the two have to agree.

Error means CI is the enforcement, in the way gofmt is enforced, and a PR touching an epic goes red until somebody regenerates. Warning leaves the artifact quietly wrong, which is the failure mode that makes a generated file worse than no file.

Recorded in both places so the answer is not given twice differently.

### What this does not ask for

No new command, if decision 1 lands on option A. The deliverable is a plan section and a sentence in the workflow block, not code.

## Acceptance criteria

- [ ] The plan records that CI verifies generated artifacts and never commits them, with the reasons.
- [ ] The plan records that no writing git command joins the 7.4 table for this purpose.
- [ ] check --fix --dry-run is documented as the CI verification entry point.
- [ ] Decision 1 is settled and recorded before any generate command ships.
- [ ] The agent workflow block names the command that repairs a red check.
