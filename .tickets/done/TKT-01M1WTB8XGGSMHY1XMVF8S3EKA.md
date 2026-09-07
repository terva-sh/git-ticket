---
schema: 2
id: TKT-01M1WTB8XGGSMHY1XMVF8S3EKA
title: Name upgrading the binary when schema_unsupported refuses
type: bug
status: done
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
created_at: 2026-09-07T02:14:53Z
updated_at: 2026-09-07T02:22:18Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

terva's third handoff, finding 2. Measured against v0.14.0 with a real v0.13.0
binary and a freshly migrated schema 3 store.

`schema_unsupported` says "config.yml declares schema 3, this reader supports
2". Every fact in that line is true, and the reader still concludes their store
is broken and reaches for `check --fix`, which cannot help. A store declaring a
level above the reader is never a store problem, and the remedy is always the
same: upgrade the binary.

Two sites raise the message, not the one terva measured. `ticket/config.go`
raises it for `config.yml`, and `ticket/parse.go` raises it for a ticket file.
Both have the gap and both should say the same thing.

The direction is why this cannot copy its neighbours. `ticket/series.go` and
`ticket/apply.go` both refuse on schema and both say "run git ticket migrate
first", because there the store is behind the reader. `schema_unsupported`
fires when the store is ahead, where migrate and `check --fix` are equally
useless. So the two families of schema refusal need opposite remedies, and
that asymmetry is what produces the wrong guess terva reported.

Plan section 4 already asks this message to "name the version needed". It names
the schema levels and stops before saying how to get a reader that supports
them, so this is closer to a compliance gap than to a new feature.

The gate is one-directional by design and terva is not asking that to change.
A newer binary reads an older store, and this repository is the proof, running
v0.14.0 against its own schema 2 store with `check --strict` green. Since
`init` writes the newest level from its first command, one person on a current
binary can create a store that everyone behind them refuses, without migrating
anything or intending anything. That is what makes this message one that gets
met often, by people who did nothing wrong.

## Acceptance criteria

- [x] The config.yml refusal names upgrading git-ticket as the remedy
- [x] The ticket-file refusal in parse.go names it the same way
- [x] Both messages still name the declared level and the level the reader supports
- [x] Neither message suggests migrate or check --fix, which cannot clear this condition
- [x] A test pins the remedy in both messages

## Notes

**agent:terva/mieli** at 2026-09-07T02:22:18Z

Built and shipped in v0.14.1. All five criteria tick.

Two sites carried the gap rather than the one terva measured. `ticket/config.go`
raises the code for `config.yml`, which is what they hit, and `ticket/parse.go`
raises it for a ticket file. Both now end "; upgrade git-ticket to read it".

The remedy is deliberately the opposite of its neighbours, and that asymmetry
is the whole finding. `ticket/series.go` and `ticket/apply.go` both refuse on
schema and both say to run migrate, because there the store is behind the
reader. This code fires when the store is ahead, where migrate and `check
--fix` are equally useless. That is the wrong turn terva reported a reader
taking against a sentence every word of which was true.

The test forbids "migrate" and "check --fix" rather than only requiring the
right words, because the failure being prevented is a plausible wrong remedy
rather than a missing one.

No plan change was needed. Section 4 already asks this message to name the
version needed, and it named the schema levels while stopping short of saying
how to get a reader that supports them, so this is closer to a compliance gap
than to a new feature.

## Summary

Shipped in v0.14.1. Both sites that raise `schema_unsupported`, `config.yml` in
ticket/config.go and a ticket file in ticket/parse.go, now name upgrading the
binary as the remedy, and neither mentions migrate or `check --fix`, which
cannot clear a store that is ahead of its reader.

Verified from the released binary shape rather than the test seam alone: a
store one level ahead refuses `list`, `ready` and `check --strict`, each exit 1
and each carrying the remedy, and the JSON envelope carries the same message
under the unchanged `schema_unsupported` code.
