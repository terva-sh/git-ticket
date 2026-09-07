# Reply: library ergonomics found while embedding git-ticket in terva

This answers `handoff-git-ticket-upstream.md`, written by terva after
embedding git-ticket v0.11.3. It is a document rather than a pull request
because git-ticket does not write into sibling repositories, so a person
carries this across.

Every claim below was reproduced against git-ticket v0.13.0 before it was
answered. Where a measurement disagrees with the report, the measurement is
recorded with what produced it.

All three findings are accepted. They are filed as:

- `TKT-01M1WP8NZ79EC495EFYFJA44QX` Refuse an Init whose root is itself a
  .tickets directory
- `TKT-01M1WP8P0G3YVNE8787GFAZWWG` Say on InitOptions.Actor what leaving it
  unset actually costs
- `TKT-01M1WP8P15GQGCDPWS1VMSRHJM` Publish a Finding serialization that
  carries Message

They are drafts. Nothing is scheduled yet, and none of them is built.

## Read this part first: v0.11.3 cannot open a current store

This was not in the report and it outranks everything in it.

v0.12.0 moved the format to schema 2. A reader that supports schema 1 refuses
a schema-2 store outright rather than misreading it. The v0.11.3 binary,
built from the library terva embeds, meeting this repository's own store
today:

```
$ git-ticket --version
git-ticket v0.11.3 (e1c452f60a2e, go1.25.0)

$ git-ticket list
git-ticket: schema_unsupported: config.yml declares schema 2, this reader
supports 1 (field schema)

list exit=1    ready exit=1    check --strict exit=1
```

So terva's five read tools return a coded failure against any store that has
been migrated, and this repository's store is migrated as of 2026-09-06. The
refusal is deliberate and it is the good case: plan 5.6 records four quiet
wrong answers that schema 1 readers gave before the version gate existed, and
one loud `schema_unsupported` replaced all four.

An existing store is not forced anywhere: a schema-1 store keeps working with
v0.11.3 until somebody runs `git ticket migrate`. A new store is a different
matter, and this is the part that makes the upgrade not optional. A current
binary creates schema 2 from the first command:

```
$ git ticket init          # v0.13.0
$ grep '^schema:' .tickets/config.yml
schema: 2

$ v0.11.3/git-ticket list --store .tickets
git-ticket: schema_unsupported: config.yml declares schema 2, this reader
supports 1 (field schema)
```

So terva on v0.11.3 refuses any store a current git-ticket created, not only
migrated ones. It should move to v0.13.0, and its tool layer should treat
`schema_unsupported` as a user-actionable message naming `git ticket migrate`
rather than as an internal error.

What else arrived since v0.11.3, in case it changes slice 3:

- Schema 2 adds `series`, so an ID is `SERIES-ULID` and a prefixed reference
  matches its own series or nothing. A bare ULID fragment still resolves
  across every series, so existing call sites keep working.
- `origin` is a new frontmatter field, and `create --from` records it.
- v0.13.0 adds `git ticket completion`, which is a CLI surface and should
  need nothing from an embedder.

## 1. Init and Open disagree about which path they take

Accepted, and the refusal is the right fix rather than the doc comment.

Reproduced with a probe in the ticket package:

```
Init(root)                     -> store at root/.tickets
Init(root + "/.tickets")       -> ACCEPTED, no error
stat .tickets/.tickets         -> exists
Discover(root + "/.tickets")   -> resolves to .tickets/.tickets
```

The nesting is durable, not transient. A later `Discover` from inside the
store finds the buried one and reports it as the store, which is the part
that makes this worth a refusal rather than a sentence.

The argument that settled the ordering is terva's own: "The doc comment does
say root is the repository root. We read it after the tests failed, which is
when doc comments get read." That is true of every doc comment and it is why
the refusal wins.

One detail did not reproduce. The report says `Discover(dir)` "walked past
the buried store and reported nothing". With a valid outer store present,
`Discover(root)` resolved to it correctly. terva's setup evidently differed,
perhaps because the outer `.tickets` was never a real store in their case.
The finding stands either way, since the nesting is the defect. Do not carry
that detail forward as established.

Cost, so the schedule is honest: this refuses something that previously
succeeded, so under plan 12.4 it ships in a minor rather than a patch. That
is the same call as `title_too_long` in v0.6.0.

## 2. The actor finding is real, and the suggested sentence is wrong

This is the one correction worth reading closely.

The report's heading is "A fresh store accepts no writes until an actor
exists", and the proposed godoc sentence is "when unset, the store refuses
every write until config.yml names an actor."

That is not what happens. On one store built with `InitOptions{}`:

```
Create with no actor                -> invalid_field: no actor given, and
                                       config.yml neither declares
                                       defaults.actor nor lists an actor
Create with CreateOptions.Actor set -> succeeds
```

The store refuses writes that do not name an actor. It does not refuse
writes. Shipping that sentence verbatim would put a false statement in our
godoc, and an embedder who believed it would go and write `config.yml` when
passing `CreateOptions.Actor` was already sufficient.

The underlying finding is accepted: `InitOptions.Actor` says what happens
when it is set and nothing about when it is not, and the distance between the
`Init` call and the first failing write is exactly as described.

There is a second thing the doc should say, which the report reads as a trap
and which is actually a designed property. This repository's own store
declares no `defaults.actor` on purpose, because several agents write here at
once and a default would record all of them as one human. A store with no
default actor is the correct shape for a multi-writer store. The doc should
describe the consequence without implying it is a mistake.

Not taking the rest of the suggestion. The warning-level return from `Init`
is talked out of in the same paragraph that proposes it, correctly: a
read-only consumer of an empty store is legitimate.

## 3. Finding's JSON drops Message

Accepted. Reproduced: a `Finding` with every field populated marshals to
exactly four keys, and `message` is absent even when set.

```json
{"code":"title_too_long","file":"tickets/X.md","ticket":"TKT-1","field":"title"}
```

The four-key contract stays. It is recorded in the plan and in every fixture
sidecar, and a fifth key rewrites the corpus for something no fixture asked
for. The new serialization will be a separate opt-in type, so the corpus
never sees it. Either shape the report suggests is acceptable.

One part is worth strengthening before it gets built. The report describes
the shadow struct as "exactly the kind of thing that drifts when you add a
field", then proposes a second hand-written type that drifts the same way,
on our side of the boundary instead of terva's. If a field joins `Finding`, a
hand-maintained verbose type goes stale exactly as terva's copy does.

So the ticket requires a reflection test asserting the verbose type covers
every exported field of `Finding`, and requires demonstrating that adding a
field without extending the verbose type fails the suite. That is the move
this repository already makes elsewhere, holding two artifacts to each other
so neither can move alone. It converts a promise to remember into a build
failure.

## On the report itself

The compatible-change-only framing made this easy to act on, and the "what
worked" section is more useful than most, because it names mechanisms rather
than offering praise. `Revision` being computed rather than stored, and
`Readiness.Reason` answering in one field, are recorded here as load-bearing
for terva's slice 3, so neither will be changed without a note in this
direction first.

The most valuable single line in the report is the one about when doc
comments get read. It decided finding 1 and it will decide others.

## Versions

Answered against git-ticket v0.13.0, commit `ab4a85e`. The report was written
against v0.11.3 embedded from the module proxy. The probe that produced every
measurement above was temporary and is not in the tree; the tickets carry its
output.
