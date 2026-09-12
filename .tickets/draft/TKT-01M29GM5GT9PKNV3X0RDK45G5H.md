---
schema: 2
id: TKT-01M29GM5GT9PKNV3X0RDK45G5H
title: Make the import preview's shared-series line agree in number
type: bug
status: draft
status_reason: null
priority: low
due_on: null
labels: []
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references:
  - ref: plan:12.8
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-12T00:35:06Z
updated_at: 2026-09-12T00:35:06Z
created_by:
  id: agent:terva/mieli
  name: Mieli
updated_by:
  id: agent:terva/mieli
  name: Mieli
extensions: {}
---

## Description

`git ticket import` previewing an export whose tickets use a series this store
declares prints a line whose verb does not agree with its count:

```text
1 ticket already use a series this store declares. If this export is your own
project's work, `git am DIR/*.patch` keeps the original IDs and is the better route.
```

"1 ticket already use" should be "uses". At two or more it is already correct,
which is why it survived review: the sentence was written against the plural.

`cli/import.go:180` is the site. The count comes from `plural(shared, "ticket")`
and the verb is a literal `use` in the format string.

### The repair already has a shape in the tree

`cli/migrate.go` carries `plural(n, noun)` and `were(n)` side by side, and
`writeMigrateUnreadable` uses them together:

```go
fmt.Fprintf(w, "%s did not parse and %s left alone; run check\n",
    plural(len(r.Unreadable), "file"), were(len(r.Unreadable)))
```

So this is a helper beside `were`, not a new idea. Whether it is a general
`verb(n, singular, plural)` or a one-off is the only open question, and one call
site is thin evidence for the general form.

### Scope, measured rather than assumed

Every `plural()` call site in `cli` was read. This is the only one that puts a
finite verb after the count. `export.go:166` says "%s naming a ticket", a
participle that agrees at any count. `import.go:164` and `185` attach their verb
to the directory rather than the count. `import.go:175`, `215` and
`migrate.go:59` and `78` have no verb after the count. `migrate.go:91` already
calls `were`.

### Where it was found

The v0.16.0 release verification, running the shipped binary against two scratch
stores. It is cosmetic and blocked nothing, and the release went out with it.

## Acceptance criteria

- [ ] The shared-series line reads "1 ticket already uses" at one and "2 tickets already use" at two
- [ ] A test covers the count of one, because that is the case that was wrong and the plural was always right
- [ ] The chosen helper sits beside plural and were in cli/migrate.go rather than being a third convention
