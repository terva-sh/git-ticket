---
schema: 2
id: TKT-01M294KXFASNKBXT3YSH73WJJR
title: Decide what git ticket import publishes under --json
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - question
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references:
  - ref: plan:12.8
    path: docs/plan.md
  - ref: plan:10
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-11T21:05:15Z
updated_at: 2026-09-11T23:06:09Z
created_by:
  id: agent:terva/mieli
  name: Mieli
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

`import` has no `--json` form. Plan 12.1 exempts only `ui`, and 12.7 exempts
`copy` with a reason, so this is a gap rather than a decision.

The exemptions that exist do not stretch to cover it. `ui` is a terminal and
`copy` writes to a clipboard, and 12.7's argument is that a clipboard write has
nothing to say in JSON. Import has plenty to say, and the caller most likely to
want it is an agent doing a handoff.

### What a consumer would want

The preview and the write answer different questions and may want different
shapes. The preview is a report: what this export carries, and what adopting it
would change. The write is a mutation with a payload no existing envelope
carries, the map from each incoming ID to the ID this store minted, which is the
one thing a caller cannot reconstruct afterwards.

That map is the crux. Without it a caller that adopts five tickets has to guess
which new ticket came from which old one, and the `origin-ticket:` reference is
the only trace, which means parsing references back out to rebuild what the
command already knew.

### The decision this needs

Whether this is a new section 10 kind or a reuse of `mutation-result`.
`mutation-result` carries one ticket and a `pathsChanged`, and an adopt produces
N tickets and a mapping, so a reuse would be a stretch of the same sort `export`
already makes by publishing a null ticket. A new kind costs a row in 10, the
`kinds` list, and a test in `envelopekinds_test.go`.

Per 12.4 a published envelope is a surface, so this is settled with the user
before it ships rather than after.

## Acceptance criteria

- [ ] The kind question is settled: a new section 10 kind, or mutation-result reused, with the reason recorded
- [ ] The ID mapping is published in whatever shape is chosen, because it is the one thing a caller cannot reconstruct
- [ ] Preview and adopt both have a --json form, and the reconciliation notices are in it rather than on stderr alone
- [ ] Section 10 and the kinds list carry it, with a test in envelopekinds_test.go
- [ ] 12.8's Envelopes paragraph stops calling this a gap
