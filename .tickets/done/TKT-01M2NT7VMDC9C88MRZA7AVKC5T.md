---
schema: 2
id: TKT-01M2NT7VMDC9C88MRZA7AVKC5T
title: Filter ready by label, and let a label filter exclude
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - area/cli
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references:
  - ref: code:list
    path: cli/commands.go
claim: null
archive: null
created_at: 2026-09-16T19:14:02Z
updated_at: 2026-09-16T19:31:17Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:claude/t3code
  name: ""
extensions: {}
---

## Description

`git ticket ready` takes no `--label` flag at all, and `git ticket list --label`
matches rather than excludes. So neither command can answer "everything except
X", which is the query a label vocabulary mostly exists to serve.

### Where this bit

terva keeps a `live-test` label for tickets whose remaining work needs real,
paid model calls. That spend is scheduled for the end of a subscription week, so
the daily question is "what is ready, minus the live-test ones".

`ready` cannot express it, because it has no label filter:

```
usage: git ticket ready

flags:
  --actor  --cross-branch  --ids  --if-revision  --json  --lock-timeout  --store
```

`list` has `--label`, but it is an inclusion match and repeating it ANDs rather
than negates, so it can only name the set to skip, never the set to keep.

The workaround now written into terva's own conventions:

```bash
git ticket ready --json | python3 -c 'import json,sys
for t in json.load(sys.stdin)["tickets"]:
    if "live-test" not in (t.get("labels") or []):
        print(t["id"], t["title"])'
```

That works, and `--json` carrying `labels` is what makes it possible at all. But
it is a JSON pipeline for what reads like a one-flag question, and it has to be
rewritten for every field somebody wants to exclude on.

### Two separable asks

**A negative filter.** Some spelling of `--not-label`, or a `!` prefix on the
existing flag. Whether it generalizes to `--not-status`, `--not-type` and the
rest is upstream's call; label is the one with a real use behind it here.

**`--label` on `ready` at all.** `ready` already shares `list`'s row shape and
most of its plumbing, and it is the command a person actually runs to decide
what to pick up. It is the natural place for a filter and currently has none.

The second is useful on its own even without the first, because "ready, in this
one area" is a query too and it is equally unavailable today.

### Scale, so the ask is not mistaken for a toy

terva's store carries 123 open tickets under a three-dimension label vocabulary
(`area/`, `scope/`, `init/`) minted on 2026-09-16. Composing inclusions works
well: `--label area/permissions --label scope/structural` returns exactly the
careful-review pile. Every exclusion in that vocabulary is a JSON pipeline.

## Acceptance criteria

- [x] ready takes --label, matching what list already accepts
- [x] Both commands take --not-label, repeatable, excluding a ticket that carries any named label
- [x] A ticket carrying no labels is included by --not-label, since everything-except must mean everything
- [x] --label and --not-label compose in one query, and --not-label wins when a label is named on both
- [x] terva's live-test query is expressible in one command, and the JSON pipeline in its conventions is no longer needed

## Notes

**agent:claude/t3code** at 2026-09-16T19:31:03Z

Built as --not-label on both commands, repeatable, settled with the user against a bang or caret prefix. The prefix form would need shell quoting and would foreclose any label starting with that character, and this store now uses area/ prefixes, so punctuation in a label is not hypothetical here.

One correction to this ticket, measured rather than argued. It says list --label repeating 'ANDs rather than negates'. It ORs: matchesAny in ticket/query.go returns on the first match, and against this store --label area/cli gives 4, --label area/tui gives 2, and both flags together give 6.

The conclusion survives and is stronger than the reason given for it. With OR you can name the keep-set in one command, so the problem is not that you cannot express it. It is that the keep-set is a hardcoded vocabulary that rots the moment somebody coins a label, which is the same complaint this ticket makes about greps, and that enumerating labels silently drops every unlabelled ticket. An everything-except query wants those tickets. That case is the one the implementation is built around and the one the test names: a ticket with no labels is excluded by nothing.

ready gained --label as well as --not-label. The ticket only asked for a filter, and inclusion was missing too, so shipping half would have left the next person to file the other half.

Exclusion is applied after inclusion, so a label named on both flags excludes. That combination is the only self-contradicting query a caller can write and it is settled by a test rather than left to evaluation order.

## Summary

list and ready both take --label and --not-label, repeatable. ready had neither. Exclusion is applied after inclusion so a label on both flags excludes, and a ticket carrying no labels is excluded by nothing, which is the case that makes this more than a negated match. Filter.NotLabels and ReadyOptions.Labels/NotLabels are the library half. terva's query is now git ticket ready --not-label live-test. The ticket's claim that repeating --label ANDs was wrong and is corrected in a note: it ORs, and the real argument is that an enumerated keep-set rots and drops unlabelled tickets.
