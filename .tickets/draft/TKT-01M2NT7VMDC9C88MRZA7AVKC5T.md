---
schema: 2
id: TKT-01M2NT7VMDC9C88MRZA7AVKC5T
title: Filter ready by label, and let a label filter exclude
type: task
status: draft
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
updated_at: 2026-09-16T19:20:24Z
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
