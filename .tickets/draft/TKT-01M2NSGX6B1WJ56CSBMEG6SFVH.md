---
schema: 2
id: TKT-01M2NSGX6B1WJ56CSBMEG6SFVH
title: Reconcile the label separator two sister stores chose differently
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - area/format
  - question
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-16T19:01:30Z
updated_at: 2026-09-16T19:01:30Z
created_by:
  id: agent:claude/t3code
  name: ""
updated_by:
  id: agent:claude/t3code
  name: ""
extensions: {}
---

## Description

Two stores that share this tool coined the same three label dimensions a day
apart and spelled the separator differently. Nothing reconciles them, and
git-ticket is where a convention would be published from.

### What each did

terva, on 2026-09-16, migrated 21 flat labels across 123 open tickets to
`area/`, `scope/` and `init/`, with bare labels kept for conditions on the work
such as `live-test` and `flake`. Its `.tickets/config.yml` sets `labels: []`, so
nothing is enforced, and a hand-written `.tickets/CONVENTIONS.md` carries the
vocabulary instead.

terva-ext-web, adopting the tool at 26 tickets on the same day, enforces an
allowlist of 16 and spells them `area:cache`, `scope:core` and
`initiative:terva-modernization`.

This store adopted `area/` on 2026-09-16 and carries `area/` only.

### Why it is worth a ticket rather than a shrug

`--label area/core` and `--label area:core` do not interoperate. Anything that
reads labels across stores, and the interchange in plan 12.8 is exactly that,
meets both spellings. An export adopted from one store into the other carries
labels the receiver's allowlist rejects, in a form no rename rule can guess,
because the two also disagree on vocabulary: terva's scope is
`contained`/`crossing`/`structural` and terva-ext-web's is `core`/`follow-up`.

terva's commit records that both separators were tested before it chose:
`area/core` and `scope:small` each survive a write, a `--label` filter, and
`check --strict`. So this is not a capability difference. It is a coin-flip that
landed twice.

### What this store can and cannot decide

It cannot pick for anybody. The label vocabulary is a store's own and the
allowlist is advisory by design, per plan 4.1, and a tool that imposed a
taxonomy would be doing the thing `config.yml` exists to prevent.

What it can do is publish a recommended spelling and say why, so a third adopter
does not coin a fourth. That is a documentation decision with a real cost
attached, which is why it is filed rather than assumed.

### Worth considering, not prescribed

- Whether `/` or `:` is recommended, and on what grounds beyond precedence.
  Both work today, so the argument has to be about reading, shell quoting, or
  what a future `--label area/*` glob would need.
- Whether `import` should be able to rewrite labels on adoption, which is the
  mechanical half and may be the more useful one. `--adopt` already drops labels
  outside the receiver's allowlist, so the hook exists.
- Whether any of this belongs in the document
  TKT-01M2NQVK39AD3TYKRPSD13430C proposes, rather than in prose here.

## Acceptance criteria

- [ ] A recommended separator is chosen and the reason is recorded, or the question is closed as a store's own to answer
- [ ] Whether import can rewrite labels on adoption is decided, since that is the mechanical half
- [ ] The decision is published where a third adopter meets it before coining a fourth spelling
