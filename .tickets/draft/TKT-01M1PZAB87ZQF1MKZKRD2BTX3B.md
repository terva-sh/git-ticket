---
schema: 1
id: TKT-01M1PZAB87ZQF1MKZKRD2BTX3B
title: Warn when a write falls back to the config default actor
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - claims
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-04T19:46:19Z
updated_at: 2026-09-04T19:46:19Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

An agent that omits `--actor` has its work filed under whoever heads the
`actors` list in `config.yml`, per 4.1, and nothing says so.

Verified against a store initialised with `human:sothr`. A create, a claim and a
note, all without `--actor`, produced:

    createdBy:    human:sothr
    updatedBy:    human:sothr
    claim actor:  human:sothr
    note byline:  **human:sothr** at 2026-09-04T19:17:42Z

So a person is signed to prose they did not write, and the claim tells every
other agent that a human is holding the ticket. Committing does not repair it,
for the reason 5.3 gives: a commit carries the committer's Git identity while an
actor is a session.

The instructions block now opens by telling an agent to pass `--actor`, which is
the cheap half. This ticket is the half that does not depend on the agent having
read anything.

### Why check is the wrong place, probably

The obvious shape is a `check` finding, but resolution happens at the write and
leaves no trace. Once the file says `human:sothr`, a fallback is byte for byte
identical to a deliberate `--actor human:sothr`. A store with one human writer
who legitimately relies on the fallback is the common case, so a check that
warns on the default actor would be noise there and would train people to
ignore it.

Ways to narrow it, none convincing on their own: warn only when `config.yml`
lists more than one actor, or only when a claim records a branch and commit
while naming a `human:` actor. Both guess.

### The likelier answer

Warn on stderr at the point of the write, naming the actor it fell back to. That
is the `## ` heading precedent: the write still happens, the exit status does not
move, and the warning reaches whoever is looking. It also has the information
the check cannot recover, because at that moment the store knows it fell back.

A stronger option worth weighing: refuse the fallback when `config.yml` lists
more than one actor, on the grounds that a multi-writer store has no sensible
default identity. That is a breaking change for anyone relying on it, so it
needs the 12.4 treatment.

Decide between those before building either.

### Watch for

`resolveActor` in `ticket/apply.go` is the one place the fallback happens, so it
is where the store still knows. The CLI reaches it through `ctx.actor()` in
`cli/cli.go`, which returns an empty `Actor` when no flag was given.

A store listing no actor at all already refuses the write with `invalid_field`
saying `config.yml` lists none. That branch is fine and should not change.

Whatever is chosen goes in plan 4.1 beside the resolution rule, and the CLI
warning table in 12.1 if it becomes a warning.

## Acceptance criteria

- [ ] An agent that omits --actor learns it happened, at the time of the write
- [ ] A single-writer store that legitimately relies on the fallback is not nagged
- [ ] docs/plan.md 4.1 records the decision beside the resolution rule
