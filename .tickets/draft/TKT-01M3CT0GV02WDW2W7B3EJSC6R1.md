---
schema: 2
id: TKT-01M3CT0GV02WDW2W7B3EJSC6R1
title: Default series stays TKT after a store stops declaring it
type: bug
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - area/cli
  - area/integration
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-25T17:32:34Z
updated_at: 2026-09-25T17:32:34Z
created_by:
  id: agent:claude-code/opus
  name: ""
updated_by:
  id: agent:claude-code/opus
  name: ""
extensions: {}
---

## Description

A store may declare a series list that leaves out TKT, and `series remove TKT` permits it. In that store, every write that relies on the default series fails, because the default is the constant `DefaultSeries = "TKT"` (`ticket/id.go`) and not something the store declares.

### Reproduce (v0.23.0)

```sh
git init /tmp/s && cd /tmp/s && git ticket init
git ticket series add DOC
git ticket series remove TKT          # allowed: no ticket carries TKT yet
git ticket create --title probe
# unknown_series: this store does not declare the series "TKT"; it declares DOC (field series)
git ticket create --series DOC --title probe    # works
```

`import --adopt` fails the same way on its first ticket, and it takes no `--series` flag, so there is no workaround inside the command. Found on 2026-09-25 when setting up a store for `Sothr-Infrastructure/documentation`. The seven tickets that should have been adopted there were re-filed by hand with `create --series DOC`, which loses what adopt records (origin provenance, and with `--same-owner` the filing instant).

### Where it comes from

- `ticket/apply.go`, in create: an empty `o.Series` becomes `DefaultSeries` before the `KnownSeries` check.
- `ticket/id.go`: `DefaultSeries` is a constant. Plan 5.6 says TKT is "the default and the only one until a store declares another", and that an empty list means `[TKT]`. It does not say what the default is once a store declares a list that excludes TKT.
- `create --help` describes `--series` as "the store's default when absent", which reads as a per-store default that does not exist.
- `series remove` refuses only while tickets carry the prefix. It does not consider that the prefix is the default.
- `import --adopt` is documented as filing "every ticket afresh under this store's series", but it has no way to name one.

### Possible fixes, for whoever plans this

1. The store's default series is the first declared entry. With an empty list that is still `[TKT]`, so every existing store keeps its behaviour. The cost is that list order becomes meaningful.
2. An explicit `defaults.series` in `config.yml`, beside `defaults.type` and `defaults.priority`, validated against the declared list and falling back to TKT.
3. `series remove TKT` refuses while TKT is the default, which makes the state unreachable but leaves a store that wants only DOC with no way to get there.
4. `import --adopt --series NAME`, needed alongside 1 or 2 whenever a store declares several series and the import should not land in the default.

1 or 2, together with 4, seems to cover it. Option 3 alone turns the bug into a missing feature. This is a plan 5.6 question, so the decision is the maintainer's.
