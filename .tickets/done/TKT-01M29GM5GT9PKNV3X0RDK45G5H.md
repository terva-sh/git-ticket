---
schema: 2
id: TKT-01M29GM5GT9PKNV3X0RDK45G5H
title: Make the import preview's shared-series line agree in number
type: bug
status: done
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
updated_at: 2026-09-15T05:01:06Z
created_by:
  id: agent:terva/mieli
  name: Mieli
updated_by:
  id: agent:terva/import-preview-number
  name: ""
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

- [x] The shared-series line reads "1 ticket already uses" at one and "2 tickets already use" at two
- [x] A test covers the count of one, because that is the case that was wrong and the plural was always right
- [x] The chosen helper sits beside plural and were in cli/migrate.go rather than being a third convention

## Implementation plan

Take the general helper rather than a one-off. verb(n, one, many) goes beside plural in cli/migrate.go and were becomes were(n) = verb(n, "was", "were"), so the two are one convention rather than a function per verb. The ticket called one call site thin evidence for the general form, and that is right, but expressing were through it makes two call sites and removes the choice of which convention a third reaches for.

## Notes

**agent:terva/import-preview-number** at 2026-09-15T05:00:03Z

The ticket's scope survey held. Re-read every plural() site in cli against the current tree and this is still the only one that puts a finite verb after the count, so the fix is one format string.

**agent:terva/import-preview-number** at 2026-09-15T05:00:03Z

No test covered this line at all before now, in either number. TestImportAdoptsDespiteASharedSeries asserts the preview offers git am but never reads the sentence that offers it. The new test drives one and two through the real export and import commands rather than calling verb directly, because the bug was in a format string and a unit test on the helper would have passed over it.

## Summary

The import preview reads "1 ticket already uses a series" at one and "2 tickets already use a series" at two. Verified in the built binary against two scratch stores at both counts, which is where the bug was found in the first place.

The helper is the general form. verb(n, one, many) sits beside plural in cli/migrate.go and were is now were(n) = verb(n, "was", "were"). The ticket was right that one call site is thin evidence for a general helper, but expressing were through it makes two and leaves no question about which convention a third reaches for.

The scope survey in the description still held against the current tree: this is the only plural() site in cli that puts a finite verb after the count, so the repair is one format string.

The line had no test in either number. TestImportAdoptsDespiteASharedSeries asserts the preview offers git am but never reads the sentence doing the offering. The new test drives one and two through the real export and import commands rather than calling verb directly, because the defect was in a format string and a unit test on the helper would have passed straight over it. Falsified by reverting the call site to the literal.
