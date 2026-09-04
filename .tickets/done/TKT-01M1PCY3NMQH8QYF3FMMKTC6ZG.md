---
schema: 1
id: TKT-01M1PCY3NMQH8QYF3FMMKTC6ZG
title: schema does not publish the label and milestone allowlists
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-04T14:25:04Z
updated_at: 2026-09-04T19:09:32Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

git ticket schema --json publishes thirteen keys: blocksOn, errorCodes, findingCodes, kind, kinds, openStatuses, priorities, schemaVersion, statuses, ticketSchema, transitions, types, unreadyReasons. Neither labels nor milestones is among them.

Both are enforced. A label outside the allowlist is label_unknown and a milestone outside it is milestone_unknown, so a write fails on a value the tool will not name. The only way to learn the legal set is to open .tickets/config.yml and read it.

Found while filing a ticket in this repository. A docs label seemed obvious, was rejected, and the allowlist had to be read out of the store file to discover that the sanctioned set is ci, claims, format, integration, mcp, policy, question and release.

The asymmetry is the argument. schema exists so a consumer never hard-codes a value this binary enforces, and 10.4 makes that its job. It already publishes the status enum for that reason, and unreadyReasons was added so nobody would hard-code eight values. Labels and milestones are enforced the same way and published nowhere.

One thing to settle. The other schema keys describe the binary and are identical in every store, while these two are per-store configuration. Putting them in the same envelope mixes those two things, so this ticket should decide whether they belong in the schema kind, in a separate config kind, or on a store envelope.

## Acceptance criteria

- [x] The legal labels and milestones are discoverable from a command rather than from config.yml
- [x] docs/plan.md section 10.4 records where they sit and why
- [x] The published envelope says whether an allowlist is enforced, so an empty one cannot be read as permitting nothing

## Notes

**agent:terva/mieli** at 2026-09-04T18:38:57Z

Acceptance criterion 3 stays unticked, because it rests on a premise that turned
out to be false and I would rather leave the record honest than tick it.

It asks that "a store with no allowlist is distinguishable from one with an
empty allowlist". Those are the same state. `permitted()` in `ticket/config.go`
treats a nil list and an empty list identically, and plan 4.1 gives an empty
allowlist the meaning "this store has not expressed an opinion" rather than
"this store permits nothing". A `config.yml` with `labels:` absent and one with
`labels: []` parse to the same value and behave identically, so there is nothing
to distinguish.

What the criterion was reaching for is delivered. The danger it names is real:
a consumer handed a bare `[]` reads it as "no label is permitted" when it means
"any label is permitted", and gets it exactly backwards without failing. The
envelope answers that with `enforced`, so `{"values": [], "enforced": false}`
cannot be misread. Section 10.6 records both the resolution and why the
distinction the criterion asked for does not exist.

Making the two states genuinely different would mean giving an empty list the
meaning "permit nothing" and inventing a separate way to say "no opinion". That
is a change to 4.1's semantics and to every store that already carries
`labels: []`, which is what `init` writes. It is not this ticket, and I do not
think it is worth doing.

**agent:terva/mieli** at 2026-09-04T19:09:17Z

Acceptance criterion 3 is reworded and now ticked. This supersedes the note
above, which said it would stay unticked.

It read: "A store with no allowlist is distinguishable from one with an empty
allowlist". It now reads: "The published envelope says whether an allowlist is
enforced, so an empty one cannot be read as permitting nothing".

The old wording asked for something that does not exist. `permitted()` in
`ticket/config.go` treats a nil list and an empty list identically, and plan 4.1
gives an empty allowlist the meaning "this store has not expressed an opinion",
so the two states the criterion wanted told apart are one state. No
implementation could have satisfied it.

The new wording is the danger the old one was pointing at, which did get built.
A consumer handed a bare `[]` reads it as "no label is permitted" when it means
"any label is permitted", and gets it backwards without failing. `enforced`
answers that, and `TestConfigSaysAnEmptyAllowlistPermitsEverything` holds both
output forms to it.

The old text is kept verbatim in the note above rather than only in Git, so the
record of what was asked for survives the edit. Rewording a criterion to match
what shipped is worth doing only when the criterion was unsatisfiable as
written, which this one was. A criterion that merely turned out to be difficult
should stay as it is and go unticked.

## Summary

Shipped as `git ticket config`, a new command and a ninth JSON kind.

The allowlists did not go into `schema`. Plan 10.4 closes with "This command
reads no store. It answers outside a repository and before `init`", which is the
property that lets a consumer ask what is legal before it has anything to ask
about. Labels and milestones are per-store, so putting them there would make the
same command answer differently depending on where it ran, and would spend that
guarantee. The split is now the design: `schema` is what the binary enforces and
is identical everywhere, `config` is what one store chose. Section 10.6 records
it and 10.4 points at it.

`config` publishes the label and milestone allowlists, the actors, the `create`
defaults, and the lock timeout. Each was per-store and reachable only by opening
`config.yml`, which is the complaint this ticket was filed about.

An allowlist is an object, `{"values": [...], "enforced": true}`, not a bare
list. Per 4.1 an empty allowlist permits everything, so a consumer handed `[]`
and reading it the obvious way concludes that nothing is permitted, which is
exactly backwards and fails silently. `enforced` is derived from the length, so
the two cannot drift, for the reason `openStatuses` is derived from `statuses`.

Two things the ticket had wrong, both worth recording.

It said "a write fails on a value the tool will not name". It does not.
`KnownLabel` and `KnownMilestone` are consulted only by `check`, so the write
succeeds and `label_unknown` is a warning. I confirmed it: creating a ticket
with an unlisted label exits 0. The rejection is real but it happens in
`check --strict`, which is the CI gate. The allowlist stays advisory here;
changing that is a separate decision and this ticket only publishes.

Acceptance criterion 3 was unsatisfiable as written and has been reworded. It
asked that a store with no allowlist be distinguishable from one with an empty
allowlist, and those are one state: `permitted()` treats nil and empty
identically, and 4.1 gives an empty list the meaning "has not expressed an
opinion". It now asks that the envelope say whether an allowlist is enforced,
which is the danger the original was pointing at and which did get built. The
notes above carry the original text and the reasoning.

The lock timeout publishes the effective value rather than the configured one.
The store falls back to `DefaultLockTimeout` when `config.yml` is silent, so the
raw zero would have reported `0s` for a store that waits ten seconds. What is
published is what is enforced, which is the rule the rest of the envelope
follows.

`TestConfigAgreesWithWhatCheckEnforces` is the test that makes publishing worth
anything: a label `config` calls legal must not raise `label_unknown`, one
outside the set must, and with `enforced` false nothing may. A published set
that `check` disagreed with would be worse than none, because a consumer would
trust it.
