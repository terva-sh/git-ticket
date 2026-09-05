# git-ticket: repository-native work tracking for agents

> Status: **design locked, ready to build.** This document supersedes the terva
> proposal now archived at `docs/proposals/archive/git-ticket.md`. Every
> question that blocked the
> Phase 1 contract is answered here: the staleness token is a content hash, the
> claim is orthogonal to status, archive moves the file with the status as the
> authority, and unknown frontmatter fields are a `check` error rather than a
> read error. What remains open is listed in section 15 and none of it blocks
> Phase 0.
>
> Repository: `github.com/terva-sh/git-ticket` (public release mirror).
> Go module path: `github.com/terva-sh/git-ticket`. Go 1.22+.

## 1. What this is

A work ledger that lives in the repository it tracks. Tickets are Markdown files
with YAML frontmatter, committed alongside the code, readable in any editor and
reviewable in `git diff`. One Go library owns the format. A `git-ticket` binary
exposes it as `git ticket …`. Terva consumes the same library, and so can
anything else.

The problem it solves is that a project outlives an agent session. Claude Code,
Codex, terva, a shell script, and a human editor may each touch one ticket
during its life. All of them should be operating on the same files, and none of
them should have to parse another tool's terminal output to do it.

Three properties decide the design:

1. The file is the source of truth. Any index or cache is disposable and must be
   rebuildable from the files alone.
2. Hand-editing is supported, not tolerated. A person opening a ticket in vim
   and changing a line is a first-class operation, and `check` is the safety net
   that catches damage. This is the reason to accept Markdown's merge behaviour
   instead of a database.
3. The tool never claims coordination it does not have. A claim recorded in an
   unpushed clone is local evidence, not a reservation.

### Scope of this document

Phases 0, 1, 2, and 4 below: the format, the core library, the standalone CLI,
and the optional stdio adapter. Terva's integration is Phase 3 and belongs to
terva, because it is committed work for terva rather than for this module.

This repository does not write into terva's. What it produces instead is
`docs/handoff-terva-phase-3.md`, a document for a terva agent to work from.
See the Policy section of `AGENTS.md` for why.

## 2. Naming

The command is singular. Git's own subcommands are singular even when they
manage a collection: `git remote`, `git branch`, `git tag`, `git stash`, `git
worktree`, `git submodule`, `git config`. Only `git notes` breaks the pattern.
Each operation here acts on one ticket or queries the set, so `git ticket show`,
`git ticket claim`, and `git ticket list` all read correctly.

Directories stay plural, because they hold many things: the store is `.tickets/`
and the tickets live in `.tickets/tickets/`. The asymmetry is deliberate. Do not
"fix" it in either direction.

Terva follows the same rule with a `terva ticket` subcommand and `ticket_*`
tools, matching the `task_*` tools that the existing `terva-tasks` extension
already ships.

## 3. Decisions locked

| Decision | Resolution |
|---|---|
| Store location | One `.tickets/` per Git root. An explicit override supports odd layouts. V1 does not discover nested stores. |
| Binary | `git-ticket`, invoked as `git ticket …`. A bare `ticket` alias is deferred. |
| Staleness token | An opaque `revision`, the SHA-256 of the ticket file bytes. Not stored in the file. |
| Precondition strictness | Optional in the library and CLI, where omitting it means last write wins under the local lock. Required in terva's tool schema. |
| Claim | Orthogonal metadata, not a status. Advisory, never exclusive. |
| Archive | The `archived` status is authoritative. The file move to `.tickets/archive/` is a mechanical consequence. |
| Unknown top-level fields | `check` errors. Ordinary reads warn and preserve them on write. Hard rejection waits for a schema major bump. |
| Statuses | Fixed set of seven in v1. Projects use labels for local categories. |
| History | Git records file provenance. Ticket fields record semantic activity. No separate event log in v1. |
| Remote operations | None. No fetch, push, merge, branch switch, or commit as a side effect of a mutation. |
| Ownership | Independent repository and Go module. Library and CLI first, terva second. |

## 4. Store layout

```text
.tickets/
├── config.yml
├── README.md
├── draft/
│   └── TKT-01K3ZYD1P0F3XKQVWM8XW8SBQ2.md
├── tickets/
│   ├── TKT-01K3ZYEE00HV9ZDBB8BEASXBBG.md
│   └── TKT-01K3ZYG8K0Y52AD43XRGM4T7WZ.md
├── done/
│   └── TKT-01K3ZYH7V0MW2AQ5H1J8YQK4TC.md
└── archive/
    └── TKT-01K3ZYJ360Q7ESC30QAD2SMY0H.md
```

A ticket's directory follows its status:

| Status | Directory |
|---|---|
| `draft` | `draft/` |
| `ready`, `in-progress`, `blocked`, `review` | `tickets/` |
| `done` | `done/` |
| `archived` | `archive/` |

The layout is the format. It is not configurable and not opt-in, because a
layout a store can decline is one no reader can count on.

A directory is a ticket store when it holds `config.yml`. `Open` returns
`store_not_found` for a directory without one, naming the file it wanted.

Existence used to be the whole test. `Open` stat'd the path, and a missing
`config.yml` fell through to the defaults in 4.1, so any directory that existed
opened as a store holding no tickets. `--store` at a typo answered `ticket-list`
with an empty array and exit 0, and `ready` reported that nothing was ready to
pick up. That is the one wrong answer indistinguishable from a right one: an
agent that lists and sees nothing concludes there is no work, rather than that
it is reading the wrong directory.

No directory is part of the test, and `tickets/` is the one that looked like it
should be. Git tracks no empty directory. A store whose open work is finished
has an empty `tickets/`, which the next commit does not record and the next
clone does not create, so requiring it would reject a store for the ordinary
achievement of having nothing in progress. A freshly initialized store commits
three files and no directories at all. A marker that disappears when a store
runs out of work is not a marker.

The name is not the test either. A store lives at `.tickets/`, and `Discover`
looks for that name walking up, but `--store` and `GIT_TICKET_STORE` name a
directory outright and the corpus points them at fixture stores called `store/`.
Requiring the name would reject every fixture. They are named that way because
`go:embed` skips any path beginning with a dot, so the realistic name would
embed nothing.

That leaves `config.yml`, which is a file, so Git tracks it and a clone has it.
`init` writes it, and 4.1 gives it content no other file carries. The cost is
that deleting it stops the store opening rather than falling back to the
defaults in 4.1. That is the right trade: deleting the config is a deliberate
act somebody can undo, where mistyping a path is an accident that used to
return a plausible answer.

The point is a person, not the tool. The tool is happy either way and `list`
already answers with open work, per section 8. Somebody reading the store in a
forge web UI, or through `ls`, has a directory as the only query they get for
free, and `tickets/` holding the working set alone is what makes that answer
worth having. Before this, `.tickets/tickets/` held 43 files of which a handful
were live.

The pipeline runs one way and a ticket moves at most three times. It is filed
into `draft/`. When it is written well enough for work or planning to start,
somebody moves it to `tickets/`, which is the same promotion that the agent
workflow block reserves for a person. Finishing it moves it to `done/`, which
holds recent work still worth reading. A person sweeps `done/` into `archive/`
periodically with `git ticket archive`, and no command does that in bulk.

The working statuses share one directory on purpose. A directory each would turn
every ordinary transition into a rename, and `ready`, `in-progress`, `blocked`,
and `review` are exactly the statuses a ticket churns through while somebody
works it. The three boundaries that do get a directory are the ones that change
whether a ticket is worth looking at.

A status this document has not defined yet lands in `tickets/`. That is the same
exclusion shape section 8 uses for the open set: name the special cases, and let
anything new take the ordinary path rather than falling out of the store.

Status is the only thing a directory keys on. A path is one dimension, so any
partition spends its single slot once, and status has the better claim because
done is the property that makes a file uninteresting. Type does not get
directories: `update --type` is a frontmatter edit today and would become a file
move, landing on the promotion of a task to an epic, which is the busiest moment
in a ticket's life. An epic is also not a boring file, so pulling epics out of
`tickets/` would remove the most interesting rows from the view this section
exists to make readable.

The status wins when it disagrees with the directory, per 6.3. A file in the
wrong place is read correctly and reported as `location_mismatch`, and
`check --fix` moves it. That is what migrates an existing store: one pass, one
rename-only commit, no schema bump, because a ticket file does not change when
its path does.

The cost is that a status change crossing a boundary is now a rename, so two
agents moving one ticket to different statuses collide as a rename conflict
rather than as a content conflict on `status:`. That collision already existed
and this makes it uglier to resolve. The pull request workflow is the mitigation,
and it is the same one that already covers two agents editing one ticket.

Discovery walks up from the current directory to the Git root and looks for
`.tickets/`. A `--store` flag or `GIT_TICKET_STORE` overrides it, and an
override may point outside a repository, in which case the worktree-aware lock
degrades as described in section 7.2.

Filenames are the ID and nothing else. Backlog.md puts the title in the
filename; this format does not, because a title change would rename the file,
break `git log` on the old path, and make ID-to-path resolution depend on
mutable data.

`README.md` is generated at `init` and explains the store to a human who finds
it in a diff. It is not read by the tool.

`epics.md` is generated too, and it is the one view a directory cannot give.
Directories key on status and this section refuses to key them on type, so
`list --type epic` has no equivalent in a file browser. The index is that
equivalent: a Markdown table of the epics in the store, written for a person and
never read by the tool.

A row holds the ID, the title, the status, and a link to the file, and nothing
derived from other tickets. A row carrying child counts or blocking state would
go stale whenever any ticket in the store changed status, so CI would go red on a
commit that touched no epic. Minimal rows are not a simplification here. They are
what keeps the staleness surface proportional to the thing being indexed.

Which epics appear is an exclusion: `done` and `archived` are left out, and
everything else appears. An inclusion list would silently drop a status added
later. Drafts appearing is also right on its own terms, because a draft epic is a
decomposition somebody is writing, which is what a person browsing a forge most
wants to see, and the status column carries the distinction for anyone who cares.

`check` reports a stale index as `epics_index_stale` and `check --fix`
regenerates it. No mutation writes it. Regenerating on every write to an epic
would make a hot file that two agents editing unrelated epics collide on, which
is the collision `blocks_on` exists to avoid. The window in which the file can be
wrong ends at the next `check --strict`.

A merge conflict in a generated file also resolves mechanically: take either
side, run `check --fix`, commit. That is worth saying, because a shared generated
file looks like a bad idea until you notice its conflicts are free.

Neither generated file is read by the tool, and neither counts toward what makes
a directory a store. A store whose `epics.md` was deleted has a stale index, not
a directory that stopped being a store.

### 4.1 config.yml

```yaml
schema: 1
actors:
  - id: human:sothr
    name: Drew Short
labels:
  - auth
  - docs
milestones:
  - v1.1
  - v1.2
defaults:
  type: task
  priority: normal
  actor: null
  claim_expiry: null
lock:
  timeout: 10s
```

`actors` names who writes to this store, and every write resolves one of four
ways. An explicit `--actor` wins. Failing that, `defaults.actor` when it names
one. Failing that, the first entry in `actors`. A store with none of the three
refuses the write, as `invalid_field` naming both ways to supply one.

The third branch is the one that goes wrong quietly. Nothing chose that actor,
it is merely first, so an agent that omits `--actor` files its work under
whoever heads the roster and the record then says a person did it. The CLI warns
on stderr when it happens, per section 10.

`defaults.actor` is how a store turns that warning off. Declaring one says the
choice is deliberate, which is what somebody working a store alone wants after
seeing the warning once. Leaving it null keeps the warning, which is what a
store several agents write to wants. `init` renders the field as null rather
than omitting it, so the opt-out is visible in a fresh config instead of being
something to find in this document.

The warning belongs to the CLI rather than to `check`, and that is not an
accident. Resolution happens at the write and leaves no trace, so once the file
names the actor a fallback is byte for byte identical to a deliberate `--actor`
naming the same one. Only the writer knows, and only while it is writing.

`labels` and `milestones` are advisory allowlists: `check` warns about a value
outside one and never errors. Configuration sets defaults and vocabulary. It
cannot add a status, change a transition rule, or grant a consumer authority it
does not otherwise have.

An empty list is not an empty vocabulary. It is a store that has not expressed
an opinion, so it permits everything. A store opts in by listing.

Advisory is the right strength for both. Either is how somebody marks work
before the vocabulary catches up, and erroring would mean committing a config
edit before a ticket could name a new label or a new release even once. The
warning still earns its place: `milestone` is a bare scalar with no registry, so
nothing else can tell `v1.2` from `v1.2.0`, and a store left alone accumulates
near-duplicates until `list --milestone` quietly answers about the wrong one.

## 5. Ticket format

### 5.1 Frontmatter

```yaml
schema: 1
id: TKT-01K3ZYEE00HV9ZDBB8BEASXBBG
title: Add token refresh handling
type: task
status: ready
status_reason: null
priority: high
due_on: "2026-10-14"
labels:
  - auth
assignees:
  - human:sothr
milestone: null
parent: null
dependencies: []
blocks_on: none
references:
  - ref: proposal:git-ticket
    path: docs/proposals/git-ticket.md
claim: null
archive: null
created_at: 2026-08-31T12:00:00Z
updated_at: 2026-08-31T12:00:00Z
created_by:
  id: human:sothr
  name: Drew Short
updated_by:
  id: agent:terva/session-123
  name: Mieli
extensions: {}
```

`title` is one line saying what the ticket is about, written for a person
reading it next to an ID that says nothing. A ULID is not something anybody
holds in their head, so the title is the only part of a reference that carries
meaning, and it is quoted beside the ID wherever one appears.

It is bounded so it stays readable inline. Over 72 characters is `title_long`, a
warning. Over 120 is `title_too_long`, an error, and a write is refused at that
length with `invalid_field` rather than being reported afterwards, the way an
invalid `priority` is refused by `create`. Length counts characters and not
bytes, so a title does not fail for being written in a language that needs more
of them.

Two thresholds rather than one, because a single cap tight enough to be useful
would be retroactive on a store that already exists. Across the 62 tickets here
the median title is 57 characters and 23 are over 60, so a cap at 60 would make
a third of the store non-conforming at once, and `check --strict` is what gates
CI. A warning names the long ones without stopping anybody, and the error
threshold sits far enough out that only a title nobody meant to write reaches
it.

There is no second, shorter title field. 120 characters is generous enough that
a `short_title` beside it would carry the same meaning twice, and two fields
that mean one thing drift.

`type` is one of `task`, `bug`, `chore`, `spike`, `epic`. `priority` is one of
`low`, `normal`, `high`, `urgent`. `blocks_on` is one of `none`, `children`.

`due_on` holds a `YYYY-MM-DD` date and means the end of that day in UTC. Absent
is `null` and means no deadline. It is the one time value in this format that is
not an RFC3339 instant, which is why it ends `_on` rather than `_at`. Everywhere
else in this section `_at` is an instant, and a `due_at` holding a day would
teach a reader otherwise and make `expires_at` on the claim block ambiguous.

The value is quoted, which is 5.3's existing rule rather than a rule for this
field: the renderer quotes any scalar a YAML reader would resolve to something
other than a string, and a bare `2026-10-14` is a YAML timestamp. Quoting is
also what makes the field portable, because a YAML 1.1 reader hands a bare date
back as a date object and a quoted one as the string this section specifies. A
hand-written bare date still parses, since the format is meant to be edited by
hand, and the next write canonicalizes it.

The field holds a constraint from outside the ticket and is not a second
priority. "It has to be done by the 14th" is a date. "I want this sooner" is
`priority`, and `priority: urgent` carrying no date stays a complete statement.
The line is worth holding, because a field that accepts eagerness fills with
dates nobody chose, and then every row reads as late and the field stops meaning
anything.

The date is stored as written rather than expanded to an instant. A deadline is
a claim about a calendar day, and no deadline was ever 23:59:59Z. Expanding also
has to pick a zone at write time, and the writer's local zone stores two
different instants for the same typed date depending on who typed it. A writer
handed an instant where a date belongs rejects it rather than truncating it,
because truncating throws away a distinction the author can be seen making.

`blocks_on` names the edges that gate a ticket beyond its `dependencies`, which
always gate, per 6.3. It is additive and never selective. `children` adds the
direct children to the gating set and takes nothing away, so an epic that waits
on both an external ticket and its own decomposition says so without choosing.

Selective was the first design and it was wrong. A value meaning "my
dependencies do not gate me" would let one field undo the rule 6.3 defines, and
because 5.3 renders every known field on every ticket, the default would spell
that change across a whole store at once. There is also no use for it: a
dependency somebody listed is one they meant, and a link that should not gate is
a `ticket:` entry in `references`.

`children` is a rule and not a set. The gating children are derived from `parent`
at read time and never written down, so decomposing an epic edits the child and
leaves the epic alone. An epic that enumerated its children would be edited by
every decomposition, which turns the one file several agents share into the one
file they all conflict on.

An epic with `blocks_on: children` and no children is not blocked. Blocking it
would put a ticket in `blocked` with nothing to name as the blocker, which
section 8 refuses for a draft on the same grounds. `check` reports the state as
`blocks_on_no_children`, because it is an authoring mistake rather than a
readiness verdict: a new ticket is a `draft` and only reaches `ready` when
somebody promotes it.

`status_reason` holds the reason 6.2 requires when a ticket enters `blocked` or
reopens from `done`. It is the current reason and not a history: a transition
that requires a reason overwrites it, and any other transition clears it. The
field answers "why is this blocked now", which a tool can act on. `Notes`
answers "what happened to this ticket", which a person reads.

`references` carry a typed stable identifier and an optional repository-relative
path. The core preserves namespaces such as `idea:`, `proposal:`, `plan:`,
`decision:`, `review:`, `commit:`, `file:`, `url:`, and `ticket:` without
interpreting the project-specific ones. It resolves `path` far enough to report
a broken link in `check`.

`extensions` is a namespaced mapping for integration data. It is the only place
a consumer may write fields the core does not define, and the core never
interprets its contents.

The claim block, when a ticket is claimed:

```yaml
claim:
  actor: agent:terva/session-123
  branch: feat/token-refresh
  worktree: /Users/sothr/wt/token-refresh
  commit: a1b2c3d4e5f6
  claimed_at: 2026-08-31T12:04:00Z
  expires_at: null
```

The archive block, when a ticket is archived:

```yaml
archive:
  archived_at: 2026-09-04T09:00:00Z
  from_status: done
  reason: shipped in v1.2
```

`from_status` exists so that archiving does not silently break dependents. See
section 6.3.

### 5.2 Body sections

Known sections, in this order:

1. `## Description`
2. `## Acceptance criteria`, a checkbox list
3. `## Definition of done`, a checkbox list
4. `## Implementation plan`
5. `## Notes`
6. `## Comments`
7. `## Summary`

`Description` is always emitted. The rest are emitted only when they have
content. Unknown sections survive a round trip and are appended after the known
ones in their original relative order.

A section opens on any line beginning with `## `, and nowhere else. A line
inside a fenced code block is text, and an indented one is text, so a ticket may
quote Markdown without the quotation becoming structure. The heading is not
matched against the list above before the split, which is what lets an unknown
section survive rather than being refused.

That rule reaches further than it looks. Text written into one section that
carries such a line ends that section early: the text above stays and everything
below becomes a section of its own. Nothing downstream reports it, so the CLI
warns at the point of writing, per section 10.

### 5.3 Deterministic rendering

The renderer must be a pure function of the parsed ticket. Two writers producing
the same logical ticket must produce identical bytes, or the golden tests and
the CLI-versus-terva acceptance criterion cannot hold.

- Frontmatter keys in exactly the order listed in 5.1, then any preserved
  unknown keys in their original order.
- Block style for non-empty sequences and mappings. An empty collection is `[]`
  or `{}`, and an absent scalar is `null`.
- No YAML anchors or aliases, and no flow style except the two empty forms
  above, which are the canonical way to write an empty collection. Quote a
  string only when YAML requires it.
- LF line endings and exactly one trailing newline.
- Checkbox items as `- [ ]` and `- [x]`.

The round-trip guarantee, enforced by test:
`render(parse(render(t))) == render(t)` for every fixture, and `parse` of a
supported file loses no content.

The renderer writes body sections verbatim and normalizes nothing. `parse`
strips the blank lines around a section, so a `Ticket` built in Go with a padded
section would otherwise render bytes that parse back to something rendering
differently, which is the guarantee above failing. The normalization therefore
happens once, in the store, on the way to disk: every write funnels through
`writeTicket`, and it puts the body in the shape `parse` returns before
rendering.

It belongs there rather than in the renderer because `writeTicket` hands back
the same ticket it rendered. Callers read that struct and the CLI serializes it,
so a renderer that normalized on its own would leave the struct and the file
disagreeing about the ticket's own text. Byte instability shows up in a diff;
that divergence would not.

The normalizer strips blank lines at the edges of a section and nothing else. It
is not `TrimSpace`. Leading whitespace on a content line survives a parse
untouched, so trimming it would fix no round trip and would silently reindent a
section that opens with an indented code block. A heading is the one part that
takes `TrimSpace`, because `parse` reads one that way.

`updated_at` and `updated_by` change on every mutation, so every diff shows at
least those two lines. That is intended: the diff should say who touched the
ticket and when. Both stay, and they are one fact rather than two fields.

Be exact about what the pair answers, because it is narrower than it looks. It
names the last writer and no earlier one, so a ticket whose description one
actor wrote and whose label another added reports only the second. The per-entry
log is elsewhere: `Notes` and `Comments` stamp every entry with its own actor and
time, per 5.2. The pair answers who last touched the file, which is the question
a reader asks of a diff.

That nothing reads either field is true and is not an argument against them.
Nothing reads `created_at` or `created_by` either, and no query, filter, sort, or
check consumes any of the four. Provenance answers a question a person asks of a
diff rather than one a caller asks of a store, so being unread is the ordinary
condition of the whole group. A rule that dropped a field for having no reader
would take `created_by` with it.

The Git object database does not already hold this, which is the claim worth
testing rather than assuming. Two things break it. Nothing here commits, per
7.4, so a mutation exists on disk for as long as it takes somebody to get around
to committing, and until then `git log` knows nothing about it. When a commit
does arrive it carries the committer's Git identity, while an actor is an agent
session. This repository's own store holds `agent:terva/mieli`,
`agent:terva/dev-loop`, and `human:sothr` across its tickets, and every commit in
it is authored by one person. Git collapses the three into that one. On the axis
an agent workflow cares about, Git holds strictly less than the pair does, not
more.

The two cannot be separated. 7.5 resolves a conflict by taking the later
`updated_at` and the actor belonging to it, so dropping the timestamp would leave
the actor with no rule to be resolved by. Keeping one and dropping the other is
therefore not a third option, whatever the two look like listed side by side.

### 5.4 Unknown fields and schema drift

Hard rejection of unknown top-level fields would make an additive v1.1 field
break a v1.0 reader in the same repository, which is exactly the mixed-speed
client problem this format has to survive. So:

| Context | Behaviour on an unknown top-level field |
|---|---|
| `check` | Error. Exit nonzero. |
| Ordinary read | Warn on stderr, parse the rest, preserve the field. |
| Write | Preserve the field, re-emitted after the known keys in its original relative order, per 5.3. |
| `schema` greater than the reader supports | Refuse with `schema_unsupported` and name the version needed. |

A field may only be removed or given a new meaning at a schema major bump.
Adding a field is a minor change.

### 5.5 IDs and reference resolution

An ID is `TKT-` followed by a 26-character Crockford base32 ULID, uppercase.
ULIDs need no central counter, so two disconnected agents cannot collide, and
they sort by creation time, which makes a directory listing chronological for
free.

Twenty-six characters is too many to type, so any command taking an ID accepts a
unique case-insensitive prefix, with or without the `TKT-` part. A prefix must
be at least four characters of the ULID to be considered, which stops a typo
from resolving by accident. An ambiguous prefix returns `ambiguous_id` and lists
the candidates. This is git's rule for object hashes and users already know it.

A `references` path resolves against the root of the Git repository holding the
store, found with `git rev-parse --show-toplevel`. Repository-relative is the
only root that survives a clone: an absolute path breaks as soon as the
repository is checked out somewhere else, and a store-relative path would mean
two different things for `.tickets/` and for a `--store` override. It is also
how a person already writes a path, `docs/plan.md` rather than `../docs/plan.md`.

When `--store` or `GIT_TICKET_STORE` points outside any repository there is no
such root, and `check` skips path resolution rather than guessing one. It
reports no `reference_path_unresolved` finding at all in that case, because a
finding measured against a guessed root is noise the user cannot act on.

A ref itself is a namespace, a colon, and an identifier, which is what 5.1 means
by a typed stable identifier. The split is at the first colon and not the last,
so `url:https://example.invalid/x` is the `url` namespace holding a URL rather
than a namespace called `url:https`.

`refs` compares the namespace without regard to case and the identifier exactly.
A namespace is a type, and the ones 5.1 lists are lower case, so `JIRA:` is not
a second type for being shouted. The identifier belongs to whichever system
issued it, and only that system can say whether its case is significant, so
git-ticket does not guess: `jira:PROJ-1234` and `jira:proj-1234` stay two
references. Neither comparison changes the stored bytes, which 5.3 renders
verbatim.

A ref with no colon is untyped. It is still stored and still round-trips, and
`check` reports `reference_untyped` against it, per section 11. That is a
warning rather than an error because the ref does name something, and only a
person knows which namespace it belonged in. A query with no colon matches an
untyped ref whole and reaches into no namespace, so `PROJ-1234` does not find
`jira:PROJ-1234`. Ask for a namespace with the colon.

## 6. Status and lifecycle

### 6.1 The status set

```text
draft
ready
in-progress
blocked
review
done
archived
```

`claimed` is deliberately absent. Claim metadata already records the actor,
branch, commit, and expiry, so a `claimed` status would be a second copy of the
same fact that can disagree with the first in both directions: a ticket in
`ready` with a live claim, or one in `claimed` with no claim block. Claim is
orthogonal, and the `ready` query filters on it.

### 6.2 Transitions

| From | Permitted to |
|---|---|
| `draft` | `ready`, `archived` |
| `ready` | `draft`, `in-progress`, `blocked`, `archived` |
| `in-progress` | `ready`, `blocked`, `review`, `done`, `archived` |
| `blocked` | `ready`, `in-progress`, `archived` |
| `review` | `in-progress`, `blocked`, `done`, `archived` |
| `done` | `in-progress`, `archived` |
| `archived` | `ready` |

Anything else returns `invalid_transition` naming the permitted targets.

A `--reason` is required for a transition into `blocked` and for reopening from
`done` to `in-progress`. Both are cases where a later reader needs to know why,
and neither has a dedicated command to make the intent obvious. It is accepted,
but not required, on every other transition.

The reason lands in two places. It is written to `status_reason`, where a query
can read it back, and appended to `Notes` with the actor and the timestamp,
where it survives the transition that clears the field. One place would have
cost one of the two: a field alone forgets why the ticket was blocked last
month, and a note alone cannot answer why it is blocked now without a human
reading prose.

`archived` is not reachable through `git ticket status`, because archiving also
moves the file. `status ID archived` is refused with a pointer to `git ticket
archive`. The reverse pair is `git ticket unarchive`, which restores the file to
`tickets/` and sets the status to `ready`.

### 6.3 Archive and dependency resolution

Archiving sets `status: archived`, records the `archive` block including
`from_status`, and moves the file to `.tickets/archive/`. The status is
authoritative. If the two ever disagree, because someone moved a file by hand,
`check` reports `location_mismatch` and the status wins.

That rule is not special to the archive. Every status implies a directory, per
section 4, and the same finding covers all four.

An archive reason lands in two places, like a status reason and for the same
reason. It goes in the `archive` block for a query to read, and into a `Notes`
entry that outlives the block. `unarchive` deletes the block, so without the
note a ticket archived as "shipped in v1.2" and then unarchived keeps nothing
that says why it was ever closed out.

A dependency is satisfied when the depended-on ticket is `done`, or when it is
`archived` with `from_status: done`. Archiving a ticket that was never done does
not satisfy anything, and `check` warns when a live ticket depends on one. This
is why `from_status` is recorded: the ordinary flow is done and then archive,
and without it every archive would silently block its dependents.

### 6.4 Claims

`git ticket claim` records the actor, the branch and worktree when they can be
determined, the commit the claim was based on, the time, and an optional expiry
from `--expires-in` or `defaults.claim_expiry`. There is no default expiry.

Claiming is permitted from `ready`, `in-progress`, `blocked`, and `review`. It
is refused on `draft`, `done`, and `archived`.

An expired claim makes the ticket eligible for another claim. It does not revoke
the original agent's work, and it grants no exclusivity to anyone.

Claiming a ticket that already carries a live claim by a different actor returns
`claim_conflict`. `--force` overrides it and records the displaced claim in
`Notes`, because taking work from another agent should leave a trace.

Re-claiming a ticket you already hold renews the claim rather than replacing it.
`claimed_at` survives, because it is the only record of when the work started
and renewing is not restarting. The expiry follows whatever supplied it.
`--expires-in` and `defaults.claim_expiry` both re-anchor the bound from now,
and when neither supplies one the renewal keeps the `expires_at` already on the
live claim. Clearing it would let the ordinary gesture for staying alive on a
long task widen a bounded claim into an unbounded one. A claim that has already
lapsed is re-acquired rather than renewed, so it takes whatever the expiry
sources give it, and that loses nothing, because a lapsed claim already grants
no exclusivity. The branch, worktree, and commit always describe the claim being
recorded now, so a renewal from a different worktree updates them.

Renewal needs an explicit duration every time, because the stored claim keeps
`claimed_at` and `expires_at` and never the duration between them. Extending a
claim by the amount it was originally given is not implementable from what is on
disk.

## 7. Concurrency

### 7.1 The revision precondition

Every read returns `revision`, the SHA-256 of the ticket file's bytes as they
sit on disk, formatted `sha256:` followed by 64 lowercase hex characters. It is
computed, never stored, so there is no field to merge-conflict on and no field a
hand-editor can forget to bump.

That last point is the reason for the choice. Three things change a ticket
behind a caller's back: another process in the same checkout, a merge or branch
switch between read and write, and a person or agent editing the file directly.
Any token the writer maintains, whether `updated_at` or a monotonic counter,
fails the third case silently, and a counter additionally has no correct value
after a merge of two branches that both incremented it. A hash of the bytes
catches all three and costs nothing, because the file is already being read
under the lock.

Mutations accept `--if-revision` on the CLI and `ifRevision` in JSON. When it is
supplied, the store re-reads and re-hashes the file after taking the lock and
returns `stale_revision` with both the expected and the actual value if they
differ. When it is omitted, the last write wins under the local lock.

The precondition is optional in the library and CLI so that a person typing
`git ticket status TKT-01K3ZYEE done` at a terminal is not forced into a
read-then-write dance where no concurrency exists. Terva's tool schema marks it
required, because agents read before they write anyway and multi-agent is where
the races actually happen.

A revision is an equality check, not an ordering, and not a merge. It tells a
caller that it lost a race. Git still resolves the content.

### 7.2 The local lock

One lock guards the whole store. Contention is rare and per-ticket locking would
complicate the multi-file operations in `check` and `archive` for no gain.

The lock file lives at `<git-common-dir>/git-ticket/store.lock`, found with `git
rev-parse --git-common-dir`. Putting it under the common Git directory means it
is shared by every worktree of the repository and is never committed. The
implementation uses `flock`, which the kernel releases when the holder dies, so
there is no stale-lock breaker to get wrong.

Acquisition blocks up to `lock.timeout`, default ten seconds, and then returns
`lock_timeout`. When the store is outside a repository, because of a `--store`
override, the lock falls back to `<store>/.lock` and the tool documents that
this does not coordinate separate worktrees.

The write path is: acquire, read, verify the precondition, write a temporary
file in the same directory, `fsync`, atomically rename over the target, release.
A failure at any step leaves the original file untouched.

### 7.3 Across clones and worktrees

Separate worktrees that share a Git directory share the lock, so they serialize
correctly. Separate clones do not, and no local command can change that.

A claim made in a clone becomes visible to another clone only after its commit
reaches a shared ref and the other clone fetches it. The data model records the
commit a claim was based on precisely so that output can show how provisional it
is, and `show` marks a claim whose commit is not an ancestor of any remote-
tracking ref as unpublished.

V1 has no sync command. Publishing happens through ordinary `git commit`,
`push`, `fetch`, and `merge`. A later sync helper may be worth evaluating, but
it must never silently push, merge, switch branches, or rewrite a worktree.

### 7.4 The Git commands this code runs

The rule above is a promise about code that runs inside someone else's
repository, and since `v0.2.0` a host can embed the command surface, so the
promise now travels into binaries this project does not build. Prose cannot hold
it. This is the enumerated list, and a test holds the source to it.

| Command | What reads it |
|---|---|
| `rev-parse` | the repository root for path resolution, the common Git directory for the lock, and `HEAD` for the commit a claim records |
| `symbolic-ref` | the branch a claim records. It fails on a detached HEAD, and that is what tells the two apart |
| `config` | `install-merge-driver` sets `merge.gitticket.name` and `merge.gitticket.driver`, and reads them back to report what it changed |
| `for-each-ref` | the refs a cross-branch query considers and the last commit date of each, which is the recency filter it applies before reading any tree, per section 8. `%(symref)` comes back too, so `origin/HEAD` can be skipped rather than read twice under a second name |
| `ls-tree` | the ticket files on one ref and the blob name of each, for a cross-branch query. It reads a named tree object and never the working tree and never the index |
| `cat-file` | one ticket as it stands on one ref, by the blob name `ls-tree` gave. Listing a tree names files without opening them, so a cross-branch read needs both, and the blob name is what lets a version shared by ten branches be read once |

Every row but `config` only reads. `config` writes, and the exception is narrow
on purpose: it touches `.git/config` alone, never history, the index, or a
tracked file, and it runs from `install-merge-driver` and from no other command.
Somebody typing that command is asking for exactly that write, which is the
whole of what it does.

Every call goes through one helper per package, `runGit` in `ticket` and
`readGit` or `writeGit` in `cli`, and `TestGitCommandsAreReadOnly` asserts three
things: no `exec.Command` in non-test code names a binary other than `git`,
every one of those calls sits in one of the helpers, and every helper call names
a command from this table. A fourth call site added tomorrow has to pass all
three, and a new helper fails the second rather than slipping past the third.

No command that moves history or the working tree joins this table. Section 15
records that decision under sync helpers, so a change that needs `fetch` or
`push` is a change to the plan first.

This table binds the tool and not a workflow, which can run plain `git` with the
tool nowhere in the pipe. Section 11 carries that rule separately, under
verifying generated artifacts in CI, and reaches the same answer: a job verifies
a repair and never commits one.

Test code is exempt and runs `git init` freely, because a fixture repository is
not a user's repository.

### 7.5 Merging ticket files

git-ticket ships a merge driver. `TKT-01M1JPW1QJR3QM4SE9T3GDTZBX` builds it, and
this section is the design it builds to.

7.1 hands content resolution to Git and stops, which is right for a format made
of text files and incomplete about what Git then does to them. Two agents each
filing a ticket merge cleanly, because each writes a file the other does not
have. Two agents editing one ticket conflict, and the reason is not what it
looks like.

Every mutation rewrites `updated_at` and `updated_by`. Two edits with nothing in
common still change those lines, so Git sees both sides modify one region and
stops. Branch A setting `priority` against branch B adding a label conflicts on
`updated_by` alone: Git merges the priority and the label without complaint and
hands back a file whose only disputed lines are the two neither agent typed.
The table below settles that cost, taking the later `updated_at` and the actor
belonging to it, which needs no judgment. Whether the two fields belong in the
published surface at all is `TKT-01M1JPXG7J6K4MQNTWA6TVMHPB`, and it is a
separate question: nothing reads either field, and 12.4 closes the option once a
consumer depends on them.

Position is the second source. `Notes` and `Comments` grow by appending, so two
appends land at the same offset and collide as one hunk, though nothing in
either entry contradicts the other.

Both are resolvable without adjudicating anybody's data, which is what makes a
driver worth shipping rather than a rule telling people not to do that.

| Field | Rule |
|---|---|
| `schema`, `id`, `created_at`, `created_by` | conflict. These never change, so a difference is corruption and not a merge |
| `updated_at` | the later of the two |
| `updated_by` | the actor belonging to that later `updated_at`, so the pair stays one fact |
| `title`, `type`, `status`, `status_reason`, `priority`, `due_on`, `milestone`, `parent`, `blocks_on` | take the side that changed, and conflict when both changed to different values |
| `labels`, `assignees`, `dependencies`, `references` | union. These are unordered sets, so two additions are compatible |
| `claim` | conflict whenever the sides differ. A claim is a mutual exclusion record, and resolving it silently hands one ticket to two agents, each holding a file that says it is theirs |
| `archive` | conflict, for the reason `claim` does. Two sides archiving out of different statuses disagree about history |
| `extensions` | merge by key, and conflict on a key both sides changed |
| `Description`, `Implementation plan`, `Summary` | prose, so conflict when both sides changed it |
| `Notes`, `Comments` | union, ordered by timestamp, dropping an entry identical on both sides |
| `Acceptance criteria`, `Definition of done` | union by item, and conflict on an item both sides changed |

A driver stays inside the 7.4 promise. Git invokes it, and it runs no Git
command itself, so the table does not grow.

Installing one takes two pieces and a repository can supply one. A
`.gitattributes` entry naming the driver is a tracked file, so `init` can write
it and a clone gets it. The `merge.*.driver` config line is not tracked, and Git
refuses on purpose to take an executable name from a repository, because a clone
would otherwise run a command the person cloning never chose.

That boundary is correct and not an obstacle to route around. It makes the
config line a person's decision every time, so the tool's job is to make it one
command they run rather than a paragraph they transcribe.

`init` writes the `.gitattributes` entry. The attribute does nothing until
somebody configures a driver by that name, so it costs a clone that never
installs one two lines and a mention in a diff, and it saves every clone that
does install one from having to know the pattern.

`git ticket install-merge-driver` is the other half. It sets the two config keys
in `.git/config`, adds the `.gitattributes` entry if `init` predates this or
somebody removed it, and prints what it changed and what was already right. It
names the running executable by its absolute path, because a driver line reading
`git-ticket` fails the moment Git runs from a directory where that is not on
`PATH`. Running it twice is not an error and the second run reports that there
was nothing to do.

## 8. Query surface

- `list` with filters on status, type, priority, label, assignee, milestone, and
  parent. Within one filter the values are alternatives and across filters they
  all have to hold. It answers with open work: every status except `done` and
  `archived`. A caller who wants those names the status, or asks for everything
  with `--all`. `--due-by` takes a
  date and selects the tickets due on or before it, which is the query the
  field exists for, because today's date answers what is late. It is a bound
  and not a set, so the alternatives rule above does not apply to it, and a
  ticket carrying no `due_on` is not due by any date and never matches
- `ready`: status `ready`, no live claim, and every dependency satisfied per 6.3.
  Only direct dependencies are read, so a dependency cycle cannot make this loop
- `show` for one complete ticket
- `search` over title, description, acceptance criteria, definition of done,
  notes, comments, summary, and references. Case-insensitive substring by
  default, `--regex` for RE2
- `deps` for direct and transitive dependencies, `--dependents` for the reverse.
  It walks `dependencies` and nothing else. When the answer is empty and the
  ticket has children, the human output names the count and points at
  `list --parent`, because "it depends on nothing" is true and useless on an
  epic. That is a pointer to the other command, not a second edge kind in the
  result, and `--json` does not carry it. This holds after `blocks_on`: a child
  that gates its epic is still not a dependency, and mixing the two edge kinds
  into one walk is what this bullet declines. `readiness` is where a caller
  learns that children are in the way
- `files PATH` for tickets referencing a path. This searches recorded `file:`
  references and is only as complete as the agents that wrote them. It is
  advisory and not derived from Git history, and the help text says so
- `refs REF` for tickets carrying a reference, matched by the rule in 5.5.
  `refs jira:PROJ-1234` is one work item and `refs jira:` is every ticket in
  that namespace. This is the lookup `search` cannot do: a search matches
  substrings across the body, so it also returns a ticket that only mentions
  the number in prose, and it cannot tell a namespace from any other text
  before a colon. It carries the same advisory caveat as `files`.

  `files` stays a separate command rather than sugar for `refs file:PATH`,
  because it also matches the `path` half of a reference, which a namespace
  query does not. The two answer different questions about the same field
- `check`, described in section 10

### What a listing answers by default

`list` answers with open work. `done` and `archived` are left out, and every
other status is in.

The rule is an exclusion rather than a list, so a status added later is included
without an edit here. Whether this format grows custom statuses is still open in
section 15, and an inclusion list would silently drop any status that arrived.
The epics index settles the same question the same way, and the two must not
disagree.

Naming a status brings it back. `--status done` answers with done tickets,
because a caller who named the status has already said what they want, and a
default that overrode them would be a filter that ignores its argument. `--all`
drops the exclusion entirely.

The default applies to every filter, `--parent` included, so `list --parent EPIC`
answers with the children still open. That is the consistent reading and the
reason to take it: a carve-out for one filter is a second rule to hold in mind,
and the caller who wants an epic's whole roster asks with `--all`. Nothing that
reasons about children reads a listing anyway. `readiness` computes
`blockingChildren` from the whole store, so a done child still counts as
satisfied whatever a listing shows.

The old default was every ticket except archived. In this repository's own store
that meant 44 rows with 30 of them done, so an agent asking what was open had to
throw most of the answer away, which is the sign that a command answered a
different question than the one asked. Archived was already excluded, so the
default already hid terminal work. The line sat between two statuses that are
terminal in the same way.

`--archived` is renamed `--all`, and `Filter.IncludeArchived` becomes
`Filter.All`. Under the old default, everything-except-archived plus archived
was everything, so the flag already meant what `--all` now says, and keeping
both would be two spellings of one behaviour.

`search` does not take this default. It searches every status, `done` and
`archived` included, and its filters narrow from there. The asymmetry is
deliberate. `list` answers what to work on, while search is how somebody finds
what was already decided, and what was already decided is what a done ticket
holds. A search that hid the answers would be worth replacing.

`files` and `refs` read the same way, for the same reason. Both answer "which
ticket is this", and the honest answer to "which ticket is PROJ-1234" is often
one that shipped last quarter. A lookup by identifier that hid finished work
would report that nothing tracks an item somebody already closed, which is worse
than reporting nothing at all.

The library reads the same way as the CLI. `Filter{}` means open work rather
than no filtering, so a host embedding `ticket` gets the answer the CLI gives
without restating the status set. `Filter{All: true}` is everything.

One caller inside this project needs everything and says so. The abbreviation
table behind a listing shortens an ID against every ticket in the store, done
and archived included, because what a listing prints gets pasted into a command
that resolves against all of them. An abbreviation computed against open work
alone would print a prefix that resolves to two tickets.

Ordering by `due_on` puts the earliest date first, the undated tickets last, and
breaks a tie on the ID. Never is the far end of "closest to late first". The ID
tiebreak is what makes the key safe to apply without asking: in a store where
nobody has set a date, the order is the one the store had before the field
existed.

Ordering by `priority` puts `urgent` first and `low` last, breaks a tie on
`due_on`, and breaks that tie on the ID. A value 5.1 does not define sorts below
`low`, because an unrecognized priority must not outrank one somebody set on
purpose. The key degrades the way the deadline key does: every ticket carries
`normal` until somebody says otherwise, so in a store where nobody has set a
priority the key changes nothing and the order falls through to the two beneath
it.

`ready` applies the priority order always and has no flag to ask for it, because
its ranking is part of what it answers. It is the order `list --sort priority`
gives with the readiness filter in front of it, so a reader comparing the two
sees one order rather than two.

Priority leads and the deadline follows it. The case that decides this is a
`low` ticket due next week against an `urgent` with no date, and both orders are
wrong somewhere, so the question is which wrong answer a person can correct
honestly. Under deadline-first the only way to lift the urgent ticket is to
invent a date for it, which corrupts a field to move a sort. Under
priority-first the way to lift the dated ticket is to raise its priority, which
is what the field means. An ordering should be gamed by telling the truth.

The other half is that undated already means never due, and never is the far end
of the scale. Deadline-first has to defend a never-due `urgent` sorting below a
`low` due in two years, and it cannot.

`list` reports what exists rather than recommending anything, so reordering it
is a change to a report somebody is reading, and the caller asks. `--sort` takes
`id`, `due_on`, or `priority`, and `id` is the default. Naming the default is
half of why the flag takes a value rather than being a bare switch.

One field has one order. Both commands sort a field the same way, because two
orderings of one field is a rule a reader has to hold in mind at the moment they
are comparing two outputs.

Every ticket carries `readiness`, which is derived from the whole store at read
time and never stored, like `revision` and `path` in 7.1. It holds the verdict
`ready` filters on, and what stands in the way when the answer is no.

`ready` filters on that verdict rather than restating the rule, so the query and
the field cannot come to disagree about one ticket. A consumer drawing a board
was otherwise forced to call `ready`, diff two ID sets, and then call `deps` per
card to explain the difference.

Blocked covers dependencies and blocking children. A draft, and a ticket
somebody else holds, are both unready with nothing in the way but their own
state, and calling those blocked would send a reader looking for a dependency
that is not there.

`reason` names why, and is empty exactly when `isReady` is true. Without it the
sentence above described a hole rather than a feature: a draft came back not
ready, not blocked, and with three empty arrays, so a consumer had to re-derive
the answer from `status` and `claim`. That re-derivation is what `readiness`
exists to prevent, and in this repository's own store it was the answer for ten
of the eleven open tickets.

One field rather than a set of booleans, because more than one can be true at
once and a consumer would then need the precedence. Answering it once here costs
less than answering it in every consumer, and the real cost is two consumers
answering it differently.

The precedence is status, then dependencies, then the claim, and it reads as
what has to change first. A draft waiting on a dependency reports `draft`,
because promoting it is the move that comes first and nobody acts on the
dependency of a ticket that is not in the queue. Nothing is hidden by the
choice: `blockingDependencies` and `blockingChildren` are still populated, so
the reason names the operative blocker and the arrays carry the rest.

Every status except `ready` is its own reason, so a status added to 6.1 becomes
a reason with no further decision. The two that are not statuses are
`waiting_on_dependencies` and `claimed`.

`waiting_on_dependencies` is spelled out rather than called `blocked` because
both senses of that word are already load bearing and neither can be renamed.
`isBlocked` means the dependency graph, and the `blocked` status means a person
marked the ticket and wrote a reason. The third name keeps one meaning per
field: `blocked` in a reason is always the status, `isBlocked` is always the
graph.

`claimed` is not `claimed_by_other`, because readiness is computed from the
store and a clock with no actor in the call. It cannot tell your own claim from
somebody else's, and a name asserting that difference would be one the code
cannot check. A caller comparing `claim.by` against itself can, and should.

The two edge kinds keep separate fields. `blockingDependencies` is published and
versioned under 12.4, so a child arriving in it would make a consumer print a
child ID labelled as a dependency with nothing to signal the difference.
`blockingChildren` is a new key, which an older consumer ignores and reads what
it always read. Widening `isBlocked` to cover both is the deliberate half: it
answers whether a ticket can be started, and an epic waiting on its children
cannot be.

A dependency that resolves to nothing, or to more than one file, blocks and
never counts as satisfied. Both are states `check` already reports, as
`dependency_missing` and `duplicate_id`. Failing closed is the only safe
reading. Until somebody repairs the store, nothing can be said to have met the
dependency, and guessing would hand an agent work whose prerequisite nobody can
point at.

The parent filter answers what is under an epic, which is the one hierarchy
question this format could record and not read back. It matches direct children
only, and little is lost by that: `list` already returns every ticket with its
`parent`, so a caller wanting a whole tree builds it from a single call. A
filter that walked the hierarchy would have to precompute a descendant set
before it could answer a question about one ticket, which is not what a filter
is.

`--parent none` selects the tickets that have no parent, which is what a board
needs for its top level. `none` cannot be mistaken for an ID because every ID
begins with `TKT-`. The library spells the same thing as an empty string, which
is how the milestone filter already reads an absent value.

The CLI resolves a parent the way it resolves every other ID, so a prefix works
and an ID matching nothing is `ticket_not_found`. Filtering on a parent that
does not exist is a mistake worth reporting, and a silent empty result reads
exactly like an epic with no children.

Search reads every file on every call. At the scale this format targets,
hundreds to a few thousand tickets, that is a few milliseconds and needs no
index. An index is deferred, and if one is ever added it must be disposable and
rebuildable from the files.

A query leaves out a ticket whose file does not parse. `list`, `ready`,
`search`, `deps`, `files`, and `refs` all read the whole store, and a query is
not the place to learn that one file is broken. `check` is, and it reports the
file as `parse_error` or `schema_unsupported`.

Naming one ticket is different. `show` and every mutation resolve a ref the
caller supplied, and a ticket that is present but unreadable answers with the
failure `check` would report rather than with `ticket_not_found`. A file sitting
on disk is not absent, and calling it absent sends a reader looking in the wrong
place. This is the rule the parent filter already follows above, where a silent
empty result reads exactly like an epic with no children. The format is meant to
be hand-edited, so a YAML typo in a ticket is the ordinary way to reach this,
not an exotic one.

Resolution therefore sees unreadable files rather than skipping them. Skipping
them is worse than a misleading error. A prefix matching both a broken ticket
and a readable one would resolve to the readable one and answer about a
different ticket than the one asked for. `git ticket list` prints
thirteen-character prefixes, which is exactly what a person then types, so that
collision is reachable rather than theoretical. A broken file's ID comes from
the parse error when the frontmatter got that far, and from the filename
otherwise, which is why a ticket file is named for its ID.

### Reading tickets from other branches

`list` and `ready` take `--cross-branch`, which widens the read from the working
tree to the branches around it. A ticket another agent claimed and moved is then
visible before their branch merges. No other command takes the flag and it is
off by default.

It is a flag rather than a store setting because it asks a different question
than "what is in my tree", and the caller is the one who knows which they want.
A store setting would answer the wider question for everybody who reads it,
including scripts that want the narrow one.

**What it reads.** `for-each-ref` over `refs/heads/` and `refs/remotes/`, both.
An agent working here pushes to `origin`, so in every other worktree their work
is a remote-tracking ref, and a scan of local heads alone would miss the case
this exists for. A ref whose last commit is older than 30 days is skipped, which
bounds the scan; that window comes from the mechanism described in section D of
`docs/review-backlog-md.md`. Each surviving ref is read with `ls-tree` over the
whole `.tickets/` subtree rather than `.tickets/tickets/`, because the store
partitions by status and a file's directory is its status.

`ls-tree` names the files and their blobs without opening them, so `cat-file`
reads the ones that matter. It is addressed by blob rather than by path and ref,
which is what makes the scan affordable: a ticket nobody has touched is the same
blob on every branch, so the cost is one read per distinct version rather than
one per ticket per ref.

**Which copy wins.** The working tree is one source among the refs, not a
privileged one, and the copy with the later `updated_at` wins. Every mutation
rewrites `updated_at`, so an uncommitted edit already carries the newest
timestamp and needs no special rule to beat a committed one. A tie goes to the
working tree, because that is the copy the caller can act on.

This is the rule 7.5 applies when Git merges two versions of one ticket. A
listing that resolved differently would predict the wrong outcome of the merge
it is warning about. It also reads a field rather than guessing from a file's
modification time, which is the guess this format declines.

**Claims are not resolved.** A live claim on any scanned ref makes the ticket
not ready, and `ready` never picks between two claims. 7.5 conflicts a differing
`claim` rather than resolving it, on the ground that resolving hands one ticket
to two agents each holding a file that says it is theirs. A query has less
standing to adjudicate than a merge does, so the conservative answer is the only
one available: if any ref says it is claimed, it is claimed.

**Provenance.** Every row names the ref its winning copy came from, in the short
form `for-each-ref` reports, so `fix/thing` and `origin/fix/thing` stay
distinguishable. A row whose winning copy is the working tree names nothing,
which makes "this file is not in your tree" the visible case rather than the
silent one. The JSON field is `branch`, in 10.1. Without it a merged view would
let an agent claim a ticket whose file it cannot open, which is a worse failure
than not seeing the ticket at all.

**It never writes.** No mutation takes the flag. A claim, a status change, or a
note still applies to the working tree alone, so seeing another branch's ticket
is not the same as being able to change it.

## 9. Mutation surface

Every mutation changes only the fields the caller named. Full-file replacement
is not an operation the API offers.

- `init`, `create`, `update`
- `status`, `claim`, `release`
- `link` and `unlink` for dependencies and references
- `ac` and `dod` to add, check, uncheck, and remove criteria items. Every
  operation repeats and they combine in one call, which lands as one write or
  none of it. Every index means the item the caller read when they typed it:
  checks and unchecks move nothing, removals apply highest first so the lower
  indexes still point where they did, and adds append last. A removal index
  named twice is applied once, because it names one item
- `note` and `comment` to append
- `plan` and `summary` to set. Each replaces rather than appends, because each
  is one statement rather than a log: a plan says how the work will go and a
  summary says where it landed. A log of either is what `Notes` and `Comments`
  already are
- `archive`, `unarchive`

Each returns the resulting ticket, its new revision, and the paths changed.

### 9.1 Removing a ticket filed by mistake

The case is concrete. `parseBody` splits a body section on any line opening with
`## `, per 5.2, so a description written with Markdown subheadings lands partly
in `body.extra`. The write succeeds, `check` passes, and `update --description`
replaces the description alone, so the stray sections survive and repairing in
place duplicates the content. `TKT-01M1HVMQ` was filed that way twice for
exactly this reason, and the repair on record is to delete the file and file it
again. So the tool documents a repair it cannot perform.

`rm` is not the problem, which is the finding that shapes the rest. Deleting an
unreferenced ticket file leaves `check --strict` green, so for the case that
motivates this the shell already does the whole job and a command wrapping one
`unlink` would add nothing.

What `rm` cannot do is say when it is wrong. Deleting a ticket that another
names in `dependencies` or `parent` leaves `dependency_missing` or
`parent_missing`, and `check --fix` declines both, because choosing between
unlinking the dependent, re-pointing it, and re-creating the target is the same
judgement it declines for `duplicate_id`. The refusal is therefore the product,
and the deletion is the part that was never hard.

`remove ID` refuses two things and deletes in every other case:

| Refusal | Code | Why |
|---|---|---|
| another ticket names it in `dependencies` or `parent` | `ticket_referenced` | the store fails `check` afterwards and `--fix` cannot repair it |
| it carries `Notes`, `Comments`, or a `Summary`, holds a claim, or is archived | `ticket_touched` | somebody wrote those after filing, and `archive` is the operation for work that happened |

Both name what they found. The first names the tickets that point at it, which
is the whole of what it buys over `rm`: finding that out by hand means grepping
the store for an ID.

A ticket can trip both. `ticket_referenced` is the one reported, because it is
the one `check --fix` cannot repair, and an `Error` carries one code.

A ticket that trips neither is removed without ceremony. That is the "filed by
mistake, nobody has touched it" case, and it is checkable rather than a
judgement: `Notes`, `Comments`, and `Summary` are empty, and `claim` and
`archive` are null.

A plan, acceptance criteria, and a definition of done do not count as touching
it. `create` seeds all three, per 12.1, so a ticket filed carefully carries them
from its first revision, and refusing on them would refuse exactly the mistakes
that were filed best.

`--force` overrides both and reports every dangling reference it created, naming
`unlink` as the repair. It overrides rather than refusing absolutely because a
refusal a person routes around with `rm` has taught them nothing, and the tool's
opinion is lost exactly where it would have mattered.

It is its own command rather than a flag or a status. It is not a `Mutation`,
because every mutation rewrites a ticket in place and this deletes the file, so
it sits on the `Store` beside `Create`. It is not a status, because every status
in 6.1 describes a ticket that still exists. And it is not a flag on `archive`,
which means keep. It is also the one operation in this section that returns no
resulting ticket, so the sentence above it does not hold: `remove` returns the
ticket it removed, which is the last state rather than the new one.

It writes no history. Section 7.4 lists every Git command this code runs and
none of them touches the index or a commit, so `remove` deletes the working-tree
file and a person stages that deletion like any other change. That is also the
undo, because the file is in Git until the removal is committed. Returning the
removed ticket serves the same end for a caller that would rather not go to Git
for it.

Removing an epic leaves `epics.md` stale. That is the `epics_index_stale`
warning `check --fix` already repairs, and `remove` does not rewrite the index
itself, for the reason `check --fix` does one job.

## 10. JSON contract

Every machine-readable operation emits a versioned envelope on stdout:

```json
{ "schemaVersion": 1, "kind": "ticket-list", "tickets": [], "unreadable": [] }
```

Kinds are `ticket`, `ticket-list`, `mutation-result`, `check-report`, `error`,
`schema`, `config`, `instructions`, and `version`. Absent scalars are `null` and
absent collections are `[]`, always present rather than omitted, so a consumer
never has to distinguish missing from empty.

A mutation result:

```json
{
  "schemaVersion": 1,
  "kind": "mutation-result",
  "ticket": { "id": "TKT-…", "revision": "sha256:…" },
  "pathsChanged": [".tickets/tickets/TKT-….md"]
}
```

`remove` emits this kind too, carrying the id and the revision the ticket had
and the path it no longer occupies. It is a write like any other and reports
like one.

The `ticket` stub holds an identity rather than a body because a caller wanting
the whole thing asks `show` for it. `remove` is the one command that breaks that
reasoning, since `show` afterwards is `ticket_not_found`. It keeps the stub
regardless. A caller needing the body reads it before removing, the library hands
`Store.Remove` callers the removed ticket in full, and Git has it whenever the
file was committed. Adding a key that carries the body is additive under 12.4
and can happen when a consumer needs one, while removing a key that shipped is a
break, so the smaller envelope is the reversible way round.

That leaves one sharp edge worth stating: a ticket created and removed without an
intervening commit is gone, because Git never saw it. This is the ordinary case
rather than the exotic one, since the mis-sectioned ticket of 9.1 is usually
fixed in the session that filed it. What survives is what the caller already had,
which is the text it meant to write.

An error leaves stdout empty in human mode and writes the envelope to stdout in
`--json` mode, with the message on stderr in both, and exits nonzero:

```json
{
  "schemaVersion": 1,
  "kind": "error",
  "error": {
    "code": "stale_revision",
    "message": "ticket changed since it was read",
    "details": { "expected": "sha256:…", "actual": "sha256:…" }
  }
}
```

A warning is not an error and does not behave like one. It goes to stderr in
both modes, never touches stdout, and moves no exit status, so a caller parsing
the envelope never has to know one was printed.

Two exist today. The first is text written into a body section carrying a line
5.2 would read as the start of another. The write still happens, because passing
several sections in one string works and is sometimes deliberate, and refusing
would break a path that functions in order to prevent a mistake. The warning
names the heading it found and `###` as the fix.

The second is a write that names no actor and lands on the first entry in
`actors`, per 4.1. It names the actor it used and `defaults.actor` as the way to
make that deliberate. A store declaring one is silent, and a store with no actor
at all is refused rather than warned.

Stable codes, which callers may switch on:

`store_not_found`, `store_exists`, `ticket_not_found`, `ambiguous_id`,
`stale_revision`, `invalid_transition`, `invalid_field`, `dependency_missing`,
`dependency_cycle`, `claim_conflict`, `ticket_referenced`, `ticket_touched`,
`parse_error`, `merge_conflict`, `schema_unsupported`, `lock_timeout`,
`validation_failed`, `usage`.

The last two are `remove`'s, per 9.1. They are their own codes rather than a
`validation_failed` apiece because the repairs differ, so a caller that reads
the code knows which to do: `unlink` the tickets that point at it, or `archive`
it instead. `claim_conflict` is the precedent, a refusal belonging to one
command and overridden by the same `--force`.

`usage` is the CLI's own: an unknown command, a missing argument, or a flag
value outside its set. It never comes from the library, which is why it names no
store condition.

Every path in the envelope is relative to the repository root when the store
sits inside one, and absolute otherwise. That covers `pathsChanged`, the `path`
of a ticket, and the `file` of a check finding.

The library reports a finding's file relative to the store instead, because a
`Report` describes a store and may be produced where no repository root is
known. The fixture sidecars in `testdata/` record that store-relative form. The
CLI converts on the way out, so a consumer sees one path convention across every
kind rather than having to know which layer produced the value.

### 10.1 A ticket

The `ticket` kind carries one ticket under a `ticket` key, and `ticket-list`
carries an array of the same objects under `tickets`.

`ticket-list` also carries `unreadable`, the ticket files the query had to leave
out because they did not parse. Section 8 says a query skips those files and
`check` reports them, which leaves a host unable to tell a short listing from a
complete one. The entries have the same four keys a `check` finding has, built
the same way, so a consumer parses one shape whichever command it called. Every
command that answers with `ticket-list` carries it, because every one of them
reads the whole store.

Human output does not show it. A person who wants to know what is broken runs
`check`, which is the command for that and reports far more than this. The
channel exists for a host that cannot run a second command and reconcile two
answers.

```json
{
  "id": "TKT-01K3ZYG8K0Y52AD43XRGM4T7WZ",
  "revision": "sha256:…",
  "path": ".tickets/tickets/TKT-01K3ZYG8K0Y52AD43XRGM4T7WZ.md",
  "branch": null,
  "schema": 1,
  "title": "Rotate the signing key without downtime",
  "type": "epic",
  "status": "in-progress",
  "statusReason": null,
  "priority": "urgent",
  "dueOn": "2026-10-14",
  "labels": ["auth"],
  "assignees": ["human:sothr"],
  "milestone": "v1.2",
  "parent": null,
  "dependencies": [],
  "blocksOn": "none",
  "references": [{ "ref": "proposal:git-ticket", "path": "docs/plan.md" }],
  "claim": null,
  "archive": null,
  "createdAt": "2026-08-31T12:00:00Z",
  "updatedAt": "2026-08-31T12:06:00Z",
  "createdBy": { "id": "human:sothr", "name": "Drew Short" },
  "updatedBy": { "id": "agent:terva/session-123", "name": "Mieli" },
  "extensions": {},
  "unknown": {},
  "body": {
    "description": "The signing key has never been rotated…",
    "acceptanceCriteria": "- [x] The verifier accepts either key\n- [ ] New tokens use the newer key",
    "definitionOfDone": "",
    "implementationPlan": "",
    "notes": "",
    "comments": "**human:sothr** at 2026-08-31T12:05:00Z\n\nSecond pair of eyes wanted.",
    "summary": "",
    "extra": [{ "heading": "Risks", "text": "…" }]
  },
  "checklists": {
    "acceptanceCriteria": [
      { "index": 1, "checked": true, "text": "The verifier accepts either key" },
      { "index": 2, "checked": false, "text": "New tokens use the newer key" }
    ],
    "definitionOfDone": []
  },
  "comments": [
    {
      "index": 1,
      "actor": "human:sothr",
      "at": "2026-08-31T12:05:00Z",
      "text": "Second pair of eyes wanted."
    }
  ],
  "readiness": {
    "isReady": false,
    "isBlocked": true,
    "blockingDependencies": ["TKT-01K4001C…"],
    "missingDependencies": [],
    "blockingChildren": [],
    "reason": "waiting_on_dependencies"
  }
}
```

The frontmatter fields keep their names in camel case. `revision`, `path`,
`branch`, and `readiness` are computed rather than stored, per 7.1 and section 8.
`unknown` holds the top-level fields this version does not define, per 5.4, which
are preserved on write whether or not a consumer understands them.

`branch` is the ref a row's winning copy came from, per section 8, and it is null
when that copy is the working tree. Every row of an ordinary query carries null,
because without `--cross-branch` there is nowhere else for a row to come from.

When `branch` is not null, `path` is the file's path on that ref rather than a
path in the caller's working tree, and opening it will fail. That pairing is why
the field exists: a row naming a ticket the caller cannot open has to say so,
and a consumer checks one field to find out.

`body` holds every section exactly as it appears in the file, one type for all of
them, so nothing a person wrote by hand is dropped on the way out.
`checklists` is derived from `body` by reading the `- [ ]` and `- [x]` lines of
the two checkbox sections. The derivation runs one way only: `body` is the
document and `checklists` is a view of it, so the two cannot disagree and there
is no question which one wins.

`index` is the number `ac` and `dod` take. It counts from one, over checkbox
lines only, so a consumer never computes it from an array position. A checklist
section that also holds prose keeps that prose in `body` and leaves the indexes
unmoved.

The top-level `comments` is the other derived view, and runs the same way: it
reads the stamps `comment` writes out of `body.comments`, which stays the whole
section verbatim. A consumer draws a thread without parsing Markdown.

An entry begins at a stamp and runs to the next one. A blank line inside an
entry does not end it, because a comment may have several paragraphs. Prose with
no stamp above it comes back as an entry with `actor` and `at` null rather than
being dropped, because a ticket is a file a person edits and text somebody typed
is still a comment they left. The consequence is that prose appended below a
stamped entry joins that entry and reads as its author's. Splitting on the blank
line would not fix the attribution, since the fragment sits under that stamp
either way, and it would take every multi-paragraph comment apart.

`notes` uses the same stamp, so the same reading applies to it. Only `comments`
is carried in the contract, because a note is written for the ticket and a
comment is written to somebody.

### 10.2 Exit statuses

Zero when the command did what it was asked, and one otherwise. A check that
found errors, an unknown flag, and a stale revision all exit one. An exit status
has room for one bit, and the codes above have more to say, so a caller that
needs to tell those apart reads the code from the error envelope.

`check --strict` promotes warnings to errors, so a store carrying only warnings
exits zero without it and one with it.

One graded bucket is reserved on top of that rule: 10 through 12 mean "an
update is available", and only the no-op modes of `self-update` use them, per
12.6. 10 is a patch-sized gap, 11 minor, 12 major, so
`self-update --check || notify` works in a cron job with no parsing, and a
script that applies patches automatically but waits on anything larger
compares against one number. The precedent is the reserved informational
bucket of zypper and terraform's `-detailed-exitcode`, not fsck's bitmask: a
version gap is one category, the highest component that moved, so a grade
carries everything a mask would and stays readable in a shell comparison.
Under v0.x this composes with 12.4: a minor is where breaks land, so tooling
that waits on 11 gets exactly the caution that numbering promises.

### 10.3 A check report

The `check-report` kind carries the findings of section 11:

```json
{
  "schemaVersion": 1,
  "kind": "check-report",
  "ok": false,
  "errors": [
    {
      "code": "dependency_missing",
      "file": ".tickets/tickets/TKT-01K3ZZ67Q0PT427VFD1F4WFWSH.md",
      "ticket": "TKT-01K3ZZ67Q0PT427VFD1F4WFWSH",
      "field": "dependencies"
    }
  ],
  "warnings": [],
  "repairs": [],
  "pathsChanged": [],
  "dryRun": false
}
```

A finding carries `code`, `file`, `ticket`, and `field`. `ticket` is null when
the file did not parse far enough to know its ID, and `field` is null when the
finding is about the file rather than one field. Findings are ordered by file,
then code, then field, so two reports of the same store compare directly instead
of having to be treated as sets.

`repairs`, `pathsChanged`, and `dryRun` describe a `--fix` pass, per 12.1. They
are present on every report, empty and false without the flag, because an absent
collection is an empty array and never omitted. A repair names the findings it
cleared as a list:

```json
{
  "kind": "move",
  "codes": ["filename_id_mismatch"],
  "ticket": "TKT-01K3ZZ2JH000GHB4EE6SNRE6MD",
  "from": ".tickets/tickets/notes-about-auth.md",
  "to": ".tickets/tickets/TKT-01K3ZZ2JH000GHB4EE6SNRE6MD.md"
}
```

`codes` is a list because a file in the wrong directory under the wrong name
raises both findings and one move settles them together.

`kind` is `move` or `rewrite`. Every repair was a move until `epics.md` arrived,
and rewriting a generated file is not a move: there is no old path, and the file
is not a ticket. A rewrite nulls `from` and `ticket` and names the file in `to`:

```json
{
  "kind": "rewrite",
  "codes": ["epics_index_stale"],
  "ticket": null,
  "from": null,
  "to": ".tickets/epics.md"
}
```

Null rather than an empty string, because this contract already says an absent
value is null and a finding already nulls `ticket` and `field` when they do not
apply. `kind` exists so a consumer switches on a field rather than inferring
from which values came back null.

`pathsChanged` names both ends of every move and the single file of every
rewrite, the way a mutation names what it touched. It is empty under `--dry-run`,
because nothing was written. `repairs` is not, so a dry run still says what it
would do. The findings a dry run would have cleared are still in `errors` or
`warnings`, since the store still has them.

`ok` mirrors the exit status: it is true exactly when the command exited zero.
A caller therefore gates on one field, and never has to reconstruct the verdict
from the arrays and the flags it passed.

`--strict` does not move a finding. The two arrays report severity as section 11
defines it, whatever the caller asked for, and strictness is a policy on top
that shows up in `ok` and the exit status alone. So `ok` may be false with an
empty `errors` array, and that is the strict run of a store carrying warnings.

A store with findings is a successful check, not a failed command. The report
goes to stdout and the exit status is one. An `error` envelope comes back only
when the check could not run at all, such as `store_not_found`.

### 10.4 The schema kind

`schema` prints what this binary enforces, so a consumer can learn the legal
values without reading this document or hard-coding them:

```json
{
  "schemaVersion": 1,
  "kind": "schema",
  "ticketSchema": 1,
  "kinds": ["ticket", "ticket-list", "mutation-result", "check-report", "error", "schema", "config", "instructions", "version"],
  "statuses": ["draft", "ready", "in-progress", "blocked", "review", "done", "archived"],
  "openStatuses": ["draft", "ready", "in-progress", "blocked", "review"],
  "types": ["task", "bug", "chore", "spike", "epic"],
  "priorities": ["low", "normal", "high", "urgent"],
  "unreadyReasons": ["draft", "in-progress", "blocked", "review", "done", "archived", "waiting_on_dependencies", "claimed"],
  "titleLimits": { "warn": 72, "max": 120 },
  "transitions": { "draft": ["ready", "archived"] },
  "errorCodes": ["store_not_found", "usage"],
  "findingCodes": [{ "code": "duplicate_id", "severity": "error" }]
}
```

`schemaVersion` is the envelope version and `ticketSchema` is the `schema` field
of a ticket file, per 5.1. They are separate numbers because the envelope and
the file format can move independently.

`transitions` has one entry per status, holding where a ticket in that status
may go, which is the table in 6.2. `errorCodes` is the section 10 list with the
CLI's `usage` appended. `findingCodes` pairs each section 11 code with the
severity that section assigns it, so a consumer reading a report knows whether
a code it has never seen is an error or a warning.

`openStatuses` is what a listing answers with by default, per section 8. It is
published for the same reason the rest of this envelope is: a consumer that
wants the open set otherwise hard-codes five strings and goes wrong the day a
sixth status arrives. It is derived from `statuses` by removing the terminal
ones, so the two cannot drift.

`titleLimits` publishes the two thresholds of 5.1 so a caller composing a title
knows where the line is before it writes one. `warn` is where `title_long`
starts and `max` is where a write is refused, which is the number that matters
to a writer. Both count characters rather than bytes.

`unreadyReasons` is every value `readiness.reason` can carry, per section 8, so
a consumer switching on it does not hard-code the list and fall through the day
a status arrives. It is derived the same way: `statuses` without `ready`, then
`waiting_on_dependencies` and `claimed`. The empty string a ready ticket carries
is not in it, because that is the absence of a reason rather than one of them.

Every one of those values is read from the code that enforces it rather than
copied into the command. A status the library accepts and this document forgot
still appears here, which makes `schema` the answer of record when the two
disagree.

This command reads no store. It answers outside a repository and before `init`,
because a consumer asks what is legal before it has anything to ask about.

That is also why the label and milestone allowlists are not here. They are
per-store rather than facts about the binary, so they would make this answer
depend on where it ran. `config`, in 10.6, is where they live.

### 10.5 The instructions kind

`instructions` carries the agent workflow block of 12.1 as one string:

```json
{ "schemaVersion": 1, "kind": "instructions", "text": "<!-- git-ticket:begin -->\n\n## Tickets\n\n…" }
```

The block is prose, so the envelope holds it whole rather than pretending it has
structure a consumer would want to walk. `text` carries the markers of 12.1,
because what a consumer pastes has to be what a later `--write` can find again.
In human mode the command prints the Markdown alone, so it can be redirected or
read.

`--write` puts it in `AGENTS.md` instead, per 12.1, and reports what it did as a
`mutation-result` whose `pathsChanged` is empty when the file was already
current. That is the kind for a command that changed files, and this one did.

Like `schema`, it reads no store and answers anywhere.

### 10.6 The config kind

`config` prints what this store configured, which is the half of the vocabulary
`schema` cannot answer for:

```json
{
  "schemaVersion": 1,
  "kind": "config",
  "ticketSchema": 1,
  "labels": { "values": ["ci", "format", "release"], "enforced": true },
  "milestones": { "values": [], "enforced": false },
  "actors": [{ "id": "human:sothr", "name": "Drew Short" }],
  "defaults": { "type": "task", "priority": "normal", "actor": null, "claimExpiry": null },
  "lock": { "timeout": "10s" }
}
```

It is a separate kind rather than more keys on `schema` because of the last line
of 10.4. `schema` reads no store, so it answers before `init` and outside a
repository, and that is the property that lets a consumer ask what is legal
before it has anything to ask about. Labels and milestones are per-store, so
putting them in that envelope would make the same command answer differently
depending on where it ran, and would cost `schema` the one guarantee that makes
it useful early. The split is the whole design: `schema` is what the binary
enforces and is identical everywhere, `config` is what this store chose.

This command reads a store and needs one, which is what separates it from the
other two commands that answer a question rather than reporting on tickets.

An allowlist is an object and not a bare list, because the list alone is
ambiguous in the direction that matters. Per 4.1 an empty allowlist permits
everything, so a consumer handed `"labels": []` and reading it the obvious way
concludes that no label is permitted, which is exactly backwards, and the
mistake is silent. `enforced` names the regime, and a consumer that reads only
that key is still right.

`enforced` is derived from the length of `values` rather than stored, so the two
cannot drift, for the reason `openStatuses` is derived from `statuses`.

There is no way to say "empty and enforcing". A store that never listed a label
and a store that listed none are the same state, because 4.1 gives an empty list
the meaning "this store has not expressed an opinion" rather than "this store
permits nothing". This envelope reports which regime is in force instead of
inventing a distinction the format does not carry.

Publishing an allowlist does not make it stricter. It stays advisory, per 4.1: a
write naming a label outside the list succeeds, and `check` reports
`label_unknown` as a warning, so it is `check --strict` where an unlisted label
fails a build. Anything else is a change to enforcement rather than to what is
published, and belongs in its own decision.

`actors`, `defaults`, and `lock` are here for the same reason the allowlists
are. Each is per-store, each changes what a caller should send, and none of them
was reachable except by reading `config.yml`. `defaults` says what `create` uses
when the caller names no type or priority, and `claimExpiry` is null when a
claim does not expire on its own.

### 10.7 The self-update kind

`self-update` answers with what it found and what it did:

```json
{
  "schemaVersion": 1,
  "kind": "self-update",
  "current": "v0.7.1",
  "latest": "v0.8.0",
  "gap": "minor",
  "asset": "git-ticket_0.8.0_linux_amd64.tar.gz",
  "target": "/home/user/go/bin/git-ticket",
  "applied": false
}
```

`gap` is `none`, `patch`, `minor`, or `major`: the highest version component
that differs between `current` and `latest`. `asset` and `target` are null
under `--check`, which stops before resolving either; `--dry-run` fills them.
`applied` is true only when the binary on disk was actually replaced, so a
no-op mode can never answer true. The exit status carries the same verdict as
`gap`, per the graded bucket in 10.2, and the envelope exists so a program
that wants more than one number does not parse prose.

Like `schema`, this command reads no store and answers outside a repository,
because the binary being out of date is not a property of any store.

## 11. Validation

`check` runs offline, is safe in CI, and separates errors from warnings. It
exits nonzero on any error, and `--strict` promotes warnings to errors.

Every finding carries a stable code, so a caller switches on the code instead of
matching a message. These codes overlap the operation codes in section 10 only
where the condition is the same one: `parse_error`, `merge_conflict`,
`schema_unsupported`, `dependency_missing`, and `dependency_cycle`. The operation
code `invalid_field` does not appear here, because a report says which field is
wrong rather than that some field is.

Errors:

| Code | Condition |
|---|---|
| `duplicate_id` | the same `id` appears in more than one file |
| `filename_id_mismatch` | the filename is not `<id>.md` |
| `parse_error` | malformed frontmatter or an unparseable body |
| `unknown_field` | an unknown top-level frontmatter field |
| `schema_unsupported` | `schema` is newer than this binary supports |
| `merge_conflict` | Git conflict markers in a ticket file, reported as this rather than as a YAML parse failure, because that is what a user needs to be told |
| `dependency_missing` | a `dependencies` entry names a ticket that does not exist |
| `parent_missing` | `parent` names a ticket that does not exist |
| `dependency_cycle` | a cycle in `dependencies` |
| `parent_cycle` | a cycle in `parent`, checked separately from dependencies |
| `blocking_cycle` | a cycle in the blocking graph that needs a child edge to close, so neither `dependency_cycle` nor `parent_cycle` sees it |
| `invalid_status` | a `status` outside the set in 6.1 |
| `invalid_type` | a `type` outside the set in 5.1 |
| `invalid_priority` | a `priority` outside the set in 5.1 |
| `invalid_blocks_on` | a `blocks_on` outside the set in 5.1 |
| `invalid_due_on` | a `due_on` that is not a `YYYY-MM-DD` date, per 5.1. A date that has passed is never a finding |
| `title_too_long` | `title` is longer than 120 characters, per 5.1 |
| `location_mismatch` | the status and the directory disagree, per section 4; the status wins |

Warnings:

| Code | Condition |
|---|---|
| `dependency_archived_incomplete` | a live ticket depends on an archived ticket whose `from_status` is not `done` |
| `claim_expired` | a claim is past its `expires_at` |
| `reference_path_unresolved` | a `references` path does not resolve against the repository root, per 5.5 |
| `reference_untyped` | a `references` ref carries no namespace, so it is not the typed stable identifier 5.1 describes and no lookup by namespace finds it |
| `label_unknown` | a label is outside the `config.yml` allowlist |
| `milestone_unknown` | `milestone` is outside the `config.yml` allowlist |
| `in_progress_unclaimed` | a ticket is `in-progress` with no claim |
| `blocks_on_no_children` | `blocks_on` is `children` and no ticket names this one as its parent |
| `title_long` | `title` is longer than 72 characters, per 5.1 |
| `epics_index_stale` | `epics.md` disagrees with the epics in the store, per section 4 |

A finding names the file, and the ticket ID and field where they apply. A file
that fails to parse yields exactly one finding, because everything downstream of
a parse failure would be noise.

Severity belongs to the code, not to the condition that raised it. A caller
reading a report has only the code to go on, so `unknown_field` is an error
wherever `check` finds it. `git ticket schema` publishes this split, per 10.4,
and a test holds the published lists to what `check` emits over the fixture
corpus so the two cannot drift.

`reference_path_unresolved` is the one check that depends on where the store
sits. A store outside a Git repository has no root to resolve against, so the
check is skipped there and reports nothing, per 5.5.

Three findings have exactly one correct repair, and `check --fix` makes them:

| Code | The repair |
|---|---|
| `filename_id_mismatch` | rename the file to `<id>.md`, which section 4 fixes and leaves no second reading of |
| `location_mismatch` | move the file to the directory the status implies, because 6.3 already rules the status wins |
| `epics_index_stale` | rewrite `epics.md` from the tickets, which are the source it is derived from |

Severity does not gate repair. `epics_index_stale` is a warning and the other two
are errors, and all three are repaired, because the repair pass recomputes what
every file should be rather than walking the findings. A warning is therefore
exactly as repairable as an error, which is what makes keeping this one a warning
free rather than merely defensible.

Nothing else is repaired, and the rest are not near misses. `duplicate_id` has
to choose which file keeps the ID, which is a judgement about which ticket is
the real one. `dependency_missing` is repaired either by dropping the edge or by
creating the ticket, and only a person knows which was meant. `label_unknown`
and `milestone_unknown` are each either a typo in the ticket or a gap in the
allowlist. A tool that guessed at those would be wrong about half of them and
silent about it, which is worse than reporting and stopping.

### Verifying generated artifacts in CI

A store holds generated state: `epics.md`, and the directory and filename every
ticket should sit under. CI verifies both and commits neither.

The verification command is `check --fix --dry-run --strict`. It plans every
repair, prints what it would do, writes nothing, and exits 1 when anything is
pending.

`--strict` is the flag doing the work, and leaving it off is the mistake worth
naming here. `epics_index_stale` is a warning, so `check --fix --dry-run` alone
exits 0 on a stale index and a job built on it goes green over the exact
condition it was added to catch. An error such as `filename_id_mismatch` exits 1
either way, which is what makes the gap easy to miss: the command looks correct
until the first warning-severity artifact arrives, and then it is silently
wrong rather than broken.

The obvious next step from "CI can detect this" is "CI should fix it and push",
and this plan refuses that step.

CI needs no write credential today. A leaked read token and a leaked write token
are not the same incident, and adding the second one buys a convenience.

A push to a pull request branch races whoever opened it. Concurrent agents in
separate worktrees are the reason this project lands work through pull requests
at all, and a job committing into a branch somebody is working in is that same
collision with one more participant that cannot be asked to wait.

A push to `main` from a job breaks the rule that nothing pushes to `main`, and
leaves every local `main` behind by a commit nobody wrote.

A push from CI also retriggers CI, so it needs a skip marker, and that is a
second mechanism to maintain for as long as the first one exists.

gofmt is the honest comparison. Nobody wants CI to reformat their code and push
it. They want CI to say the code is unformatted and one local command that fixes
it. `check --fix` is that command, and the agent workflow block names it, so an
agent reading a red check is told what to run.

Section 7.4 enumerates the Git commands this code runs and both only read. That
constrains the tool and not a workflow, because a workflow can run plain `git`
with the tool nowhere in the pipe. This section is the rule for the workflow. No
writing command joins 7.4 for this purpose, and no job commits a repair.

The two generated things sit at different scopes, and they stay under separate
commands. `epics.md` and file placement are inside `.tickets/`, which
`check --fix` owns and already walks. The agent workflow block is outside it, at
the consumer's repository root, where `instructions --write` puts it.

One flag covering both was considered and declined. `check` is the
store-integrity command, and a consumer's `AGENTS.md` is a prose file that
project owns. A tool that rewrites it as a side effect of a verification will
eventually clobber something somebody wrote by hand, and the flag that did it
would read as harmless. A workflow verifying both therefore runs two commands.
That is the smaller price.

## 12. Interfaces

### 12.1 CLI

```text
git ticket init   [--instructions]
git ticket install-merge-driver
git ticket merge-driver BASE OURS THEIRS
git ticket list   [--status S --type T --priority P --label L --assignee A --milestone M --parent P --due-by DATE --sort id|due_on|priority]
git ticket ready
git ticket ui       # browse the store interactively; no --json form
git ticket show   ID
git ticket search QUERY [--regex]
git ticket create --title T [--type --priority --label --assignee --milestone --parent --blocks-on --due-on --depends-on --description --description-file --plan --plan-file --ac --dod]
git ticket update ID [--title --type --priority --description --description-file --milestone --parent --blocks-on --due-on --add-label --remove-label --assign --unassign]
git ticket status ID STATUS [--reason R]
git ticket claim  ID [--expires-in D] [--force]
git ticket release ID
git ticket link   ID [--depends-on OTHER | --ref proposal:x [--path P]]
git ticket unlink ID [--depends-on OTHER | --ref proposal:x]
git ticket ac     ID [--add TEXT] [--check N] [--uncheck N] [--remove N]
git ticket dod    ID [--add TEXT] [--check N] [--uncheck N] [--remove N]
git ticket plan   ID (TEXT | --file PATH)
git ticket note   ID (TEXT | --file PATH)
git ticket comment ID (TEXT | --file PATH)
git ticket summary ID (TEXT | --file PATH)
git ticket deps   ID [--transitive] [--dependents]
git ticket files  PATH   # the tickets that reference a path
git ticket refs   REF    # the tickets carrying a ref, whole or a bare namespace
git ticket check  [--strict] [--fix [--dry-run]]
git ticket archive ID [--reason R]
git ticket unarchive ID
git ticket remove ID [--force]   # delete a ticket filed by mistake, per 9.1
git ticket migrate [--to N] [--dry-run]
git ticket instructions [--write]
git ticket schema
git ticket config   # what this store configured, including the allowlists
git ticket self-update [--check | --dry-run]   # replace this binary with the latest release, per 12.6
```

Global flags: `--json`, `--store PATH`, `--if-revision R`, `--actor ID`,
`--lock-timeout D`. They are accepted before or after the subcommand, because
`git ticket --json list` and `git ticket list --json` are both what a person
types.

`--version` prints what the binary is and exits. It is top level only, and a
subcommand refuses it, because a flag a command accepts and then ignores is
worse than one it rejects. Nothing is stamped at link time. `go build` already
records the module version, the commit, and whether the tree was dirty, and
`runtime/debug` reads them back, so there is no ldflags recipe and no version
variable to keep in step with a tag. A build with no tag reachable says `devel`.
It needs no store, because it describes the binary and not a ledger.

Every section of prose can be read from a file instead of an argument. The
format is Markdown, and a paragraph passed as one shell word makes the shell
part of the authoring surface: an apostrophe ends a single-quoted string and a
backtick inside a double-quoted one runs a command. Writing prose long enough to
be worth writing is the ordinary case, not the exceptional one, so it gets a
path that does not go through the shell's parser.

The shape follows how the command already takes the text. A named flag gets a
named sibling, so `--description` is joined by `--description-file` and `--plan`
by `--plan-file`. The four commands that take the text as a positional get one
`--file`, since there is only one thing it could fill. A `PATH` of `-` reads
stdin.

Giving both the text and its file is a usage error, naming both, the way
`--depends-on` with `--ref` is. A caller who typed two meant one, and picking by
precedence writes something nobody asked for. Reading stdin twice in one command
is refused for the same reason: `--description-file -` with `--plan-file -`
cannot give both flags what the caller meant, since the second read returns
nothing.

Trailing whitespace is stripped from what a file or stdin supplies. A final
newline terminates the last line of a text file rather than being content, and
`printf` and a heredoc disagree about whether to emit one. Nothing else is
touched, so indentation inside the prose survives.

The heading warning of section 10 fires the same way and names the flag that
carried the text, so `--description-file` is what a reader sees. A 5.2 heading is
more likely from a file, not less, because a file is where somebody writes enough
prose to want subheadings.

`merge-driver` is plumbing that Git invokes with the three temporary files of a
merge, rather than something a person types. Section 7.5 is what it does, and
`install-merge-driver` is the command that points Git at it.

The store is found in this order: `--store`, then `GIT_TICKET_STORE`, then
discovery walking up from the current directory to the Git root. `config.yml`
has no say, because it lives inside a store and cannot name one.

A `--store` or `GIT_TICKET_STORE` that names a directory holding no store is
`store_not_found`, and never a fall back to discovery. Searching elsewhere after
being told where to look is how a tool writes to the wrong store.

`check --fix` repairs the two findings of section 11 that have exactly one
correct repair, and touches nothing else. It is the first command that writes
without being told which ticket to write, so `--dry-run` reports the moves and
makes none, and the report names every path it touched the way a mutation does.

The repair moves the file and does not re-render the ticket. Both findings are
about where a file sits, so a pass that also rewrote contents would be doing
something the caller did not ask for.

A move onto a path that already exists does not happen, and neither does one
where two files want the same destination. Both mean a second ticket is
involved, and which of them is the real one is the `duplicate_id` judgement this
deliberately declines to make. The repair is dropped and the finding stays, so
the store still says what is wrong.

The store lock is held across the whole pass, planning through the re-check, so
the report describes the store the repairs left rather than one somebody else
has changed since.

`instructions` prints an agent workflow block for pasting into a project's
`AGENTS.md` or equivalent, per 10.5. The block tells an agent how to find work,
claim it, record what it learned, and finish, and it names only commands this
binary has. A test holds it to that, because prose that tells a reader to run
something that does not exist is worse than no prose.

It opens by telling an agent to pass `--actor` on every write, as
`agent:tool/session`. That is the block's first instruction because it is the
one whose absence is silent: with no flag the store falls back to the first
actor in `config.yml`, per 4.1, which is usually a person, so an agent's notes
arrive under their name and its claim reports a human holding the ticket. The
record is wrong and nothing downstream says so. Nor does committing repair it,
for the reason 5.3 gives: a commit carries the committer's Git identity while an
actor is a session, so a commit collapses every actor in the store into one
person. The provenance pair is only worth storing if the writes that fill it say
who was writing.

It also carries one rule about writing rather than about running: an agent
naming a ticket in prose writes the title beside the ID the first time, as
`TKT-01M1PQ7T (Build git ticket remove, per plan 9.1)`. Later mentions in the
same piece of writing use the bare ID, because the point is orienting a reader
once and not padding every line.

This is in the block rather than left to each project because the cost lands on
whoever reads the agent, not on the agent. A ULID is unreadable by design, so a
summary quoting three bare IDs asks a person to look up three tickets to learn
what it says. The title in 5.1 is bounded for this: it has to fit inline for
this rule to be worth following.

The block is fenced by two markers, so it can be replaced later without
disturbing what a person wrote around it:

```markdown
<!-- git-ticket:begin -->

## Tickets
…

<!-- git-ticket:end -->
```

The markers are part of the block itself rather than added by the writer, so
every way it leaves the binary carries them: stdout, the `instructions` kind of
10.5, and the file. A block somebody pasted by hand is refreshable for that
reason. An HTML comment is the form because it renders as nothing, so it does
not clutter a file a person reads and edits.

`instructions --write` puts the block in `AGENTS.md` at the repository root, or
in the working directory when there is no repository, since the command answers
anywhere. What it does depends on what it finds:

| The file | What happens |
|---|---|
| is not there | it is created holding the block |
| carries both markers | the text between them is replaced and every other byte is left alone |
| carries neither marker | the block is appended, and the file is refreshable from then on |
| is already current | nothing is written |
| carries anything else | it refuses |

That last row is one marker without its partner, a pair out of order, or a
second copy of either. None of them leaves an honest reading of where the block
ends, and the wrong guess deletes prose somebody wrote. A refusal costs a person
one edit, so it refuses and names what to fix.

Writing nothing when the file is already current keeps a no-op out of a diff. A
command that reports a change with nothing in it is one nobody runs.

`init --instructions` does the same write, so a project that already has an
`AGENTS.md` gets the block appended rather than being told to paste it. It
checks the file before creating the store, so a refusal leaves nothing
half-built. Without the flag `init` writes no such file. That file is one the
user maintains, and everything outside the markers stays theirs.

`schema` prints the values and codes this binary enforces, per 10.4. Like
`instructions`, it reads no store and answers anywhere.

### 12.2 Library

The import path is `github.com/terva-sh/git-ticket/ticket`. Development happens
on the internal Forgejo and a public mirror serves that path, so the module path
names where a consumer fetches the code rather than where it is written. Never
put the internal hostname in `go.mod`: it would publish a name that resolves for
nobody outside the network, and it is the one string in the project that cannot
be corrected cheaply once anything imports it.

The API terva and any other consumer builds against:

```go
package ticket

func Discover(dir string) (*Store, error)
func Open(path string) (*Store, error)
func Init(root string, opts InitOptions) (*Store, error)

func (s *Store) Get(ctx context.Context, ref string) (*Ticket, error)
func (s *Store) List(ctx context.Context, f Filter) ([]*Ticket, error)
func (s *Store) Search(ctx context.Context, q Query) ([]*Ticket, error)
func (s *Store) Ready(ctx context.Context) ([]*Ticket, error)
func (s *Store) Readiness(ctx context.Context) (map[string]Readiness, error)
func (s *Store) Check(ctx context.Context) (*Report, error)
func (s *Store) Fix(ctx context.Context, o FixOptions) (*FixResult, error)
func (s *Store) Apply(ctx context.Context, ref string, m Mutation, o ApplyOptions) (*Result, error)

type ApplyOptions struct {
    IfRevision string // empty means no precondition
    Actor      Actor
}

type Result struct {
    Ticket       *Ticket
    PathsChanged []string
}
```

`Mutation` is a typed set of operations rather than a struct of pointers, so
"set the title to empty" and "do not touch the title" cannot be confused.

A second package, `github.com/terva-sh/git-ticket/cli`, exports the whole
command surface for a host that wants the commands rather than the library:

```go
package cli

func Run(args []string, env Env) int

type Env struct {
    Dir    string
    Getenv func(string) string
    Stdout io.Writer
    Stderr io.Writer
    Now    func() time.Time // nil means the real clock
}
```

That is the entire API: two identifiers behind every command in 12.1, both
output modes, and the exit statuses in 10.2. `cmd/git-ticket/main.go` is the
reference caller and does nothing else, so a host embedding the commands runs
the same code path a person at a terminal does.

It is exported rather than internal because of what the alternative costs. A
host that cannot import this has to write flag parsing, rendering, and error
mapping over `ticket` a second time, which is the second parser 12.1 exists to
prevent, and it drifts from the day it is written. Terva is the first such host
and the reason this moved; a shell alias or another agent harness gets the same
benefit. `Env` was already fully injectable before the move, because the tests
drive `Run` through it.

The two packages answer different questions and a caller usually wants one of
them, not both. Use `cli` for a command surface, where the result is rendered
text and an exit status. Use `ticket` for structured values, which is what a
tool or a UI needs. Terva does both: `terva ticket` calls `cli.Run`, and the
`ticket_*` tools call the store directly.

Package layout:

```text
git-ticket/
├── cmd/git-ticket/     the binary, and the reference caller for cli
├── ticket/             the public library
├── cli/                the public command surface: flag parsing and rendering
├── testdata/fixtures/  the corpus from Phase 0
└── docs/plan.md
```

The library must not import a CLI framework, and nothing in `ticket/` may import
terva. `cli` may import `ticket`, and never the reverse.

### 12.3 Optional stdio adapter

`git-ticket mcp` speaks MCP over stdio against the same library. It is a
process-local adapter, not a service. It performs no permission decisions of its
own; the host decides what a caller may do.

### 12.4 Compatibility

Three numbers move here, and treating them as one is the easiest mistake to
make:

| Number | Where it lives | What it tracks |
|---|---|---|
| the module version | Git tags, and a consumer's `go.mod` | the Go API of `ticket` and `cli` |
| `schema` | the frontmatter of every ticket, and `ticket.SchemaVersion` | the on-disk file format |
| `schemaVersion` | the JSON envelope, published by `git ticket schema` | the machine-readable output |

They are independent. Section 10.4 already says the last two move separately,
and the module version moves separately from both.

Everything a machine reads is covered by the compatibility promise: the `ticket`
package API, `cli.Run` and `Env`, the JSON envelope and its kinds per section
10, the exit statuses in 10.2, the error and finding codes, and the on-disk
format in sections 4 through 11. `cli` is covered for the same reason it is
exported at all, per 12.2: a command surface a host cannot rely on across a
minor release is one no host can build on.

Human-readable output is not covered. It is written for a person at a terminal
and may be reworded, recolumned, or reordered in any release. A consumer parsing
it has picked an interface this project does not offer, and `--json` exists so
that nobody needs to.

A covered surface breaks when something that worked stops working or changes
meaning. Adding beside the old thing is not a break. So a new frontmatter field,
a new error code, a new envelope kind, and a new `schema` the reader also
understands are all minor changes. Removing a field, renaming a code, changing
what a value means, and dropping support for a `schema` that used to parse are
all major ones.

Breaks are recorded here as they land, because a break nobody wrote down is
indistinguishable from a regression.

`list` and `Filter{}` narrow from every non-archived ticket to open work, per
section 8. That changes what a covered surface means rather than adding beside
it, so it is a break under the rule above. `Filter.IncludeArchived` becomes
`Filter.All` in the same change. Renaming an exported field breaks a keyed
composite literal as well as an unkeyed one, which is the wider of the two, and
is worth saying plainly to anyone who has already written `Filter{...}`.

`archive_location_mismatch` becomes `location_mismatch`, per sections 4 and 11,
because placement stopped being about the archive alone once every status
implied a directory. Renaming a code is the example the rule above gives of a
major change, and a name that misleads rots faster than one that breaks.

A repair in a check report gains `kind`, and its `ticket` and `from` become
nullable, per 10.3. Adding `kind` is additive, but a consumer that read `from` as
always being a path now has to handle null, so this is a break rather than an
addition. It arrives with the first repair that is not a move.

Both are taken now for the reason this section already gives for staying at
`v0.x`: nothing consumes these surfaces yet, so this is the cheapest either
change will ever be. Waiting does not avoid a break, it moves it to a release
where somebody has to be told.

A title over 120 characters is refused, per 5.1 and section 11. `title_long` at
72 is a warning and additive, but `title_too_long` is an error, and the library
refuses the write with `invalid_field` and exit status 1 rather than reporting
it afterwards. It refuses in the mutation path, so every command that sets a
title is covered and not `create` alone. A title that was accepted before is
rejected now, and a store already holding one fails `check` where it passed, so
this is a break on the writing side and on the reading side. The bound is
inclusive: a title of exactly 120 characters still writes.

The bound exists because the title is quoted beside the ID wherever one appears
and a ULID carries nothing a reader holds in their head, so an unbounded title
breaks the listing it sits in rather than the ticket alone. It is taken now for
the reason the two above were: nothing consumes this surface yet.

`readiness` gains `reason` and the schema kind gains `unreadyReasons`, per
section 8 and 10.4. Both are additive, so a consumer that ignores the new keys
reads exactly what it read before and this is a minor release. `Readiness` gains
an exported field, which is additive for the same reason: it breaks an unkeyed
composite literal, and nothing constructs a `Readiness` outside this module
because the library computes them and hands them back.

A status added later widens `unreadyReasons` without breaking anyone, which is
what publishing the list buys. A consumer that hard-coded the eight values
instead would fall through on the ninth, and that is the failure the key exists
to prevent.

A ticket envelope gains `branch`, per section 8 and 10.1. It is additive and
null on every row an ordinary query answers with, so a consumer that ignores it
reads what it read before and this is a minor release. `--cross-branch` adds
beside the default rather than changing it: without the flag, `list` and `ready`
answer exactly as they did. This one is cheap for a reason worth naming, which
is that the field was designed with the flag rather than retrofitted to it. A
consumer that later reads `branch` without knowing about `--cross-branch` sees
null and is correct.

`updated_at` and `updated_by` stay in the ticket envelope and are covered from
`v1.0.0`, per 5.3. This is a decision not to break rather than a break, and it
is recorded here because the alternative was live and the window for taking it
cheaply closes when Phase 3 arrives. The pair moves together or not at all: 7.5
resolves the actor by the timestamp, so a release dropping one and keeping the
other would publish a field with no rule behind it.

That decision does not follow from the rule this section uses elsewhere. Both
breaks above were taken because nothing consumes the surface yet, so now is the
cheapest they will ever be, and the same reasoning would have made dropping the
pair free today. What stopped it is that the field is right rather than that the
break is expensive, and those are different arguments that happen to point the
same way most of the time.

The four-directory layout is not a schema break. A ticket file does not change
when its path does, so `schema` stays at 1 and no store needs `migrate`. Every
file a migrated store holds still parses in any binary that could parse it
before.

It does cost something a reader should not have to find out for themselves. An
older binary walks `tickets/` and `archive/` and nothing else, so against a
migrated store it silently answers without the drafts and without the done
tickets. It does not fail, which is the problem: a listing that is quietly short
reads exactly like a listing of a smaller store.

That sits close to the rule this section states about a store never upgrading
itself, and it is worth being exact about why it does not break it. Nothing
self-upgrades. A store moves only when a person runs `check --fix`, and until
they do, every binary agrees about it. What the rule protects is a colleague who
has not upgraded, and the honest statement is that migrating a shared store is a
decision to make once the people reading it have the newer binary. It is one
rename-only commit, so the diff says plainly what happened.

The module version tracks the Go API alone. A `schema` bump is not a Go major:
learning to read schema 2 while schema 1 still parses is additive, so it ships
as an ordinary minor release and the import path does not move. A consumer that
needs to know what it is talking to reads `ticket.SchemaVersion` at runtime,
which is why that constant is exported. The module goes to `/v2` when the Go
API breaks, and not because the file format moved.

A store never upgrades itself. When the library learns to write a newer schema,
an existing store stays where it is. Reading never rewrites, a mutation writes
back the schema the file already declared, and a new ticket is written at the
store's declared schema rather than the binary's maximum. Upgrading a binary
therefore cannot make a repository unreadable to a colleague who has not
upgraded, which is the failure this rule exists to prevent. A store moves only
through an explicit migration that a person runs, described in 12.5.

The module is `v0.x` and stays there until Phase 3 lands. Plain semver reads
`v0` as promising nothing, which is not what this section means. While the
module is `v0.x`, a break in a covered surface bumps the minor and a fix bumps
the patch, so `v0.2` to `v0.3` carries the warning that `v1` to `v2` will carry
later. Terva is the first real consumer and the first thing that will find the
gaps. Reading the parent hierarchy back was the first gap it surfaced, and the
break it looked like never arrived, because the answer was a `parent` filter on
`list` that 10.1 did not move for. That is the argument for waiting rather than
against it. What a gap costs is not knowable until a consumer hits one, and
`v1.0.0` is a promise not to break a covered surface. Making that promise
before anything has used those surfaces guesses at which ones were right.

### 12.5 Schema migration

`config.yml` declares the store's schema. It is the level the store's files are
written at and the lowest a reader has to understand, which is why a reader
refuses a store declaring more than it supports. Only a migration changes it.

That declaration is what keeps a store from drifting. `create` stamps the store's
declared schema and not the binary's maximum, so a newer binary working in an
older store writes files that store's colleagues can still read. Without the
rule a store becomes mixed through ordinary use, and the tickets an older binary
cannot read are the newest ones, which are the ones being worked on.

`git ticket migrate` and `Store.Migrate` perform the change. A person runs the
command and a host embedding the library calls the method, because both need it
and neither can drive the other.

It converts the whole store in one pass, under the store lock, and it is
idempotent. A run interrupted by a crash or a full disk is finished by running it
again, because a ticket already at the target is skipped rather than rewritten.

`config.yml` is written first, before any ticket. The two failure modes are not
symmetric. A config a reader does not understand refuses the whole store, loudly
and on every command, while a ticket it does not understand drops out of queries
and is reported only by `check`. An interrupted migration should therefore leave
a store an old reader refuses outright rather than one it reads with tickets
missing. Announcing the change up front is honest and going quiet halfway is not.

Migration writes files and does not commit. Publishing stays the user's ordinary
Git workflow, per 7.4 and the sync-helper decision in section 15.

There is no downgrade. A field added in a later schema has nowhere to go in an
earlier one, and a migration that quietly dropped it would lose work.

`check` warns when a store's files disagree with its `config.yml`, naming how
many and pointing at `migrate`. It warns rather than errors, because such a store
is correct for a reader that understands both levels. A half-finished job should
still not be invisible.

Only the `create` rule is built. The command, the method, and the warning land
with schema 2, along with the fixtures that prove them and the finding code
registered in section 11. Building them now would mean untested code for a state
that cannot occur. A ticket declaring more than the reader supports does not
parse, so no fixture can hold a store whose files merely disagree with its
config. The `create` rule ships now because it is load-bearing whether or not a
second schema ever arrives.

### 12.6 Self-update

`git ticket self-update` replaces the running binary with the latest release,
and its no-op modes answer whether it would. The source is the GitHub releases
of the public mirror, resolved from the module path, because that is the one
host every installed binary can reach and the release pipeline of 12.1 already
publishes there. The internal forge is not consulted: a binary in the wild has
never heard of it.

The name is `self-update` and not `update`, because `update` has meant "change
a ticket's fields" since Phase 2 and a command that means two things means
neither.

Three modes. `--check` asks the releases API for the latest tag, compares it
with the running version, reports, and exits by the graded bucket of 10.2. It
downloads nothing. `--dry-run` additionally resolves the asset name for this
OS and architecture and the path it would replace, and exits by the same
bucket, so the two no-op modes differ only in how much they name. The bare
command downloads the asset and `checksums.txt`, verifies the sha256 before
anything moves, unpacks the binary, and replaces the executable atomically:
write beside the target, then rename. On Windows the running binary is renamed
aside first, because a running executable cannot be overwritten there.

Refusals, each with exit 1 and a reason: a binary reporting `devel`, because
that is somebody's development tree and replacing it destroys a build
silently, the repair being `just install` or `go install`; a checksum that
does not verify; a target the process cannot write, naming the path so the
user can decide whose permissions are wrong; and a release carrying no asset
for this OS and architecture.

A prerelease is never offered. The releases API's `latest` excludes them, and
this command asks for nothing else.

The network exception is scoped to this one command and its no-op modes. The
library still reaches no network, per 7.4, and no other command does either:
`check` keeps the promise of section 11. `self-update` touches no store, holds
no lock, and works outside a repository, per 10.7.

A binary that is already at the latest, or ahead of it, reports up to date and
exits 0. Ahead happens between a tag push and the mirror's release job
finishing, and treating it as an update would downgrade.

## 13. Phases

### Phase 0: format and fixtures

Sections 4 through 11 of this document are the format decision, so what remains
is the corpus. Build `testdata/fixtures/` covering valid tickets of each type
and status, invalid frontmatter, an unknown top-level field, an unknown Markdown
section, a file with conflict markers, a duplicate ID, a dependency cycle, a
parent cycle, an archived dependency both with and without `from_status: done`,
an expired claim, and a `schema: 2` file.

Exit criteria: every fixture has an expected `check` result recorded beside it.

### Phase 1: core library

Store discovery and `init`. Parse, render, validate. ULID generation and prefix
resolution. `List`, `Search`, `Get`, `Ready`, `Check`. Field-level mutation
through `Apply`, with the revision precondition and the flock-based store lock.

Exit criteria: the round-trip property holds on every fixture, golden tests pin
the rendered bytes, the fixture corpus produces its recorded `check` results, and
two concurrent processes writing the same ticket produce one success and one
`stale_revision`.

### Phase 2: standalone CLI

Every command in 12.1, human output and `--json`, the documented error codes and
exit statuses, `instructions` and `schema`.

Exit criteria: the JSON contract has a test per kind, `git ticket check` runs
green in this repository's own CI with no network, and a scripted end-to-end run
covers create through claim through done through archive.

### Phase 3: terva integration

Owned by terva. It starts once Phase 2 tags a release, so that terva builds
against a stable module rather than a moving one. `v0.1.0` closed Phase 2 and
`v0.2.0` added the `cli` package terva embeds, so the gate is open.

The git-ticket side of this phase is one artifact,
`docs/handoff-terva-phase-3.md`, which a terva agent works from. Everything
else, the command, the tools, the board, and the permission wiring, is built in
terva by terva.

What this phase can still ask of this repository is a library or format change,
and the answer goes through the usual route: a plan change first, then code,
then a tag. The one already visible was reading the parent hierarchy back,
because a board that shows an epic's children needs it. Section 8 now answers
that with a `parent` filter on `list`.

### Phase 4: adapters and views

The stdio MCP adapter. Import of the useful Backlog.md fields, with no runtime
dependency on Backlog.md. A local browser or TUI view, and only after the file
and agent contracts have held through at least one real project. No remote
helpers, per the sync-helper decision in section 15. Publishing stays the
user's ordinary Git workflow.

One carve-out, decided 2026-09-04 on TKT-01M1QBSD: the TUI's rendering
infrastructure, the `tui` package with its terminal abstraction and frame
renderer, may land ahead of this gate, because it consumes no ticket contract
and so cannot be destabilized by one. The gate keeps its force for the view
itself, the part that reads and writes tickets.

## 14. Acceptance criteria

Format:

- a fresh clone can init or discover a store with no network
- a ticket reads correctly in an ordinary Markdown viewer
- `git diff` shows the fields a mutation changed and little else
- parse and render round-trip every fixture without loss
- two disconnected processes create tickets without an ID collision
- a `schema` bump produces an actionable message, not a parse error

Contract:

- every operation has a documented, versioned JSON shape
- a field-level update leaves unrelated fields byte-identical
- missing dependencies, duplicate IDs, and both cycle kinds fail `check`
- an invalid transition names the permitted targets
- conflict markers report as `merge_conflict`

Concurrency:

- two local processes cannot write the store at once
- a stale `ifRevision` returns a conflict instead of overwriting
- worktrees sharing a Git directory share the lock
- the documentation states the cross-clone visibility delay for claims
- no command performs a remote operation as a side effect

## 15. Deferred questions

A question is named by the subject in bold and the ULID of its ticket. This
section numbered its questions once and stopped, because a number has to be
chosen and two agents working in parallel choose the same one without noticing.
That happened twice. A ULID is generated, so it cannot collide, and
`git ticket show` resolves it to the ticket holding the detail.

**Custom statuses** (`TKT-01M1F7Z2Y5H1ZJAHRGF3XE6F91`). Whether they are worth
the cost. The trigger is a workflow that needs a state the seven in 6.1 cannot
express and that a label plus `status_reason` cannot express either. That second
half is the part worth checking, because labels are already open and
`status_reason` already answers "why is this blocked now", so most requests for a
new status are really requests for one of those. The cost, if the trigger is ever
met, is that the transition table in 6.2 stops being a constant and becomes data
`check` validates against, 4.1 stops being able to say configuration cannot add a
status, and the status enum `git ticket schema` publishes becomes store-specific,
so no consumer can hard-code it again.

**Stdio adapter tool discovery** (`TKT-01M1F7Z2ZXFX7W4MJ9H1KB8SFZ`). What the
adapter should expose. Phase 4, not started. The trigger is a host that wants to
drive git-ticket and cannot embed the `cli` package, which is the only reason an
adapter exists at all. Terva is not that host, because 12.2 exports `cli` for it
to embed. Much of the answer is already built. `git ticket schema` publishes the
enums, the error codes and the finding codes, so an adapter can generate most of
a tool description rather than restating it and drifting. What is undecided is
whether every command becomes a tool, or only the ones an agent should reach
for.

**Backlog.md import and a local view** (`TKT-01M1F7Z30Q3PZFS1Q7B0F715Z9`). When
each is worth building. These are two questions with two triggers. Import waits
for a real Backlog.md project somebody wants to move, and the count of tickets in
it is the evidence, because an importer for a backlog nobody has is guesswork
about a format read from the outside. The local view has its trigger written
already in Phase 4, which is that the file and agent contracts have held through
at least one real project. Neither is blocked on anything in this repository.

**A merge driver for ticket files** (`TKT-01M1HE7KX06FY8W1GYXH9MXGBP`) is
answered in 7.5. git-ticket ships one, and 7.5 holds the field rules and the
install story. Building it is `TKT-01M1JPW1QJR3QM4SE9T3GDTZBX`, kept separate
because the question was whether to ship a driver and not what its code looks
like.

Running the question changed what the driver is for. The ticket framed the work
as deciding which fields merge by union, which are last-writer-wins, and which
always conflict, which assumed the conflicts are about the data. They are mostly
not. Two branches were made to edit one ticket in ways that share nothing, one
setting `priority` and one adding a label, and they conflicted on `updated_by`
alone while Git merged both edits without complaint. Every mutation rewrites
that field and `updated_at` beside it, so any two concurrent edits collide on
lines neither agent typed. `Notes` and `Comments` supply the rest by appending
at the same offset.

That makes the first useful driver a small one. Resolving two provenance stamps
and two append-only sections clears the conflicts a real workflow hits, and
adjudicates nothing anybody wrote. The full table in 7.5 exists so the rest is
decided rather than discovered later, and most of its rows say conflict.

It also raised `TKT-01M1JPXG7J6K4MQNTWA6TVMHPB`, filed on the conflict evidence
and then narrowed, because the driver retires that argument rather than hiding
it: the rule for both fields is mechanical and 7.5 states it. That question is
answered next.

**The provenance pair in the published surface**
(`TKT-01M1JPXG7J6K4MQNTWA6TVMHPB`) is answered in 5.3 and recorded in 12.4.
`updated_at` and `updated_by` stay, and they are one fact rather than two
fields.

The ticket left three arguments for dropping them once 7.5 had retired the
conflict cost. Two did not survive being checked against the code and the store.

Nothing reads either field, which is true and is not a criterion. Nothing reads
`created_at` or `created_by` either, and no query, filter, sort, or check
consumes any of the four. Provenance answers a question a person asks of a diff
rather than one a caller asks of a store, so being unread is the ordinary
condition of the group. A rule that dropped a field for having no reader takes
`created_by` with it, and nobody proposed that.

The stronger argument was that a Git-native format restates what the object
database already holds, with less precision. That is false here, and measurably.
Nothing in this project commits, per 7.4, so a mutation sits in the working tree
until somebody gets around to committing it and `git log` knows nothing until
then. When a commit does arrive it carries the committer's Git identity, while
an actor is an agent session. This repository's own store holds
`agent:terva/mieli`, `agent:terva/dev-loop`, and `human:sothr` across its
tickets, and every commit in it is authored by one person. Git collapses the
three into that one, so on the axis an agent workflow cares about it holds
strictly less than the pair does.

Dropping one and keeping the other is not a third option. 7.5 resolves the actor
by taking the one belonging to the later timestamp, so a format without
`updated_at` publishes an `updated_by` with no rule behind it.

What the ticket did not know is that the surface it was arguing about had no
guard. No test in either package named `updatedAt` or `updatedBy`, so removing
both from the envelope would have passed the whole suite in silence, which is
the same failure `--version` shipped with in v0.4.0. `cli/provenance_test.go`
now holds the four keys as literal strings, and holds the pair to moving
together.

The honest limitation stays and is written into 5.3. The pair names the last
writer and no earlier one. The per-entry log is `Notes` and `Comments`, which
stamp every entry with its own actor and time.

**External tracker integration** (`TKT-01M1HFQ5F4D4KF45A5PFQ39XST`). How
git-ticket integrates with one. Storing the identifier always worked, because
5.1 leaves the reference namespace open and `decodeReferences` takes the ref
verbatim, so `jira:PROJ-1234` links and `check` is content with it. Three gaps
sat above that storage, and two are now closed.

The lookup is `refs REF`, specified in section 8 and matched by the rule in 5.5.
The namespace half of the identifier problem went with it: `refs` compares the
namespace without regard to case, so `JIRA:` and `jira:` are one type, and the
identifier exactly, so `PROJ-1234` and `proj-1234` stay two references. An
untyped ref is now `reference_untyped`, a warning in section 11.

The lookup was built before the validation on purpose, and that order is the
durable part of this entry. A namespace grammar invented with no caller is a
guess, and a finding code is expensive to guess wrong: `schema` publishes it, so
12.4 puts it on the covered surface the moment it ships. Building the lookup
first showed how little validation it actually needs. Case is a comparison rule
rather than a stored-bytes rule, so it costs no finding at all, and what was
left is the one structural claim 5.1 already makes.

What remains is `extensions`, which has no mutation, so the one place 5.1
reserves for a consumer's own fields round-trips through parse and render but
can only be written by hand-editing the file. That stays deferred for the same
reason the grammar was: there is no consumer until Phase 3, and Phase 3 is
terva's. A mutation shaped for a caller that does not exist is the guess this
entry just finished arguing against.

**Long-term archiving** (`TKT-01M1HFPCRWCW72EA6EKACFSYZ5`). What a store does
with an archive that grows without bound. Every ticket ever closed stays in
`archive/` as its own file, and nothing prunes it, rolls it up, or compresses
it. The sketch to start from is a tarball per archived month for anything older
than thirty days, plus an index tooling reads instead of opening them.

The trigger is a store where the archive is measurably the reason something is
slow or large, rather than a store where it looks untidy. This repository's
archive is seven files.

The cost, if the trigger is ever met, is that the archive stops being files a
person can open. 6.3 makes a dependency satisfied by a ticket archived out of
`done` and by nothing else, so `check` and `deps` read archived tickets to
answer questions about live ones, and 11 scans `archive/` for a location
mismatch and for an ID duplicated across both directories. An index carrying
each archived ID and its `from_status` answers all of that without opening a
tarball, and an index carrying less makes every `check` unpack a month. A
`.tar.gz` also gives up what the format is for. One late arrival rewrites a
whole month, review sees an opaque object, and `tar` records mtimes and entry
order, so a rebuild differs from an identical input. The index is the separable
half and costs the format nothing, which is the argument whoever settles this
has to answer rather than inherit.

**What counts as a store** (`TKT-01M1HJTCYTZZHGGPCXT2SAJMFR`) is answered in
section 4. A directory is a store when it holds `config.yml`, and `Open`
returns `store_not_found` without one. The rule lives in `Open` rather than in
the CLI's store resolution, because the trigger for settling it was Phase 3,
where a consumer reaches the library directly and would otherwise inherit the
old behaviour through an API the CLI never touches.

The marker already existed and nothing read it. `init` wrote it, `Discover`
walked up looking for the name `.tickets`, and section 4 described the layout,
so the definition was everywhere except the check at open time. Existence was
the whole test, which made `--store` at a typo answer `ticket-list` with an
empty array and exit 0.

This was first settled as `config.yml` and `tickets/` together, and Git
overruled it. Git tracks no empty directory, so `init`, commit, clone produces
a store with three files and no subdirectories, and requiring `tickets/`
rejected it. `TestWorktreesShareOneLock` failed on exactly that: it commits a
store and adds a worktree, and the linked checkout had no `tickets/` because
the directory was empty. The rule would also have rejected any store that
finished its open work. A directory cannot be a marker in a format stored in
Git unless something keeps it non-empty, and nothing here does.

The name `.tickets` was never a candidate. `--store` and `GIT_TICKET_STORE`
name a directory outright, and the corpus points them at fixture stores called
`store/`, so requiring the name would reject all seventeen.

Running the question found one more thing the ticket did not have.
`check --strict` already refused an empty directory, but through
`epics_index_stale`, whose message says `epics.md` "does not match the epics in
this store" while failing to be a store. Plain `check` exited 0 on the same
directory, because that finding is a warning. The finding was right that the
file was missing and wrong about what that meant, which is what a check reports
when the thing it assumes goes untested.

A question leaves this list by being answered in place, so the entries above and
below are the whole record and no tally of them belongs here. This paragraph
carried one for a while, and it was exactly the kind of fact this document
should not hold: nothing could check it, and updating it was a step somebody had
to remember at the end of unrelated work.

The questions were numbered once, and the numbering is the other thing this
section gave up for the same reason. A number has to be chosen, and it collided
twice. Two settled questions both called themselves question 7, the module
path and the parent hierarchy. A third was filed as question 9 from a
concurrent worktree while question 9 already meant the merge driver. Commit
messages and pull request bodies from that period still cite numbers, and the
subject of each entry is what decodes them.

**Section 15 as a hand-maintained index** (`TKT-01M1HJKBYGENBTJ7F9S71BN3Q1`) is
answered here. It stays hand-maintained, and the parked entries stay in this
document rather than becoming a query against the store.

The section turned out to be authorial rather than an index. Four of the eight
parked entries were longer than the ticket descriptions they name, and custom
statuses ran 756 characters here against 192 there. A pointer would have
dropped that prose, so the real price of the move was a migration nobody had
counted, and the section would have lost the part that makes it worth reading.

Splitting the parked half from the settled half also splits one narrative
across two homes. The settled entries have to stay prose, because they carry
answers and cross-references into other sections that no query reproduces. Move
only the open ones and settling a question means migrating its content between
homes, while a reader asking what was decided about something has to know
whether it was settled before knowing where to look. That costs more than the
collisions it avoids.

Those collisions are cheap and they announce themselves. Two entries appended
in parallel conflict in Git, which stops the merge and is resolved by keeping
both. The failure is a delay, never a wrong answer. Generation would trade it
for a determinism gate that drifts quietly, which is the cost that still
stands.

The other half of that objection is spent. It said a generated list would need
a label taxonomy the format wants for nothing else, and `question` is now in
the `config.yml` allowlist with every parked ticket carrying it, so
`git ticket list --label question` answers today. The label was worth having on
its own, for asking the store what is open without reading this section. It
does not reopen the question, because the two reasons above never depended on
it: the entries are longer than the tickets they name, and splitting the parked
half from the settled half would still split one narrative across two homes.

The friction is real and stays. A new entry lands where the last one did. Keep
both and put the newer one last, which is the order the ULIDs already give.

**The compatibility policy** (`TKT-01M1F7Z31HR5GV06D6Y7WZWJK4`) is answered in
12.4. The module version, the file `schema`, and the envelope `schemaVersion`
move independently. Everything a machine reads is covered and human output is
not. A covered surface breaks only when something that worked stops working or
changes meaning, so adding beside the old thing is always minor. The module
version tracks the Go API alone, which makes a `schema` bump an ordinary minor
release rather than a `/v2`. A store never upgrades itself. `v1.0.0` waits for
Phase 3, because no consumer has exercised a covered surface yet, and the
promise `v1.0.0` makes about those surfaces is a guess until one has. Settling
it is what raised the schema migration.

**Reading the parent hierarchy back** (`TKT-01M1FCMN7QEWM584N192NBC7TD`) is
answered in section 8 by a `parent` filter on `list`. Running the question
before answering it is what shrank it. The data was never missing. `list`
already returns every ticket with its `parent`, so a consumer could always
rebuild the tree from one call. What was missing was a way to ask for one epic's
children without pulling the whole store, and any way at all for a person at a
terminal to ask. A filter is additive under 12.4, so it ships as a minor release
and 10.1 does not move, which is what settled it against the two richer answers.
`show` rendering children would put data derived from other files onto the
ticket object, and `deps --children` would overload a command whose whole
contract is that it walks `dependencies`.

**Renewing a claim** (`TKT-01M1F7Z2XAV593RH0KAVBYZQSR`) is answered in 6.4, and
running the question is what shrank it. No verb was missing. `claim ID
--expires-in 1h` run again already re-anchors the bound, which is renewal. What
the question hid was a defect underneath it. A re-claim that supplied no expiry,
against a store with no `defaults.claim_expiry`, cleared `expires_at` outright,
so the natural way to stay alive on a long task widened a bounded claim into an
unbounded one, and reset `claimed_at` on the way past. Both now survive a
renewal. Fixing it cost no compatibility, because every `config.yml` in this
repository and the corpus sets `claim_expiry: null`, so no store had an expiry
to lose. The richer answer, a `--renew` flag that extends a claim by the amount
it was first given, was ruled out by the file format rather than by taste. The
claim stores two instants and never the duration between them.

**Sync helpers around ordinary Git commands** (`TKT-01M1F7Z2Z33HW6FW44TCQVWB7M`)
are answered no, and two independent mechanisms had answered the question before
anyone sat down to decide. The library side is 7.4. A sync helper means adding
`fetch` or `push` to the enumerated table, and `TestGitCommandsAreReadOnly` holds
the source to that table, so the addition cannot happen quietly. The workflow
side is newer. The multi-agent friction that would justify a helper did arrive,
and what absorbed it was a process rule rather than a tool. Work lands on `main`
through a pull request, and pushing to `main` is forbidden outright, so a
`git ticket sync` would automate the fetch, rebase, and push this project now
tells its own agents not to do. Phase 4 no longer holds a slot for one. What the
friction did raise is the merge driver above, which is a different question,
because merging two edits of one ticket needs no network and no writing Git
command.

**An explicit schema migration** (`TKT-01M1H9X166M1ATNK9S7ET26BVQ`) is answered
in 12.5, and designing it before there is a schema 2 is what turned up the reason
to do it early. `config.schema` was read in exactly two places, to refuse a store
the reader is too old for and to write itself back, and nothing consulted it when
writing a ticket. `create` stamped the binary's maximum, so a newer binary would
have written newer files into an older store with no migration run at all, and
the tickets a colleague could not read would have been the newest ones. That rule
ships now. The command, the library call, and the `check` warning are specified
in 12.5 and land with schema 2, because a ticket declaring a schema the reader
does not support fails to parse, so no fixture can express a store that is merely
mixed, and code no test can reach is worse than code not yet written.

Two were settled during Phase 1, before any reader had shipped, so adding
`status_reason` to schema 1 cost no compatibility. Where a `blocked` reason
lives is answered by `status_reason` in 5.1 and 6.2: the field holds the
current reason and `Notes` keeps the history. What a `references` path resolves
against is answered in 5.5: the root of the Git repository holding the store,
and no finding at all when the store sits outside one.

**The module path** (`TKT-01M1F8XG6KXN6QXYWF6EHVB88P`) was settled by publishing
rather than by argument. `go.mod` declares `github.com/terva-sh/git-ticket` and a
public mirror now serves that path, so the declared path is the real one and
nothing changes. The alternative, renaming the module to the host that serves it,
was wrong on its own terms. A private hostname does not belong in a public
artifact, and a module path is the most load-bearing string in a Go project to
change later. See 12.2.

Three were settled during Phase 2, because building the CLI is what forced
them. The precedence between `--store`, `GIT_TICKET_STORE`, and `config.yml` is
answered in 12.1: the CLI is the first thing that had to resolve a store.
Whether an archive reason also goes to `Notes` is answered in 6.3, and it does,
on the same argument that puts a status reason there. Whether `update` carries
`--type` and `--parent` is answered by 12.1 listing both, because every other
field `create` sets already had an `update` flag and the library already had
`SetType` and `SetParent`.

One was settled after Phase 2, because the same trap sprang three times.
Whether the renderer canonicalizes body section text
(`TKT-01M1HQ3D5BMBBP9CEVXBHP3YSN`) is answered in 5.3, and it does not. The
normalization sits in `writeTicket` instead, which is the one place every write
already passes through, so no future writer has to remember to trim. The
renderer stays a faithful echo of the struct because the store hands that same
struct back to callers, and a renderer that normalized alone would leave the two
disagreeing. Settling it also caught the per-writer trims using `TrimSpace`,
which is stronger than the round trip needs and would have reindented a section
opening with an indented code block.

One more was settled twice, and the second answer is the one that shipped.
Whether an epic can block on its children
(`TKT-01M1HXHJWR806ETJQCE49AEZB3`) is answered in 5.1 and 8, and it can.
`blocks_on` names the edges that gate a ticket beyond its dependencies,
`readiness` gained `blockingChildren`, and 11 gained `invalid_blocks_on`,
`blocks_on_no_children`, and `blocking_cycle`.

The first answer, settled on paper, was wrong, and writing the code is what
showed it. That answer made the enum selective, with `none`, `listed`, and
`children`, defaulting to `none`. The two halves contradict each other. If a
value selects which edges gate, then `none` means dependencies stop gating, and
because 5.3 renders every known field on every ticket, shipping that default
would have switched dependency blocking off in every store at once. `listed`
existed only to carry the behaviour the default was quietly discarding, which is
the tell: a value that exists to undo the default is a default facing the wrong
way.

The field is additive instead. Dependencies always gate, `children` adds the
child edge, and `none` is honestly the default because it takes nothing away.
`TestBlocksOnIsAdditive` holds it there, and it is the test to read first if this
ever looks like it wants a third value.

The rest of the paper answer survived intact: the gating children are derived
from `parent` rather than enumerated, an epic with no children is not blocked and
`check` warns instead, children get their own field rather than joining
`blockingDependencies`, and `deps` still walks dependencies alone.

Three more were settled for the generated epics index
(`TKT-01M1HXHJXRFP7VMH7D35YNTG5H`). 4 and 11 now carry them, and the reasoning
stays here.

The tension this ticket recorded with the store partition resolved by both
shipping. The partition keys directories on status and refuses to key them on
type, so it never answered `list --type epic` and never could. The index is that
answer, and section 4 now holds both halves: a directory for the status view, a
generated file for the type view.

Which epics appear is an exclusion and not a list. `done` and `archived` are left
out and everything else appears, which today means `draft`, `ready`,
`in-progress`, `blocked`, and `review`. An inclusion list would silently drop any
status added later, and whether this format ever grows one is itself an open
question in this section. Including drafts is also the better answer on its own
terms, because a draft epic is a decomposition somebody is writing, which is what
a person browsing a forge most wants to see.

The index covers epics and nothing else, and a second view is a change to this
plan rather than a key in `config.yml`. Configurable views put queries in
configuration, which the tool then has to validate and every consumer then wants
its own copy of in the store. The generator may be reusable inside the binary.
The choice of what it renders is not the store's to make.

A stale index is a warning, and the tradeoff this looked like does not exist.
`--strict` promotes warnings to errors, per 10.3, and every CI invocation in
this plan carries it, so a warning is enforced exactly as hard as an error
everywhere enforcement happens. Keeping it a warning holds the line that a
derived file falling behind is not a malformed store, which is the same line
that keeps "this ticket is late" out of `check` entirely.
`TKT-01M1HXPJP7EGJVC2J7GC6Y887E` records the same answer, because the two cannot
be allowed to diverge.

Severity does not gate repair, which is the fact that makes the warning free.
`planRepairs` recomputes where each file belongs rather than walking the
findings, so a warning is exactly as repairable as an error. What does need work
is the shape rather than the severity: a `Repair` today is a rename, carrying
`From` and `To`, and rewriting a generated file is not a rename. However that
lands it is an implementation cost, not a second format decision.

Three more were settled for a deadline on a ticket
(`TKT-01M1HPCVRK1989NDNR9PJS36S4`), and 5.1, 8, and 11 now carry them. The
reasoning stays here, because those sections say what the format is and this is
why it is that.

The field is `due_on` and it holds a date, `2026-10-14`, meaning the end of that
day in UTC. Neither name the ticket offered survives, because the naming
question and the precision question turned out to be one question. `due_at` was
the option that fit the convention, and the convention it fits is that `_at`
holds an RFC3339 instant, which every timestamp in 5.1 does. A `due_at` holding
a date would teach a reader that `_at` sometimes means a day, and `expires_at`
on the claim block is what goes ambiguous then. One new suffix carries a
distinction a reader needs anyway: `_at` is an instant, `_on` is a date.
`complete_by` reads better in a sentence and says nothing about which of the two
it holds.

The date is stored as written rather than expanded to an instant, because
expanding it throws away what somebody meant. A deadline is a claim about a
calendar day, and no deadline was ever 23:59:59Z. Expanding at write time also
has to pick a zone. The writer's local zone stores two different instants for
the same typed date depending on who typed it, and UTC is the end-of-day rule
above with fake precision spelled into the file. Either way the file stops
recording that somebody said the 14th, and no reader can then tell an expanded
date from an instant somebody chose. Backlog.md stores a minute-precision
instant and rejects date-only input, which is both halves of this wrong at once.

`ready` orders by `due_on` ascending with no flag, undated last, ID as the
tiebreak. The tradeoff the ticket described does not survive that tiebreak. A
store where nothing carries a deadline sorts exactly as it sorts today, because
the date key does nothing until somebody sets a date, and setting one is a
deliberate act by the person who wants the order to change. The default
reorders no existing caller, and the flag would have been a thing to learn
before getting behaviour the reader had already asked for by typing a date.

Undated last is "closest to late first" with never at the far end. That order is
part of what `ready` answers rather than an index into it. `ready` recommends
what to start next, so its ranking is the recommendation, while `list` reports
what exists and stays chronological.

This settled one sort key and did not make `ready` rank by priority.
`TKT-01M1J2YR9D5242F6H7TPEV4M8K` took that question, which the deadline key
raised rather than created, because `ready` sorted by ID and an urgent ticket
already placed below a low one filed before it. It is answered below, and the
answer put priority above the key this entry settled.

`check` reports a `due_on` it cannot parse and never reports one that has
passed. Validation is about the store, and a date going by changes no file. A
`check` that went red because a calendar day turned would fail CI for a reason
no commit caused, which is how a team learns to stop passing `--strict`.

Building it settled a fourth thing, which the ticket had carried as one line of
acceptance criteria rather than as a question. `list` filters with `--due-by`,
an inclusive bound, and sorts only when asked with `--sort due_on`, while `ready`
sorts always. 8 holds the asymmetry and its reason: `ready` ranks, and a list
reports. The inclusive bound is the same kind of choice. "Due by the end of the
month" is the query somebody types, and a strict `--due-before` loses the last
day of it.

Every fixture that parses gained the field, because 5.3 renders an absent scalar
as `null` rather than omitting it. That was 42 files, migrated by re-rendering
each one through the package renderer rather than by hand. Two more carry
frontmatter and did not gain it, because `conflict-markers.md` and `schema-2.md`
exist to fail parsing, so a field they never reach is a field they must not
carry.

**Whether `ready` ranks by priority** (`TKT-01M1J2YR9D5242F6H7TPEV4M8K`) is
answered in section 8. It does. `ready` orders by priority, then `due_on`, then
the ID, and `--sort` gains `priority` so the field that ranks one command can
order the other.

Both orderings are wrong somewhere, and the case that decides it is a `low`
ticket due next week against an `urgent` with no date. What settled it is which
wrong answer a person can correct honestly. Under deadline-first the only way to
lift the urgent ticket is to invent a date for it, which corrupts a field to
move a sort. Under priority-first the way to lift the dated ticket is to raise
its priority, which is what the field means. An ordering should be gamed by
telling the truth.

The store made the case harder to argue with than the ticket did. Across 53
tickets nothing carried a `due_on` at all, while 23 carried a priority other
than the default. The key that shipped ranked nothing and fell through to the ID
tiebreak on every row, while the field people actually set was not a key. A
command that says its ranking is part of what it answers was ordering by filing
date.

An urgency score folding the two into one number was ruled out before the
question was asked, and that holds. A weight nobody can see turns an order a
person can predict into one they have to reverse-engineer, and the weights
become configuration the store then has to validate.

Two more were settled for what a listing answers
(`TKT-01M1J755Q274KHQX9XFXAK6A55`), and 8, 10.4, and 12.4 now carry them.

`list` answers with open work, which is every status except `done` and
`archived`. The old default was every ticket except archived, and in this
repository's own store that was 44 rows with 30 of them done. The line already
existed and sat in the wrong place. Archived was excluded for being terminal,
and done is terminal in the same way.

Stating it as an exclusion rather than as a list of the five open statuses is
the part that has to survive. **Custom statuses** is still open, and an
inclusion list would silently drop any status that arrived where an exclusion
keeps working. The epics index answered the same question the same way, and two
views of one store disagreeing about what counts as live is worse than either
answer.

The library moves with the CLI. `Filter{}` means open work rather than no
filtering, so a host embedding `ticket` gets the answer the command gives
without restating the set, and `Filter{All: true}` is everything. That is a
break to a covered surface, taken now because nothing consumes it yet, and 12.4
records it along with the `IncludeArchived` rename it carried.

`search` is the one place the new default would have been wrong, so it does not
take it. Search is how somebody finds what was already decided, and a decision
lives in a done ticket. 8 holds that asymmetry and its reason.

The store layout was settled next
(`TKT-01M1HVMQQQE3K6VZG7793RXVXN`), and 4, 6.3, 11, and 12.4 now carry it.

The store partitions into `draft/`, `tickets/`, `done/`, and `archive/`, keyed
on status. It is mandatory rather than opt-in, so the `directories` key that
ticket spent half its length designing does not exist. A layout a store can
decline is one no reader can count on, and the whole value here is a reader
being able to count on it.

The question this answers is not one the tool had. `list` answers with open work
now, so a binary was never the problem. The reader is a person in a forge web UI
or at `ls`, whose only free query is a directory. That is also why closing this
in favour of a generated index was rejected: an index is a second artifact that
can go stale, where a directory cannot lie about which files are in it.

The working statuses share `tickets/` because a directory each would make every
ordinary transition a rename, and those four are exactly what a ticket churns
through while somebody works it. An unknown status lands in `tickets/`, which is
the same exclusion shape as the open set: name the special cases and let
anything new take the ordinary path.

Status is the only key, and type is refused outright. `update --type` is a
frontmatter edit that would become a file move, landing on the promotion of a
task to an epic. Epics are also the interesting rows, so pulling them out of
`tickets/` would empty the view of the thing worth seeing.

Migration is `check --fix` and nothing more. Placement is store layout rather
than file format, so no ticket file changes and `schema` stays at 1, which is
why this needed neither a bump nor the `migrate` command 12.5 designs and
nothing has built. 12.4 records what a migrated store costs an older binary,
which is silence about two directories rather than a parse failure.

`archive_location_mismatch` became `location_mismatch` in the same change,
because the condition outgrew the name the moment a second directory could be
wrong.

Two more were settled for verifying generated artifacts in CI
(`TKT-01M1HXPJP7EGJVC2J7GC6Y887E`), and 11 carries both.

CI verifies and never commits. The step from "CI can detect this" to "CI should
fix it and push" is refused, and 11 gives the reasons: the write credential CI
does not need today, the race against whoever opened the branch, every local
`main` left behind by a commit nobody wrote, and the skip marker a
self-triggering job then maintains forever. 7.4 gained a pointer, because that
table binds the tool and a workflow can run plain `git` with the tool nowhere in
the pipe.

Artifacts inside and outside the store stay under separate commands.
`check --fix` owns `.tickets/`, and `instructions --write` owns the workflow
block at the consumer's repository root. One flag covering both was declined. A
verification that rewrites a project's own `AGENTS.md` will eventually clobber
something written by hand, and the flag that did it would read as harmless.

Settling this corrected the entry point the ticket proposed. That had been
probed against `filename_id_mismatch`, an error, where `check --fix --dry-run`
exits 1. `epics_index_stale` is a warning, so the same command exits 0 on a
stale index. The verification command is `check --fix --dry-run --strict`, and
without `--strict` a job goes green over the exact condition it was added to
catch.

**A hand-set order in the format** (`TKT-01M1HPCJH5RWSA6Z28QM3QV1H0`). Whether a
ticket carries an ordinal somebody sets. Raised by reading Backlog.md, section
B6 of `docs/review-backlog-md.md`. `List` sorts by ID, which 5.5 makes
chronological because a ULID sorts by creation time, so the only sequence this
format can express is the order tickets were filed in.

Half of it is already answered, and the half that is left is the one no flag
reaches. Sorting a listing is display, and display needs no format change:
`list --sort priority` and `ready` ranking by priority arrived with
`TKT-01M1J2YR9D5242F6H7TPEV4M8K`, a flag over data every consumer already had.
A sequence set by hand is different, because no interface can persist a field
the format does not define. Backlog.md keeps an ordinal per task, and that is
what makes its drag-and-drop stick.

The trigger is a board with reorderable columns, or a request for an order
inside one priority level where a fifth priority is not the answer. Neither has
happened.

The cost, if it is ever met, is a frontmatter field on every ticket under 5.3,
which means every fixture. The harder part is that an ordinal would be the first
field whose value means nothing alone and only means something against its
neighbours. 7.5 changed what that costs rather than removing it. Concurrent
inserts now have somewhere to be resolved, which is one row in the field table,
and the honest entry for that row may well be conflict, because two agents
inserting at the same position genuinely disagree.

**Reading tickets from other branches** (`TKT-01M1HPCJJ1FFHG7HXC8QG1JRAG`) is
answered in 8, with the JSON in 10.1. Raised by reading Backlog.md, section D of
`docs/review-backlog-md.md`, and the one place their design is ahead of this one
on a problem this repository actually has.

Every query reads the working tree alone, so a ticket claimed and moved to
`in-progress` in another worktree is invisible until the branch merges. That is
the situation `AGENTS.md` describes as ordinary here: several agents at once, in
separate worktrees, each holding a `main` it believes is current.

The trigger is two agents claiming the same ticket because neither could see the
other's claim. It has not fired, and the pull request rule is what has been
absorbing the pressure.

The merge driver does not cover this, and reading 7.5 as though it did would be
a mistake. 7.5 resolves two edits to one ticket file once Git is already merging
them. This question is about the window before that, when a claim exists on a
branch nobody has merged yet.

The shape is settled and 8 carries it. `--cross-branch` is a flag on `list` and
`ready`, off by default, because it asks a different question than "what is in
my tree" and the caller is the one who knows which they want. Every row names
the ref its winning copy came from, null for the working tree, so a merged view
cannot let an agent claim a ticket whose file it cannot open. The scan reads
`refs/heads/` and `refs/remotes/` both, because an agent here pushes to `origin`
and a scan of local heads alone would miss the case this exists for.

Resolution follows 7.5 rather than inventing a second rule. The later
`updated_at` wins for display, which is what the merge driver already does, so a
listing cannot predict the wrong outcome of the merge it is warning about. A
claim is never adjudicated, which is also 7.5: a live claim on any scanned ref
makes the ticket not ready. Backlog.md's conflict rule is still declined, and
now for a sharper reason than taste. It resolves two branches by file mtime and
offers a knob to pick which guess to make, where `updated_at` is a field every
mutation writes, so this format reads where theirs guesses.

The reads are honest reads, so 7.4 permits the shape, but the table does not
grow until code runs them. Drafting the rows against this repository changed
which rows they are: `ls-tree` and `for-each-ref`, not `ls-tree` and `log`.
`for-each-ref` returns a branch name and its commit date in one call, which is
the recency filter, and `log`'s other use was the mtime guess this declines.
The drafted rows and their reasons are on the ticket, and they move into 7.4 in
the commit that adds the code.

The trigger for building it has still not fired.

**A reason on `readiness`** (`TKT-01M1J758NJ5T40TX6K7GBEZMCY`) is answered in 8,
with the JSON in 10.1 and 10.4 and the compatibility note in 12.4.

The complaint was that `readiness` reported `isReady: false`, `isBlocked: false`
and three empty arrays for every draft in the store. That is correct and it is
useless. A caller learned that a ticket could not be picked up and nothing about
why, so the reason had to be re-derived from `status` and `claim`, which is the
restating that `readiness` exists to prevent.

`reason` names what has to change first, and is empty exactly when `isReady` is
true. It is one enum rather than a set of booleans, so the precedence is
answered once here instead of separately in every consumer.

Two names moved during the work rather than in the ticket. The dependency reason
is `waiting_on_dependencies` and not `dependencies`, because `isBlocked` already
means the graph and the `blocked` status means somebody marked it, so each word
keeps one meaning per field. The claim reason is `claimed` and not
`claimed_by_other`, because `readinessOf` takes a store and a clock and no actor
at all. It cannot tell your claim from somebody else's, and a name asserting a
difference the code cannot check is one no reader should trust. A caller holding
`claim.by` can draw that distinction properly.

The precedence is status, then dependencies, then the claim. A draft waiting on a
dependency reports `draft`, because promoting it is the first move. Nothing is
hidden by that: `blockingDependencies` and `blockingChildren` stay populated, so
the reason ranks the blockers while the arrays carry them.

`UnreadyReasons` is derived from `Statuses` the way `OpenStatuses` is, so every
status except `ready` is its own reason and one added to 6.1 becomes a reason
with no edit and no decision. That matters to **Custom statuses** above, which is
still open: whatever it settles, the reason list follows rather than needing a
migration of its own.

## 16. References

- [Backlog.md](https://github.com/MrLesk/Backlog.md) and its
  [CLI instructions](https://github.com/MrLesk/Backlog.md/blob/main/CLI-INSTRUCTIONS.md),
  the strongest external Markdown workflow and the source of the acceptance
  criteria and definition of done fields.
- [`wedow/ticket`](https://github.com/wedow/ticket), a small Bash and Markdown
  tracker that showed how little the basic loop needs.
- [Beads](https://github.com/gastownhall/beads), agent-first, but with a Dolt
  database as the source of truth and JSONL as an export.
- [`git-bug`](https://github.com/MichaelMure/git-bug), which stores records in
  Git objects rather than worktree files, and
  [issue #1556](https://github.com/git-bug/git-bug/issues/1556) on locking under
  agent workflows. Rejected as the storage model because the records are not
  ordinary files a human can edit.
- `docs/handoff-terva-phase-3.md` for what Phase 3 asks of terva, and what it
  may ask back of this module.
