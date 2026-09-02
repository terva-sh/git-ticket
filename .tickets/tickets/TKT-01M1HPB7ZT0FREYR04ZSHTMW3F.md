---
schema: 1
id: TKT-01M1HPB7ZT0FREYR04ZSHTMW3F
title: Add a milestones allowlist to config.yml
type: task
status: done
status_reason: null
priority: normal
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
created_at: 2026-09-02T18:33:19Z
updated_at: 2026-09-02T19:25:08Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

milestone is a bare scalar with no registry. Nothing validates it, so v1.2 and v1.2.0 are two milestones and check cannot tell a typo from a new milestone. A store accumulates near-duplicates and the list filter quietly answers about the wrong one.

config.yml already establishes the pattern. Plan 4.1 makes labels an advisory allowlist that check warns about and never errors on, which is exactly the strength this wants: a typo is worth reporting and a new milestone should not need a config edit before the ticket can be filed.

Backlog.md goes much further, with a file per milestone carrying a description and a due date, and add, rename, remove, archive, and list commands. rename cascades into every task and remove makes the caller choose clear, keep, or reassign. That cascade is real and a UI cannot supply it, but it should wait for a store that wants dates on a milestone. See section B4 of docs/review-backlog-md.md.

Scope: a milestones key in config.yml, and a milestone_unknown warning in plan section 11 beside label_unknown. Adding a warning code is additive under 12.4. The corpus needs a fixture and a sidecar, because TestCorpusCoversEveryPlanCode holds section 11 and testdata to each other.

## Implementation plan

1. Add a milestones key to Config, rendered and parsed beside labels, with
   KnownMilestone carrying the same advisory strength as KnownLabel.
2. Emit milestone_unknown as a warning from Check.
3. Add a milestone-unknown store fixture and its sidecar.
4. Record the key in plan 4.1 and the code in section 11.

## Summary

Added a milestones allowlist and the milestone_unknown warning.

The strength matches label_unknown under 4.1, which is what the ticket asked
for: a typo is worth reporting and naming a new release should not need a
config edit first. An empty list is not an empty vocabulary, it is a store
that has not expressed an opinion, so it permits everything and a store opts
in by listing. The empty milestone is always permitted, because a ticket
naming none is not a ticket naming a wrong one.

The clean fixture already carried v1.2, so its allowlist lists v1.1 and v1.2
and the new fixture names v1.2.0 against it. That is the near-duplicate the
ticket describes, and it comes from the corpus rather than being invented.

Not in scope, and still not: a file per milestone with a description and a due
date, and the rename cascade Backlog.md has. That cascade is real and a UI
cannot supply it, but it wants a store that puts dates on a milestone first.
TKT-01M1HPCVR is the due date question and depends on this.
