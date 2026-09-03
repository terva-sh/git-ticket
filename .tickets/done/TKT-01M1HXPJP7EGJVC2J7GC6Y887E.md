---
schema: 1
id: TKT-01M1HXPJP7EGJVC2J7GC6Y887E
title: Record that CI verifies generated artifacts and never commits them
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - ci
  - policy
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references:
  - ref: plan:git-commands
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-02T20:41:51Z
updated_at: 2026-09-03T01:35:49Z
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

### Open: does one command cover artifacts outside the store

Generated artifacts split by scope, and the split is about authority rather than convenience.

Inside `.tickets/`: file placement today, and the epics index once it lands. `check --fix` owns these and already walks the store.

Outside the store: the agent workflow block, which `instructions --write` puts into a consumer's `AGENTS.md` at the repository root.

Option A, they stay separate. `check` is the store-integrity command and has no business rewriting a project's `AGENTS.md`. A tool that edits a prose file the project owns, as a side effect of a CI verification, will eventually clobber something somebody wrote by hand.

Option B, one command, something like `generate --check`, covering both. One thing for a workflow to run and one thing for a person to remember.

Option A is the conservative call and probably the right one. The difference between writing inside a store the tool owns and writing a file the project owns is real, and one flag covering both hides it at exactly the moment a reader should see it.

### Settled: a stale generated artifact is a warning

This is decision 3 on TKT-01M1HXHJXRFP7VMH7D35YNTG5H, settled there and recorded at the end of plan section 15. The two agree by construction.

The tradeoff it looked like does not exist. `--strict` promotes warnings to errors, per 10.3, and CI runs `check --strict`, so a warning is enforced exactly as hard as an error everywhere enforcement actually happens. Nothing is given up by declining to call it an error.

What is kept is the line that a derived file falling behind is not a malformed store, which is the same line that keeps "this ticket is late" out of `check` entirely.

Severity also does not gate repair, which is what makes the warning free rather than merely defensible. `planRepairs` recomputes where each file belongs rather than walking the findings, so a warning is exactly as repairable as an error.

### What this does not ask for

No new command, if decision 1 lands on option A. The deliverable is a plan section and a sentence in the workflow block, not code.

## Acceptance criteria

- [x] The plan records that CI verifies generated artifacts and never commits them, with the reasons.
- [x] The plan records that no writing git command joins the 7.4 table for this purpose.
- [x] check --fix --dry-run is documented as the CI verification entry point.
- [x] The agent workflow block names the command that repairs a red check.
- [x] The open decision, whether one command covers artifacts outside the store, is settled and recorded before any generate command ships.

## Summary

Settled and recorded. Plan section 11 gained "Verifying generated artifacts in CI", 7.4 gained a pointer saying that table binds the tool and not a workflow, and section 15 records both answers.

The open decision landed on Option A. Artifacts inside and outside the store stay under separate commands: `check --fix` owns `.tickets/`, `instructions --write` owns the workflow block at the consumer's repository root. A verification that rewrites a project's own AGENTS.md would eventually clobber something written by hand, and the flag that did it would read as harmless.

Working the ticket corrected its own premise, which is the part worth carrying forward. The description said `check --fix --dry-run` exits 1 when a repair is pending, probed against a misnamed file. That is an error. `epics_index_stale` is a warning, so the same command exits 0 on a stale index. A four-case matrix pinned it: warning without --strict exits 0, warning with --strict exits 1, error exits 1 either way, and nothing is written in any of the four. The documented entry point is therefore `check --fix --dry-run --strict`.

Be precise about what changed for this repository, because it is less than it looks. CI already ran `check --strict`, which already gated a stale index, since --strict promotes the warning. Moving to `check --fix --dry-run --strict` adds the "would rewrite" line naming the repair. It does not change what passes or fails here. The --strict finding matters for anyone writing a fresh workflow from the ticket's original wording, which would have gone green over the condition it was added to catch.

The agent workflow block gained a "When the store check fails" subsection naming both commands. justfile, .forgejo/workflows/ci.yml, README.md, and AGENTS.md moved to the documented entry point, and a `just fix` recipe is the local repair half.

Verified by mutation: a deliberately stale .tickets/epics.md made `just check` print the repair and exit 1, `just fix` restored it, and no residue remained.

No new command shipped, as the ticket predicted for Option A.
