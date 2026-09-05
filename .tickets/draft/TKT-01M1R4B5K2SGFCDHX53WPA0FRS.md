---
schema: 1
id: TKT-01M1R4B5K2SGFCDHX53WPA0FRS
title: Decide how prefixed ID tracks subdivide a store
type: spike
status: draft
status_reason: null
priority: low
due_on: null
labels:
  - question
  - format
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-05T06:33:23Z
updated_at: 2026-09-05T19:50:24Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

A deferred design raised 2026-09-04, in conversation. The question: can a
store carry user-defined ID prefixes beside the `TKT-` default, so a large
project subdivides its tickets, and what does that cost?

Nothing structural requires `TKT-`. It is one constant, `IDPrefix` in
ticket/id.go, and the ULID behind it does all the identity work: uniqueness,
sort order, and collision-freedom between disconnected agents come from the
26 characters. The prefix exists so a reference is findable in prose, which
is why the corpus test scans with `TKT-[0-9A-Z]+`. So the format is free to
vary it.

What a prefix buys that a label does not: it travels with the reference. A
label is invisible at the point of mention, but the prefix sits inside the ID
wherever it is quoted, in a dependencies list, a commit message, a PR body.
`IDEA-01M1...` in a blocking list says at a glance that the wait is on
something undecided. It also gives per-track rules a routing key that
survives being quoted.

### The shape of the design

One store, not many. Per-component stores would need cross-store resolution,
would fragment the lock, and would blind `check`. Prefixes inside one store
keep one resolution universe and one lock.

Partition-first. The monorepo case, one prefix per subcomponent with `TKT-`
for the whole, is the driver. The idea-pipeline case rides along free once
the mechanism exists, because `draft`, a ticket type, and links already
express "not work yet" without a prefix. Design for the partition and let
the pipeline be a beneficiary.

Cross-track creation mints a new ticket on the receiving side. It is never a
move, so the originating ticket keeps its ID and every old back reference
keeps resolving: done and archived tickets remain in the store as files, so a
reference to a closed idea is never dangling. Closing the source is explicit,
a `--close-source` flag or a manual `status done`, never a side effect of
create, because an idea that fans out to three implementation tickets must
not close on the first. A plausible spelling is `git ticket create --from
IDEA-x`, copying title, description, and criteria, and linking both
directions.

Provenance wants its own field. `parent` is taken: a ticket born from
`IDEA-y` may also belong to epic `TKT-z`, and parent holds one value. A
dedicated `origin:` field that `check` verifies resolves, the way it
verifies `parent`, keeps both.

The schema/config split does real work here. The ID format, one or more
uppercase letters, a hyphen, a 26-character ULID, is schema, identical
everywhere. The set of prefixes a store uses is config, beside the label
allowlist, with `TKT` the default when nothing is declared. Per-track create
defaults and allowlists are a later config dimension, not part of the first
cut.

Resolution barely changes. A bare ULID fragment still resolves uniquely
across every track, since ULIDs cannot collide, so `show 01M1QK` keeps
working. A prefixed fragment splits at the first hyphen, and `minPrefixLen`
stays at four.

### The hard part

Compatibility is not purely additive. An old binary reading `IDEA-...` sees
an invalid ID, which is an error it reports, not an unknown field it
preserves. A store that opts in probably needs a schema bump so old readers
refuse cleanly with `schema_unsupported` instead of reporting every prefixed
ticket as corrupt. That question deserves its own section in plan 5.5, or a
new section, before any code, and it is the main cost of the feature.

### The name

"Prefix" describes the mechanism, not the meaning, so the concept needs a
name. Candidates, unsettled: track (the current lean: "the idea track", "the
tui track", parallel lanes with their own rules), stream (workstream is the
established PM word), series (bookkeeping flavor that fits a ledger), desk
(routing flavor, but implies a person behind it).

### The trigger

terva actually needs the subdivision, meaning its ticket volume makes one
undifferentiated `TKT-` sequence hard to work, or a second repository asks
for a partition. Whoever picks this up settles the name and writes the plan
section, compatibility first, before touching ticket/id.go.

## Notes

**agent:terva/mieli** at 2026-09-05T19:50:24Z

Two claims in the description above are false, demonstrated on
2026-09-05 against the binary at 0e9faf4. A third claim, that the
prefix is one constant, undercounts. The resolution question the
description never asks is settled at the end.

### The prefix is four sites, not one

`IDPrefix` is one constant, and `ValidID`, `NormalizeRef` and
`ResolveRef` in ticket/id.go all read it. Beyond those: the filename
fallback in `file.id()` at ticket/store.go:312 decides whether an
unparseable file is a ticket by testing the prefix, and
`shortestUnique` rebuilds a display ID as `IDPrefix + b[:n]` in two
places, cli/commands.go:96 and tui/view/list.go:462.

### "Resolution barely changes" is false

The method: a scratch store with two ordinary tickets, one of them
retrofitted by hand to `IDEA-01M1SHS5631486JEM819TNMKV4` in both the
filename and the frontmatter `id`, with a `depends_on` pointed at it
from the other.

`show 01M1SH`, a bare ULID fragment of the prefixed ticket, returned
the *other* ticket with exit 0. The control says what should have
happened: two ordinary tickets sharing eight leading characters answer
`ambiguous_id: "01M1SHTM" matches 2 tickets` with exit 1. So a query
that should have been ambiguous became a confident wrong answer.

The mechanism is `NormalizeRef`, which strips `TKT-` and nothing else.
A prefixed ID therefore never normalizes to a bare ULID, drops out of
fragment matching entirely, and is reachable only by its full literal
string. The claim that a bare ULID fragment still resolves uniquely
across every track is not true today and does not become true for
free.

`list` printed the same ticket as `TKT-IDEA-01M`, because
`shortestUnique` normalizes the ID, strips nothing, and pastes `TKT-`
back onto a string that already carries `IDEA-`. The listing invents an
ID that exists nowhere and resolves to nothing.

### The hard part is backwards

The description says an old binary reading `IDEA-...` "sees an invalid
ID, which is an error it reports". It does not. `files()` globs every
`.md` rather than filtering on the prefix, the ticket parses, and
`check --strict` on that store reported exactly one finding,
`dependency_missing` for the cross-reference, and said nothing about
the prefixed ticket itself. `show IDEA-<full ULID>` returned it with
exit 0, matching by full literal string through the fragment branch.

So an old binary does not refuse cleanly in one place. It accepts the
ticket, lists it under a fabricated abbreviation, resolves it only by
its full name, silently removes it from fragment resolution, and turns
every cross-track reference into `dependency_missing`. Four quiet
wrong answers instead of one loud refusal.

That inverts the argument for the schema bump. It is not a courtesy so
old readers refuse cleanly instead of reporting corruption. It is the
only thing standing between a prefixed store and wrong answers from
every binary already installed, and it has to land before any store
writes its first prefixed ID.

### The fork, settled

The prefix cannot be both decoration and identity. Either
`NormalizeRef` strips any `[A-Z]+-`, in which case bare ULIDs resolve
across tracks as the description wants, but `IDEA-<ulid>` and
`TKT-<ulid>` become one reference and a wrong prefix resolves happily.
Or the prefix is part of the identity, a prefixed reference must match
the ticket's own prefix, and a bare ULID still resolves across tracks.

Settled with the user on 2026-09-05: the prefix is part of the
identity. The argument for prefixes over labels is that they travel
with the reference, and decoration nothing checks does not travel, it
just rides along. The cost is that `ResolveRef` splits a reference and
compares two fields rather than one string, and `shortestUnique`
abbreviates within a track while staying resolvable, which is more
than pasting a constant back on.

### Two costs the description does not price

`origin:` is a new frontmatter scalar, and per 5.3 an absent scalar
renders as `null` rather than being omitted, so adding it means editing
every fixture that carries frontmatter. `status_reason` cost exactly
that.

The silent mis-resolution above is reachable today by anything that
writes a foreign ID into the store, which is what the Backlog.md
importer draft would do if it preserved source IDs. Whichever of the
two lands first pays for the schema bump.

The trigger has still not fired. Confirmed with the user on the same
day: this remains a design conversation, and nothing here promotes the
ticket.
