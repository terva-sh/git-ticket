---
schema: 1
id: TKT-01M1Q24QQQZBTV0E3Z58S2S2EK
title: Say how to read a CI status without taking a stale row
type: chore
status: done
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
updated_at: 2026-09-04T20:51:12Z
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

- [x] AGENTS.md names the dedupe as the trap, not just which row to read
- [x] The guidance carries a command whose output is ordered by updated_at
- [x] It settles whether statuses or actions/tasks is the endpoint to reach for, and names the field carrying the verdict there

## Notes

**agent:terva/mieli** at 2026-09-04T20:50:11Z

Reading this PR's own CI with the new rule caught a defect in the new rule.

The first wording said the singular endpoint "answers `total_count: 2` with one
row per context". That number was copied from a settled commit, where both
`(pull_request)` and `(push)` had reported. Forty seconds after pushing, the
same endpoint answered `total_count: 1`, because `(push)` had not fired yet.

A hard-coded 2 invites the reader to treat the count as a completeness check and
wait for a second row that arrives minutes later or, on a branch with no push
build, never. The rule now says the count is how many contexts have reported so
far, not how many will, and tells the reader not to wait for a particular value.

Worth noting the shape of the mistake, because it is the same one the ticket is
about: an observation taken from one moment, written down as though it were the
invariant.

**agent:terva/mieli** at 2026-09-04T20:51:12Z

Correcting the note above, which got the mechanism wrong in the same way it was
complaining about.

It said `(push)` "had not fired yet", and the first repair said the two contexts
post minutes apart. Both assume the row is coming. It is not. `.forgejo/workflows/ci.yml`
triggers on `pull_request` and on `push` to `main` only, so a PR branch gets the
`(pull_request)` row and never a second one. The `(push)` row appears when the
commit lands on `main`, which is at merge.

Checked against three commits. `802a77e`, an open PR head, answers
`total_count: 1` and still did minutes later. `6066b70` and `8158e78` both
answer 2, and in each case the `(push)` timestamp sits a few minutes after the
`(pull_request)` one, at the time that PR was merged rather than at the time its
branch was pushed.

So the rule now tells the reader not to poll for a second row on a branch, which
is the actionable form. Two wrong guesses about the same field, both from
reading one moment's output and generalising, before checking the trigger block
that settles it in two lines.

## Summary

The answer was a third endpoint, which neither option in the third criterion
named. `GET /commits/{sha}/status`, singular, does the collapsing server-side:
one row per context, each already the newest, plus a top-level `state`. There is
nothing left to dedupe, so the trap cannot be walked into.

Reproducing the trap first is what settled it. The plural `/statuses` groups its
rows by context and orders each group newest first, so on `6066b70` the array
reads `success, pending, pending, success, pending, pending`. A dict keyed on
context in one pass keeps the last row it visits, which is the oldest of each
group. That is not a subtle ordering assumption, it is the documented shape
producing a wrong answer from correct-looking code.

AGENTS.md now names the dedupe rather than telling a reader which row to prefer,
because the old wording assumed a person scanning rows by eye and said nothing
to anybody writing a script.

`actions/tasks` stays as the fallback for when no status exists yet, with its
three edges recorded: `?limit=N` is ignored, and a request for 3 rows came back
with 185; `conclusion` is null on every row including the failures, so `status`
carries the verdict; and nothing scopes it to a commit, so the caller matches
`head_sha`.

The second criterion asked for a command whose output is ordered by
`updated_at`. The chosen endpoint does not need one, so rather than reword the
criterion to match what shipped, the plural path kept a real sorted command and
the criterion is satisfied as written.

Both commands were extracted from AGENTS.md and run, rather than retyped from
memory. The curl fallback further down named the plural path and would have
contradicted the new rule, so it moved to the singular one too.

The top-level `state` is deliberately not described as folding contexts
together. Every red commit in this repository's history carries a single
context, so a disagreement between two was never observed, and the rule points
at the `(pull_request)` row instead, which is what gates a PR anyway.
