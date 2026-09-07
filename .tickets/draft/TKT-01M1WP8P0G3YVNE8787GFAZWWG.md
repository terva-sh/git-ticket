---
schema: 2
id: TKT-01M1WP8P0G3YVNE8787GFAZWWG
title: Say on InitOptions.Actor what leaving it unset actually costs
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

`Init(dir, InitOptions{})` succeeds and records no actor in `config.yml`. A
later `Create` that names no actor fails with `invalid_field: no actor given,
and config.yml neither declares defaults.actor nor lists an actor`. The
mistake is in the `Init` call and the failure arrives at the first write,
which in a test helper is a different function in a different file.

`InitOptions.Actor` says only "Actor is recorded in config.yml as the first
known actor, when set." It does not say what happens when it is not.

Reported by terva. See docs/reply-terva-library-ergonomics.md.

### The report overstates it, and the correction is the point

terva's heading reads "A fresh store accepts no writes until an actor
exists", and their proposed sentence for the godoc is "when unset, the store
refuses every write until config.yml names an actor."

That is false, and shipping it verbatim would put a wrong statement in our
godoc. Measured on one store built with `InitOptions{}`:

    Create with no actor                -> invalid_field
    Create with CreateOptions.Actor set -> succeeds

The store refuses writes that do not name an actor. It does not refuse
writes. An embedder who believed their sentence would go and write
`config.yml` when passing `CreateOptions.Actor` was already sufficient.

### The second thing the doc should say

This is a designed property, not an oversight. This repository's own store
deliberately declares no `defaults.actor`, per AGENTS.md, because several
agents write here at once and a default would record all of them as one
human. A store with no default actor is the right shape for a multi-writer
store, so the doc should describe the consequence without implying it is a
mistake to be avoided.

### Not taking the rest

terva also floats a warning-level return or a documented second result from
`Init`. They talk themselves out of it in the same paragraph, correctly: a
read-only consumer of an empty store is legitimate. One sentence at the
decision point is the whole fix.

## Acceptance criteria

- [ ] InitOptions.Actor says that leaving it unset means config.yml names no default actor
- [ ] It says every write must then supply an actor explicitly, and does not claim the store refuses every write
- [ ] It records that a store with no default actor is a legitimate shape for a multi-writer store
- [ ] Nothing behavioural changes, so the suite passes unmodified
