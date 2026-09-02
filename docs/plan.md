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
├── tickets/
│   ├── TKT-01K3ZYEE00HV9ZDBB8BEASXBBG.md
│   └── TKT-01K3ZYG8K0Y52AD43XRGM4T7WZ.md
└── archive/
    └── TKT-01K3ZYJ360Q7ESC30QAD2SMY0H.md
```

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
  claim_expiry: null
lock:
  timeout: 10s
```

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

`type` is one of `task`, `bug`, `chore`, `spike`, `epic`. `priority` is one of
`low`, `normal`, `high`, `urgent`. `blocks_on` is one of `none`, `children`.

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
ticket and when.

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
`check` reports `archive_location_mismatch` and the status wins.

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

Both only read. Every call goes through one helper per package, `runGit` in
`ticket` and `readGit` in `cli`, and `TestGitCommandsAreReadOnly` asserts three
things: no `exec.Command` in non-test code names a binary other than `git`,
every one of those calls sits in one of the helpers, and every helper call names
a command from this table. A fourth call site added tomorrow has to pass all
three, and a new helper fails the second rather than slipping past the third.

No writing command joins this table. Section 15 records that decision under sync
helpers, so a change that needs `fetch` or `push` is a change to the plan first.

Test code is exempt and runs `git init` freely, because a fixture repository is
not a user's repository.

## 8. Query surface

- `list` with filters on status, type, priority, label, assignee, milestone, and
  parent. Within one filter the values are alternatives and across filters they
  all have to hold. Archived tickets are left out unless the caller asks for
  them, by filtering on the `archived` status or by asking for them outright,
  because a list of work is about work that is still live
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
- `check`, described in section 10

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
`search`, `deps`, and `files` all read the whole store, and a query is not the
place to learn that one file is broken. `check` is, and it reports the file as
`parse_error` or `schema_unsupported`.

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

## 10. JSON contract

Every machine-readable operation emits a versioned envelope on stdout:

```json
{ "schemaVersion": 1, "kind": "ticket-list", "tickets": [], "unreadable": [] }
```

Kinds are `ticket`, `ticket-list`, `mutation-result`, `check-report`, `error`,
`schema`, `instructions`, and `version`. Absent scalars are `null` and absent
collections are `[]`, always present rather than omitted, so a consumer never
has to distinguish missing from empty.

A mutation result:

```json
{
  "schemaVersion": 1,
  "kind": "mutation-result",
  "ticket": { "id": "TKT-…", "revision": "sha256:…" },
  "pathsChanged": [".tickets/tickets/TKT-….md"]
}
```

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

Stable codes, which callers may switch on:

`store_not_found`, `store_exists`, `ticket_not_found`, `ambiguous_id`,
`stale_revision`, `invalid_transition`, `invalid_field`, `dependency_missing`,
`dependency_cycle`, `claim_conflict`, `parse_error`, `merge_conflict`,
`schema_unsupported`, `lock_timeout`, `validation_failed`, `usage`.

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
  "schema": 1,
  "title": "Rotate the signing key without downtime",
  "type": "epic",
  "status": "in-progress",
  "statusReason": null,
  "priority": "urgent",
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
    "blockingChildren": []
  }
}
```

The frontmatter fields keep their names in camel case. `revision`, `path`, and
`readiness` are computed rather than stored, per 7.1 and section 8. `unknown`
holds the top-level fields this version does not define, per 5.4, which are
preserved on write whether or not a consumer understands them.

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
  "codes": ["filename_id_mismatch"],
  "ticket": "TKT-01K3ZZ2JH000GHB4EE6SNRE6MD",
  "from": ".tickets/tickets/notes-about-auth.md",
  "to": ".tickets/tickets/TKT-01K3ZZ2JH000GHB4EE6SNRE6MD.md"
}
```

`codes` is a list because a file in the wrong directory under the wrong name
raises both findings and one move settles them together.

`pathsChanged` names both ends of every move, the way a mutation names what it
touched. It is empty under `--dry-run`, because nothing was written. `repairs`
is not, so a dry run still says what it would do. The findings a dry run would
have cleared are still in `errors`, since the store still has them.

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
  "kinds": ["ticket", "ticket-list", "mutation-result", "check-report", "error", "schema"],
  "statuses": ["draft", "ready", "in-progress", "blocked", "review", "done", "archived"],
  "types": ["task", "bug", "chore", "spike", "epic"],
  "priorities": ["low", "normal", "high", "urgent"],
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

Every one of those values is read from the code that enforces it rather than
copied into the command. A status the library accepts and this document forgot
still appears here, which makes `schema` the answer of record when the two
disagree.

This command reads no store. It answers outside a repository and before `init`,
because a consumer asks what is legal before it has anything to ask about.

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
| `archive_location_mismatch` | the status and the directory disagree; the status wins |

Warnings:

| Code | Condition |
|---|---|
| `dependency_archived_incomplete` | a live ticket depends on an archived ticket whose `from_status` is not `done` |
| `claim_expired` | a claim is past its `expires_at` |
| `reference_path_unresolved` | a `references` path does not resolve against the repository root, per 5.5 |
| `label_unknown` | a label is outside the `config.yml` allowlist |
| `milestone_unknown` | `milestone` is outside the `config.yml` allowlist |
| `in_progress_unclaimed` | a ticket is `in-progress` with no claim |
| `blocks_on_no_children` | `blocks_on` is `children` and no ticket names this one as its parent |

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

Two findings have exactly one correct repair, and `check --fix` makes them:

| Code | The repair |
|---|---|
| `filename_id_mismatch` | rename the file to `<id>.md`, which section 4 fixes and leaves no second reading of |
| `archive_location_mismatch` | move the file to the directory the status implies, because 6.3 already rules the status wins |

Nothing else is repaired, and the rest are not near misses. `duplicate_id` has
to choose which file keeps the ID, which is a judgement about which ticket is
the real one. `dependency_missing` is repaired either by dropping the edge or by
creating the ticket, and only a person knows which was meant. `label_unknown`
and `milestone_unknown` are each either a typo in the ticket or a gap in the
allowlist. A tool that guessed at those would be wrong about half of them and
silent about it, which is worse than reporting and stopping.

## 12. Interfaces

### 12.1 CLI

```text
git ticket init   [--instructions]
git ticket list   [--status S --type T --priority P --label L --assignee A --milestone M --parent P]
git ticket ready
git ticket show   ID
git ticket search QUERY [--regex]
git ticket create --title T [--type --priority --label --assignee --milestone --parent --blocks-on --depends-on --description --plan --ac --dod]
git ticket update ID [--title --type --priority --description --milestone --parent --blocks-on --add-label --remove-label --assign --unassign]
git ticket status ID STATUS [--reason R]
git ticket claim  ID [--expires-in D] [--force]
git ticket release ID
git ticket link   ID [--depends-on OTHER | --ref proposal:x [--path P]]
git ticket unlink ID [--depends-on OTHER | --ref proposal:x]
git ticket ac     ID [--add TEXT] [--check N] [--uncheck N] [--remove N]
git ticket dod    ID [--add TEXT] [--check N] [--uncheck N] [--remove N]
git ticket plan   ID TEXT
git ticket note   ID TEXT
git ticket comment ID TEXT
git ticket summary ID TEXT
git ticket deps   ID [--transitive] [--dependents]
git ticket files  PATH   # the tickets that reference a path
git ticket check  [--strict] [--fix [--dry-run]]
git ticket archive ID [--reason R]
git ticket unarchive ID
git ticket migrate [--to N] [--dry-run]
git ticket instructions [--write]
git ticket schema
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

**A merge driver for ticket files** (`TKT-01M1HE7KX06FY8W1GYXH9MXGBP`). Whether
git-ticket ships one. Two agents each adding a ticket merge cleanly and two
editing the same ticket do not, which 7.1 hands to Git and leaves there. A driver
stays inside the 7.4 promise, because Git invokes the driver and the driver runs
no Git command itself. It would have to say which fields merge by union, which
are last-writer-wins, and which must always conflict, and how a user installs
one, since a driver needs both a `.gitattributes` entry and a `git config` line
that a repository cannot set for itself.

**External tracker integration** (`TKT-01M1HFQ5F4D4KF45A5PFQ39XST`). How
git-ticket integrates with one. Storing the identifier works, because 5.1 leaves
the reference namespace open and `decodeReferences` takes the ref verbatim, so
`jira:PROJ-1234` links today and `check` is content with it. Three gaps sit above
that storage. Nothing enforces a namespace, so an untyped `PROJ-1234` is accepted
and `JIRA:proj-1234` is a different reference from `jira:PROJ-1234`, which sits
badly with 5.1 calling a reference a typed stable identifier. There is no lookup
by ref, so "which ticket is PROJ-1234" falls to `search`, which matches
substrings across the body and so also finds a ticket that only mentions the
number in prose. `files PATH` covers the `file:` namespace and has no equivalent
for the others. And `extensions` has no mutation, so the one place 5.1 reserves
for a consumer's own fields round-trips through parse and render but can only be
written by hand-editing the file.

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

**What counts as a store** (`TKT-01M1HJTCYTZZHGGPCXT2SAJMFR`). Nothing decides
what makes a directory a store, so any directory that exists is one. `Open`
returns `store_not_found` only when its path is missing or is not a directory,
and a missing `config.yml` falls through to the defaults in 4.1, so opening an
empty directory yields a store with no tickets rather than an error. The CLI
inherits it, and `--store DIR` at a directory holding nothing answers
`ticket-list` with an empty array and exit 0. A typo therefore answers "no
tickets" where it should answer "no store", which is the one wrong answer that
reads exactly like a right one. An agent that lists and sees nothing concludes
there is no work.

The markers exist already. `init` writes `config.yml`, `tickets/` and `archive/`
under `.tickets`, and `Discover` walks up looking for that name, so 4 is the
definition everywhere except the check at open time. What has to be decided is
which marker is the contract, because that is a format promise rather than an
implementation detail. Requiring `config.yml` is the cheapest test and the
strictest, and it stops a store whose config was deleted from opening at all.
Requiring `tickets/` describes the store better and survives a missing config,
which 4.1 already treats as defaults. Requiring the name `.tickets` would break
`--store` pointed at a fixture directory, which the corpus relies on. Also open
is whether the rule belongs to `Open` or only to the CLI's store resolution,
since a caller passing an explicit path in Go has said what it means and a
person typing `--store` has not.

Twelve questions have left this list. They were numbered once, and the numbering
is the thing this section gave up. A number has to be chosen, and it collided
twice. Two settled questions both called themselves question 7, the module path
and the parent hierarchy. A third was filed as question 9 from a concurrent
worktree while question 9 already meant the merge driver. Commit messages and
pull request bodies from that period still cite numbers, and the subject of each
entry below is what decodes them.

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
the `config.yml` allowlist with all seven parked tickets carrying it, so
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
(`TKT-01M1HXHJXRFP7VMH7D35YNTG5H`), which is not built yet, so the answers live
here until 4 and 11 carry them.

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
`--strict` promotes warnings to errors, per 10.3, and CI runs `check --strict`,
so a warning is enforced exactly as hard as an error everywhere enforcement
happens. Keeping it a warning holds the line that a derived file falling behind
is not a malformed store, which is the same line that keeps "this ticket is
late" out of `check` entirely. `TKT-01M1HXPJP7EGJVC2J7GC6Y887E` records the same
answer, because the two cannot be allowed to diverge.

Severity does not gate repair, which is the fact that makes the warning free.
`planRepairs` recomputes where each file belongs rather than walking the
findings, so a warning is exactly as repairable as an error. What does need work
is the shape rather than the severity: a `Repair` today is a rename, carrying
`From` and `To`, and rewriting a generated file is not a rename. However that
lands it is an implementation cost, not a second format decision.

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
