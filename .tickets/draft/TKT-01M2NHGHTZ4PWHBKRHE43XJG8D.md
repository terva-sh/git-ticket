---
schema: 2
id: TKT-01M2NHGHTZ4PWHBKRHE43XJG8D
title: Add a doctor command for ticket hygiene
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels: []
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-16T16:41:30Z
updated_at: 2026-09-16T16:41:38Z
created_by:
  id: agent:claude/t3code
  name: ""
updated_by:
  id: agent:claude/t3code
  name: ""
extensions: {}
---

## Description

`git ticket check` answers whether the store is *valid*: a ticket under the wrong filename, a ticket in a directory its status does not imply, a stale `epics.md`, an unresolved reference. Those are things that are broken.

Hygiene is a different question, and nothing asks it. A ticket can be perfectly valid and still be one nobody can pick up: no acceptance criteria, no description beyond a title, `in-progress` with no claim, a claim that expired weeks ago, an epic with no children, a dependency on something already done, a draft that has sat untouched since it was filed. None of that is invalid and all of it costs the next reader time.

Add `git ticket doctor`: hygiene rules, reported by level, in descending order of how much they matter, so the first thing printed is the thing most worth fixing.

`check` and `doctor` stay separate. `check` is what CI runs and what `--fix` repairs, and it must keep answering a yes-or-no question about validity. Hygiene is advice, it is going to be opinionated, and a store that is untidy should not fail a build.

### The rule levels

Rules carry a level and the output is ordered by it, highest first. A store with one serious finding and forty cosmetic ones should open with the serious one rather than bury it.

Whether a level ever fails the command is worth deciding rather than assuming: a `--strict`-style flag that makes the top level an error is the obvious shape, off by default.

### Unresolved, and the reason this is a draft

The note this came from reads: "There should be a doctor command for ticket hygiene rules. Like every ticket should have one or more levels in descending priority."

The first sentence is clear. The second admits at least two readings and they lead to different work:

1. **Rules have levels.** The example is about the command: findings carry a severity and print in descending priority. That is the reading written above.
2. **Tickets should have levels.** The example is a hygiene rule itself — every ticket must sit somewhere in a hierarchy, under an epic, one or more levels deep — and "descending priority" describes the hierarchy rather than the output.

The second is a real and checkable rule, and it is not what this ticket currently describes. Confirm which was meant before starting; if it is the second, both belong here, one as the command and one as its first rule.

## Acceptance criteria

- [ ] Hygiene rules are reported by level, highest first
- [ ] check keeps answering only whether the store is valid, and doctor never fails a build by default
- [ ] Each finding names the ticket and says what would resolve it
- [ ] The ambiguous rule from the filing note is resolved before any rule is written
