---
schema: 1
id: TKT-01M1Q24QQQZBTV0E3Z58S2S2EK
title: Say how to read a CI status without taking a stale row
type: chore
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - ci
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-04T20:35:41Z
updated_at: 2026-09-04T20:35:41Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

AGENTS.md already says a commit's statuses carry several rows for one context
and to "read the newest and not the first". That wording assumes a person
scanning rows by eye. It does not survive being read by anything that dedupes.

An agent working PR #80 deduped the rows into a dict keyed by context, which
keeps whichever row it visited last. Forgejo returns the array newest-first, so
the surviving row was the oldest. The read printed `pending` three times across
four minutes while the run had been `success` since 20:19:44, and the timestamp
it printed never moved, which is what finally gave it away. A second endpoint,
`actions/tasks`, disagreed and was right.

The failure mode is worth naming because it is silent in the safe direction
only by luck. A stale `pending` costs a few minutes of waiting. The same dedupe
against a forge that returns oldest-first would surface a stale `success` and
merge a red branch.

The rule that survives a script is to sort by `updated_at` and take the last,
or to filter for the newest row per context explicitly. Something like:

    tea api "repos/terva-sh/git-ticket/commits/$(git rev-parse HEAD)/statuses" |
      python3 -c "import json,sys;[print(r['updated_at'],r['status'],r['context']) for r in sorted(json.load(sys.stdin),key=lambda x:x['updated_at'])]"

Worth deciding at the same time whether the guidance should just point at
`actions/tasks?limit=N` instead. It answers with one row per run and no history
to pick through, so there is nothing to dedupe. The catch is that it is not
scoped to a PR, so the caller matches on `head_sha` themselves, and its
`conclusion` field read `None` on every row in that session while `status` read
`success`. Whichever endpoint the rule names, it should name the field that
actually carries the verdict there.

## Acceptance criteria

- [ ] AGENTS.md names the dedupe as the trap, not just which row to read
- [ ] The guidance carries a command whose output is ordered by updated_at
- [ ] It settles whether statuses or actions/tasks is the endpoint to reach for, and names the field carrying the verdict there
