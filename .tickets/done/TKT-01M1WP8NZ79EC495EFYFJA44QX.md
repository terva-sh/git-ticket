---
schema: 2
id: TKT-01M1WP8NZ79EC495EFYFJA44QX
title: Refuse an Init whose root is itself a .tickets directory
type: task
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
created_at: 2026-09-07T01:03:34Z
updated_at: 2026-09-07T01:33:28Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

`Open(path)` takes the store directory itself. `Init(root, opts)` takes the
repository root and creates `root/.tickets` under it. An embedder who read
`Open`'s signature first passes the store path to `Init` and gets a store at
`.tickets/.tickets`, with no error.

Reported by terva, which embedded git-ticket v0.11.3 and lost a debugging
round to it. See docs/reply-terva-library-ergonomics.md.

### Verified on v0.13.0, not inferred

A probe in the ticket package ran the mistake:

    Init(root)                     -> store at root/.tickets
    Init(root + "/.tickets")       -> ACCEPTED
    stat .tickets/.tickets         -> exists
    Discover(root + "/.tickets")   -> resolves to .tickets/.tickets

So the nesting is durable rather than transient. A later `Discover` from
inside the store finds the buried one and reports it as the store.

One detail from the report did not reproduce. terva wrote that `Discover`
"walked past the buried store and reported nothing". With a valid outer
store present, `Discover(root)` resolved to it correctly. Their setup
evidently differed, so do not repeat that part as established.

### The decision

Refuse, rather than document. This is terva's preference order and their
argument for it is the strongest line in their report: "The doc comment does
say root is the repository root. We read it after the tests failed, which is
when doc comments get read."

Nobody intends `.tickets/.tickets`, so the refusal has no false positives.

### What it costs

A new operational error code beside `store_not_found` and `store_exists`. It
is not a `check` finding, so plan section 11 and the fixture corpus are
untouched, which is the expensive part this avoids.

Under plan 12.4 it refuses something that previously succeeded, so it ships
in a minor rather than a patch. That is the `title_too_long` precedent from
v0.6.0. Confirmed with the user before filing.

## Acceptance criteria

- [x] Init refuses a root whose base name is .tickets, with a coded error that names the mistake
- [x] The new code joins the operational set in ticket/errors.go, and no plan section 11 code or fixture sidecar changes
- [x] A test runs terva's exact sequence: Init(root), then Init(root/.tickets), and asserts the refusal
- [x] The CLI path is checked too, so git ticket init inside a store does not create a nested one
- [x] It ships in a minor, per 12.4, because it refuses what previously succeeded

## Summary

Shipped in v0.14.0, from terva's first handoff.

`Init` refuses a root whose base name is `.tickets` with `invalid_root`, naming
the parent to pass instead. The code is `invalid_root` rather than
`nested_store`, because the plan already spends "nested store" on a store deeper
in the tree that discovery skips, and one phrase with two meanings is worse than
a longer name. Plan section 10 defines it.

`InitOptions.Actor` documents what leaving it unset costs: config.yml names no
default actor, so every write must supply one. terva's proposed wording said the
store refuses every write, which is false, and a store with no default actor is
the correct shape for several writers.

`FindingVerbose` and `ReportVerbose` carry `message` and `title` for a consumer
showing a report to somebody. Finding's four-key contract is untouched and no
fixture changed. A reflection test holds the verbose types to their originals,
and a second test proves that check can fail, because relocating terva's shadow
struct to this side of the boundary without a guard would only move the drift.
