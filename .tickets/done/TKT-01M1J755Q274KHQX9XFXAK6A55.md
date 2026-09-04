---
schema: 1
id: TKT-01M1J755Q274KHQX9XFXAK6A55
title: Decide what list shows by default, because done tickets crowd it out
type: spike
status: done
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
references: []
claim: null
archive: null
created_at: 2026-09-02T23:27:06Z
updated_at: 2026-09-04T17:47:57Z
created_by:
  id: human:sothr
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

`git ticket list` with no filter returns every non-archived ticket. In this store that is 44 rows, 30 of them `done`. An agent asking what is open has to pipe the output through something that throws most of it away, which is the tell that the command did not answer the question asked.

The exclusion the default already makes is the argument for changing it. `--archived` is opt-in, so the design has already accepted that a default listing should hide terminal work. It hides the 7 archived and shows the 30 done, and no principle separates those two cases.

Seeing open work today means naming every open status:

```sh
git ticket list --status draft --status ready --status in-progress --status blocked --status review
```

Five flags, and the incantation breaks the moment a status is added. Custom statuses are a live question in section 15 under **Custom statuses**, so that is not hypothetical.

### This is TKT-01M1HVMQQQE3K6VZG7793RXVXN one layer up

That ticket argues `ls .tickets/tickets/` answers a question nobody asks, with 37 of 38 files inactive. Same ratio, same argument. The difference is that `ls` is the fallback view for somebody without the tool, and `list` is the view an agent actually uses, so the command is the more valuable of the two and nobody filed it.

Decide them together. If `list` defaults to open work, some of the pressure to partition the directory goes away.

### Decision 1: change the default, or add a flag

Option A, `list` defaults to open work and `--all` opts back in. Fixes the common case with no new vocabulary to learn. It is a breaking change to human output, which 12.4 does not cover, and to `list --json`, which it does.

Option B, add `--open` as a shorthand for the open statuses and leave the default alone. Not breaking. It also leaves the bad default in place for anyone who has not read the flag list, which is exactly the agent this would help.

Option C, accept `--status open` as a pseudo-status. Cheapest to type, but it puts a value in `--status` that `schema` does not publish, and 12.1 tells a caller to read the schema rather than hard-code the values.

Recommendation is A, with B's flag as the escape hatch. Under 12.4 a default change to `list --json` is a change to a machine-readable surface, so it lands in a minor release with the compatibility note saying so.

### Decision 2: does done count as open

No, and neither does `archived`. Everything else does. State it as an exclusion rather than a list, so that a status added later is included without an edit. TKT-01M1HXHJXRFP7VMH7D35YNTG5H settled the same question the same way for the epics index, and the two must not disagree.

### Where this came from

A review of the work-finding path on 2026-09-02. The reviewing agent ran `git ticket list`, got 44 rows, and filtered them in Python to answer "what is open".

## Summary

Shipped in PR #41. `list` answers with open work, every status except `done` and `archived`, stated as an exclusion so a status added later is included without an edit. Naming a status brings it back and `--all` drops the exclusion.

Both decisions went the way the ticket recommended. Decision 1 took option A, changing the default rather than adding a flag, because 12.4 argues for taking a break while the module is v0.x and nothing consumes the surface. Decision 2 excluded `done` and `archived` as an exclusion, matching how the epics index settled it.

Two consequences the ticket did not anticipate. The library moved with the CLI, so `Filter{}` means open work, and `Filter.IncludeArchived` became `Filter.All` because under the old default `--archived` already meant everything. `search` does not take the default, because finding what was already decided means reading a done ticket.

Plan 8 carries the rule, 10.4 publishes `openStatuses`, 12.4 records both breaks, and 15 records the decisions. Against this store the default went from 53 rows to 16.
