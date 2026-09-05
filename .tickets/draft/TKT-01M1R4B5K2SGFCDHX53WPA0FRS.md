---
schema: 1
id: TKT-01M1R4B5K2SGFCDHX53WPA0FRS
title: Decide how prefixed ID series subdivide a store
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
updated_at: 2026-09-05T20:19:20Z
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

**agent:terva/mieli** at 2026-09-05T20:11:57Z

The name is settled: series. Decided with the user on 2026-09-05,
after a test rather than a lean. This supersedes the "The name" section
of the description above, which named track as the current lean. That
lean is withdrawn.

### The method

Two passes. First, a case-insensitive whole-word grep for each
candidate across the Go and Markdown files, excluding .tickets/, to
find what each word already means in this project. Second, writing each
candidate out in every register it would appear in: the config key, the
create flag, the error message, and the prose compounds.

### What the grep found

`stream` is taken. Eleven live uses, every one of them meaning I/O: the
streams on `Env`, stdin in cli/prose.go, the byte stream in
tui/input.go, and the escape-sequence stream in cli/ui.go. The stdio
adapter draft would add more. Disqualified outright.

`track` collides in a subtler way that the description's lean did not
price. Zero uses as a noun, seventeen as a verb, and the two worst
sit where a reader cannot miss them: docs/plan.md line 1 is
"repository-native work tracking for agents", and the `--cross-branch`
help says "remote-tracking refs", which is git's own term. So
`tracks: [TKT, IDEA]` would land in a document that says "a work ledger
that lives in the repository it tracks" on its first page, where the
natural reading of `tracks:` is "the things this store tracks", which
is every ticket.

`series`, `lane` and `docket` are unused. `desk` appears once.

### Why series

An invoice or cheque series is a numbered run distinguished by a
prefix. That is not an analogy for the mechanism, it is the mechanism,
and it puts the name in the register this project already occupies,
which is a ledger.

It also names the half being built first. The description defers
per-track create defaults and allowlists to "a later config dimension,
not part of the first cut", so the first cut is purely ID
partitioning. Track names the deferred half, the parallel lanes with
their own rules. Series names the half that ships.

The known weakness, recorded rather than argued away: series reads the
same singular and plural, so the config key `series:` does not announce
that it holds a list the way `tracks:` would. And the word hints at an
ordinal position that the ULID does not supply. Neither was judged to
outweigh a clean namespace.

### Why not the others

`desk` implies a person or a team behind the partition, but the driver
is subdividing a monorepo by component, and `assignees` already carries
the person. `lane` is free today but a board view is a plausible future
here, and a board's swimlanes group by status or assignee, so the
collision would arrive with the feature that most wants the word.

### The vocabulary this settles

The config key is `series`, holding the list of prefixes a store uses,
beside the label and milestone allowlists. The create flag is
`--series`. In prose: "the idea series", "a cross-series reference",
"per-series create defaults", "series-aware abbreviation".

What this does not settle is the error spelling. The templates work set
a precedent that a name resolving to nothing refuses with
`invalid_field` naming the directory rather than earning a new code, so
a missing series may well do the same. A new error code is a surface
plan 12.4 covers and needs its own decision.

### One consequence applied

This ticket's title said "prefixed ID tracks", which the settled name
makes wrong. It is now "Decide how prefixed ID series subdivide a
store". The description's own prose still says track throughout and is
left as it stands, because it is the record of what was thought at the
time and this note is the correction.

**agent:terva/mieli** at 2026-09-05T20:19:20Z

The site count above is now wrong, and my own merge is what made
it wrong. PR #135 landed on 2026-09-05, hours after the note that
wrote it, and collapsed the two copies of `shortestUnique` into one
exported `ticket.ShortestUnique` in ticket/id.go.

### The prefix is three sites, not four

`IDPrefix` is still one constant. `ValidID`, `NormalizeRef` and
`ResolveRef` in ticket/id.go still read it, and the filename fallback
in `file.id()` still tests it to decide whether an unparseable file is
a ticket. The third site is now `ShortestUnique` alone, at
ticket/id.go:168, where cli/commands.go:96 and tui/view/list.go:462
each had their own copy before.

The line numbers in the section above are dead. cli/commands.go:70 is
`storeAbbreviations`, which collects the IDs of the whole store and
hands them to the library at line 89, and tui/view/list.go:434 is
`abbreviateIDs`, which maps tickets to IDs and calls the same
function.

### The two copies had already begun to drift

Worth recording, because it is the argument this ticket will need if
anyone asks whether the centralization was necessary. The copies were
not identical text. `diff` between them at ca6d57c^ shows four
differences: the CLI took `ids []string` and the TUI took
`tickets []*ticket.Ticket`, the TUI declared `abbrevLen` inside the
function where the CLI had it at package scope, the shared comment
was wrapped to different widths, and the loops named their variables
differently.

None of that changes what the rule computes, and the live-store diff
across three views confirmed no output moved. But two copies of one
rule that already disagree about their own signature are two copies
that will eventually disagree about the rule, and prefix-aware
abbreviation is exactly the change that would have done it.

### What that changes for the work, and what it does not

The cost drops by one site and the shape does not change. Every claim
this ticket makes about behaviour still holds: `ShortestUnique`
normalizes, strips nothing, and pastes `TKT-` back onto a string that
may already carry a prefix, so `list` would still print
`TKT-IDEA-01M`.

What improves is where the fix goes. The abbreviation half of this
work is now one function beside `ResolveRef`, which is the function it
has to agree with, and `TestShortestUniqueRoundTrips` already holds
the two together. A prefix-aware abbreviation and a prefix-aware
resolution can no longer be built in different files by different
sessions and be found to disagree later.

The trigger has still not fired. This note corrects a survey, it does
not start the work.
