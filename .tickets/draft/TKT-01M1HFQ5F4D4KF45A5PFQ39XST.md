---
schema: 1
id: TKT-01M1HFQ5F4D4KF45A5PFQ39XST
title: Decide how git-ticket integrates with an external tracker
type: spike
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - format
  - question
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references:
  - ref: plan:deferred-questions
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-02T16:37:30Z
updated_at: 2026-09-03T16:53:44Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

A deferred question in plan section 15, now narrowed to its last third.

Two of the three original gaps are closed. `git ticket refs REF` is the lookup by ref, specified in plan section 8 and matched by the rule in 5.5: it finds a whole ref such as jira:PROJ-1234, or every ticket in a namespace with a bare jira:. Case is settled without a finding at all, because the namespace is compared without regard to case and the identifier exactly, which leaves the stored bytes alone. An untyped ref is now the section 11 warning `reference_untyped`.

What is left is `extensions`. 5.1 calls it the only place a consumer may write fields the core does not define, and it still has no mutation, so it round-trips through parse and render but can only be written by hand-editing the file.

The question is whether it gets one and what shape that takes. It stays deferred on purpose: there is no consumer until Phase 3, and Phase 3 is terva's to build, so a mutation designed now is shaped for a caller that does not exist. That is the same guess the namespace grammar would have been, and building the lookup first is what showed the grammar was not needed.

The trigger is a real consumer asking for it, which means terva's integration work naming the fields it wants to write, or a second consumer arriving with the same need. Until then a hand-edited `extensions` block round-trips correctly and nothing is lost but convenience.

## Notes

**agent:terva/mieli** at 2026-09-03T06:01:30Z

Groomed. This is the only draft in the set with no trigger to wait for, and its three gaps are present-tense. I checked all three against the current binary rather than rereading the ticket.

An untyped reference is accepted: `link ID --ref PROJ-1234` stores it and `check --strict` is green. So are `JIRA:proj-1234` and `jira:PROJ-1234` on the same ticket, as three unrelated references. That is what 5.1 calling a reference a typed stable identifier is worth today.

There is still no lookup by ref. `files PATH` covers the file: namespace and nothing covers the others, so which ticket is PROJ-1234 falls to `search`, which substring-matches title, description, notes and comments and so also finds a ticket that merely mentions the number.

`extensions` still has no mutation. It round-trips through parse and render, and hand-editing the file is the only way to write it, though 5.1 calls it the one place a consumer may write fields the core does not define.

Nothing here is blocked and nothing is waiting. What it wants is a decision on scope before code: whether this is validation alone, or validation plus a lookup, or the whole sync surface including a way to write extensions.

**agent:terva/mieli** at 2026-09-03T16:53:17Z

Scope settled and built: validation plus a lookup by ref. The extensions mutation stays out.

Shipped is `git ticket refs REF`, specified in plan section 8 and matched by the rule in 5.5, which finds a whole ref or a bare namespace, plus `reference_untyped`, a section 11 warning for a ref carrying no namespace.

The order was the decision, and it is the part worth keeping. Building the lookup first showed how little validation it actually needed. Case turned out to be a comparison rule rather than a stored-bytes rule, so it costs no finding at all: `refs` compares the namespace without regard to case and the identifier exactly, and the file on disk is untouched. What was left was the one structural claim 5.1 already makes, so the new code is a warning rather than a namespace grammar invented with no caller to test it against. A finding code is expensive to guess wrong, because `schema` publishes it and 12.4 covers it from that moment.

`refs` reuses the ticket-list envelope, so there is no new JSON kind and nothing to add to 10.1.

Neither half charged the corpus what I expected. An empty `references: []` raises nothing, so the 30 fixtures carrying one needed no edit; the cost was one new fixture store and its sidecar.
