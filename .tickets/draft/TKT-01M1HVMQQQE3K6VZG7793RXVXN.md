---
schema: 1
id: TKT-01M1HVMQQQE3K6VZG7793RXVXN
title: Partition the store into draft, tickets, done, and archive directories
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - format
  - question
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references:
  - ref: plan:store-layout
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-02T20:05:53Z
updated_at: 2026-09-03T00:09:57Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

`.tickets/tickets/` should hold what somebody could work on, and nothing else. Today it holds everything, so a person opening the directory in a forge web UI reads 43 files to find the handful that are live. The split exists for the human. Keeping only actionable work in `tickets/` is what makes it possible to be sure you are looking at what you meant to focus on.

The store becomes a pipeline of four directories, and a ticket moves through them as its status changes:

```
draft/     a ticket somebody filed, not yet worth starting
tickets/   the working set: ready, in-progress, blocked, review
done/      finished recently, still worth reading
archive/   retired, swept out of done periodically
```

### The mapping

| Status | Directory |
|---|---|
| `draft` | `draft/` |
| `ready`, `in-progress`, `blocked`, `review` | `tickets/` |
| `done` | `done/` |
| `archived` | `archive/` |

A ticket is created in `draft/`. When it is written well enough for work or planning to start, somebody moves it to `tickets/`, which is the same promotion the agent workflow block already reserves for a person. When it is marked done it moves to `done/`. `done/` is cleaned into `archive/` periodically, by a person running `git ticket archive`.

### Mandatory, not opt-in

An earlier draft of this ticket proposed an opt-in `directories` key in `config.yml`, and offered closing the ticket in favour of a generated index. Both were rejected. The layout is the format, so there is no config key to shape and no decision left about what it should look like.

That also settles the invariant question the earlier draft raised. Placement is an invariant like `archive/` already is, not an advisory preference applied when the tool happens to touch a file. Advisory placement rots: done tickets end up spread across two directories indefinitely, and then you check both and trust neither, which is worse for the legibility goal than having no `done/` at all.

### Structurally this already exists

`archive/` is already a status-derived directory. 6.3 rules that the status wins when the status and the directory disagree, `archive_location_mismatch` reports the drift, and `check --fix` repairs it by moving the file. The current rule is a status-to-directory function with one hardcoded entry, and this generalizes it to four. That removes a special case rather than adding one.

### Migration is check --fix, and there is no schema bump

The frontmatter `schema` versions the file format, and placement is store layout, not file format. No field changes, no rendering changes, and nothing about a ticket file is different. So this needs no schema bump and no dependency on `git ticket migrate`, which plan 12.5 designs but nothing has built.

`check` reports every file in the wrong directory and `check --fix` moves it, exactly as it already does for the archive. Migrating an existing store is therefore one rename-only commit that reviews cleanly, which is also how this repository's own store moves.

Reads keep working throughout, because 6.3 already says the status wins over the directory. A store that has not been migrated is reported, not broken.

### The finding is renamed

`archive_location_mismatch` becomes `location_mismatch`, because placement stopped being about the archive alone and a misleading name rots. Renaming a code is a break under 12.4, taken now while the module is `v0.x` and nothing consumes the surface, on the same reasoning that took the `list` default break. Every sidecar naming the old code has to follow, which `TestCorpusCoversEveryPlanCode` enforces.

### Costs, accepted rather than dismissed

A status change that crosses a directory boundary is now a rename. Two agents moving one ticket to different statuses collide as a rename conflict rather than a content conflict on `status:`. That collision already exists and this makes it uglier to resolve, which matters because several agents work this repository at once in separate worktrees. The pull request workflow is the mitigation, and it is the same one that already covers two agents editing one ticket.

`git log` needs `--follow` to trace a ticket across its moves. A ticket now moves at most three times in its life, at the three transitions that matter, so the history stays readable.

### Out of scope

A bulk sweep of `done/` into `archive/` is not built here. `git ticket archive ID` exists and the periodic clean is a person running it.

Directories for `ready`, `in-progress`, `blocked`, or `review` are refused. Those are the working set, they churn, and a directory for each would turn every ordinary status transition into a rename. The plan should say so rather than leaving it as an obvious next step.

Directories keyed on anything but status are refused, type included. A path is one dimension, so any partition scheme spends its single slot on one axis, and status has the better claim: done is the property that makes a file uninteresting. The note below records the numbers behind that.

## Acceptance criteria

- [ ] docs/plan.md section 4 carries the four-directory layout and the status-to-directory mapping before any code lands.
- [ ] Placement is mandatory and derived from status: draft/ for draft, tickets/ for ready, in-progress, blocked and review, done/ for done, archive/ for archived.
- [ ] location_mismatch replaces archive_location_mismatch in section 11, in the code, and in every sidecar naming it.
- [ ] check reports a file in the wrong directory for all four directories, and check --fix moves it in one pass.
- [ ] No schema bump and no dependency on migrate: no ticket file changes, only its path.
- [ ] A store that has not been migrated still reads correctly, because 6.3 already rules that the status wins over the directory.
- [ ] A corpus fixture covers a store with a misplaced file, with its sidecar.
- [ ] This repository's own store is migrated by check --fix in one rename-only commit.
- [ ] The plan refuses directories for ready, in-progress, blocked and review, and refuses partitioning on anything but status, type included.

## Notes

**agent:terva/mieli** at 2026-09-02T20:43:21Z

Design conversation settled one constraint on this ticket and raised one threat to it.

Settled: if the store partitions into directories at all, the directory encodes status and nothing else. Epics do not get a directory, and neither does any other type.

A path is one dimension, so any partition scheme spends the single slot on one axis, and status has the better claim because done is the property that makes a file uninteresting. The numbers agree: there is 1 epic across 47 tickets and it sits in archive/, so an epics/ directory would hold zero files today and tickets/ would go from 47 to 46. Type is also mutable and cheaply so. update --type is a frontmatter edit today, and under type directories it becomes a file move, which lands on promoting a task to an epic, the busiest moment in a ticket's life. Epics are also not the boring files, so pulling them out of the default listing removes the most interesting rows.

Whatever shape decision 1 takes, the config key should be documented as the partition dimension rather than as one of several, which closes this door instead of leaving it ajar.

Raised: TKT-01M1HXHJXRFP7VMH7D35YNTG5H proposes a generated .tickets/epics.md, regenerated by check --fix and reported stale by check. It chases the same goal as this ticket, seeing what matters without running the tool, and it gets there with no renames, no path churn, and no git log --follow.

If that ships, most of the motivation here is gone. The two should be decided together, and this ticket may be closed by that one rather than shipped alongside it.

**agent:terva/mieli** at 2026-09-03T00:09:57Z

This ticket was recommended for closure on 2026-09-02 and the recommendation was rejected. Recording both, because the reasoning that survived is narrower than the reasoning that was offered.

The case for closing was that TKT-01M1J755Q274KHQX9XFXAK6A55 fixes the tool view for free, and that a generated index reaches the no-tool view with no renames. The first half is true and shipped: `list` now answers with open work, so an agent asking what is open no longer wades through done tickets.

It does not answer this ticket, because the audiences are different. `list` serves whoever has the binary. The directory split serves a person reading the store in a forge web UI or through `ls`, and a generated index is a second artifact that can go stale where a directory cannot lie about which files are in it.

Three things changed shape as a result. The layout is mandatory rather than opt-in, so the config-key decision the earlier draft raised is moot and there is no `directories` key. The pipeline is four directories rather than two added to two, with `draft/` as where a ticket is filed and `done/` as where recently finished work rests before a periodic sweep into `archive/`. And the migration is `check --fix` with no schema bump, because placement is store layout rather than file format and no ticket file changes.
