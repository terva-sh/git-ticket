---
schema: 2
id: TKT-01M1WP8P15GQGCDPWS1VMSRHJM
title: Publish a Finding serialization that carries Message
type: task
status: draft
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
created_at: 2026-09-07T01:03:34Z
updated_at: 2026-09-07T01:03:56Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

`Finding.MarshalJSON` emits exactly `code`, `file`, `ticket`, `field`. The
reasoning is on the type: the recorded contract is four keys, every fixture
sidecar records them, and adding a fifth rewrites the corpus. `Message` and
`Title` carry `json:"-"` for that reason.

The reasoning holds for the corpus. It does not reach an embedder whose
consumer is a language model reading a tool result, which wants the human
sentence beside the code. terva now maintains a parallel five-field struct
and copies every finding into it.

Reported by terva. See docs/reply-terva-library-ergonomics.md.

### Verified on v0.13.0

A `Finding` with every field populated marshals to four keys:

    {"code":"title_too_long","file":"tickets/X.md","ticket":"TKT-1","field":"title"}

`message` is absent even when set, and a `Report` carries the same four
keys per finding.

### What to build

An explicit second serialization the embedder opts into, so the four-key
contract stays untouched and the corpus never sees the new type. terva
suggests `Report.Verbose()` returning a type whose JSON carries `message`,
or an exported `FindingVerbose` conversion. Either shape is fine.

### The part worth strengthening

terva frames their shadow struct as "exactly the kind of thing that drifts
when you add a field", and then proposes a second hand-written type that
drifts the same way, on our side of the boundary instead of theirs. If a
field joins `Finding`, a hand-maintained `FindingVerbose` goes stale exactly
as their copy does.

So pair it with a test that asserts the verbose type covers every exported
field of `Finding`, by reflection. That is the move
`TestNoticeAgreesWithTheHeaders` and the corpus tests already make in this
repository: two artifacts held to each other so neither can move alone. It
turns a promise to remember into a build failure.

### Scope

Additive. No existing key changes, no fixture changes, and nothing in plan
section 11 moves. A new exported type is a new surface under 12.4, so it
ships in a minor.

## Acceptance criteria

- [ ] An exported verbose serialization carries message beside the four contract keys
- [ ] Finding.MarshalJSON still emits exactly code, file, ticket, field, and no fixture sidecar changes
- [ ] A reflection test asserts the verbose type covers every exported field of Finding
- [ ] Adding a field to Finding without extending the verbose type fails the suite, demonstrated by trying it
