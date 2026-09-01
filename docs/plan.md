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
and the optional stdio adapter. Terva's integration is Phase 3 and lives in
terva's own repository at `docs/plans/git-ticket.md`, because it is committed
work for terva rather than for this module.

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
defaults:
  type: task
  priority: normal
  claim_expiry: null
lock:
  timeout: 10s
```

`labels` is an advisory allowlist: `check` warns about a label outside it and
never errors. Configuration sets defaults and vocabulary. It cannot add a
status, change a transition rule, or grant a consumer authority it does not
otherwise have.

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
`low`, `normal`, `high`, `urgent`.

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

## 8. Query surface

- `list` with filters on status, type, priority, label, assignee, and milestone.
  Within one filter the values are alternatives and across filters they all have
  to hold. Archived tickets are left out unless the caller asks for them, by
  filtering on the `archived` status or by asking for them outright, because a
  list of work is about work that is still live
- `ready`: status `ready`, no live claim, and every dependency satisfied per 6.3.
  Only direct dependencies are read, so a dependency cycle cannot make this loop
- `show` for one complete ticket
- `search` over title, description, acceptance criteria, definition of done,
  notes, comments, summary, and references. Case-insensitive substring by
  default, `--regex` for RE2
- `deps` for direct and transitive dependencies, `--dependents` for the reverse
- `files PATH` for tickets referencing a path. This searches recorded `file:`
  references and is only as complete as the agents that wrote them. It is
  advisory and not derived from Git history, and the help text says so
- `check`, described in section 10

Search reads every file on every call. At the scale this format targets,
hundreds to a few thousand tickets, that is a few milliseconds and needs no
index. An index is deferred, and if one is ever added it must be disposable and
rebuildable from the files.

## 9. Mutation surface

Every mutation changes only the fields the caller named. Full-file replacement
is not an operation the API offers.

- `init`, `create`, `update`
- `status`, `claim`, `release`
- `link` and `unlink` for dependencies and references
- `ac` and `dod` to add, check, and uncheck criteria items
- `note` and `comment` to append
- `summary` to set. It replaces rather than appends, because a summary is one
  statement of where the ticket landed and a log of those is what `Notes` and
  `Comments` already are
- `archive`, `unarchive`

Each returns the resulting ticket, its new revision, and the paths changed.

## 10. JSON contract

Every machine-readable operation emits a versioned envelope on stdout:

```json
{ "schemaVersion": 1, "kind": "ticket-list", "tickets": [] }
```

Kinds are `ticket`, `ticket-list`, `mutation-result`, `check-report`, `error`,
`schema`, and `instructions`. Absent scalars are `null` and absent collections
are `[]`, always present rather than omitted, so a consumer never has to
distinguish missing from empty.

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
    "comments": "",
    "summary": "",
    "extra": [{ "heading": "Risks", "text": "…" }]
  },
  "checklists": {
    "acceptanceCriteria": [
      { "index": 1, "checked": true, "text": "The verifier accepts either key" },
      { "index": 2, "checked": false, "text": "New tokens use the newer key" }
    ],
    "definitionOfDone": []
  }
}
```

The frontmatter fields keep their names in camel case. `revision` and `path` are
computed rather than stored, per 7.1. `unknown` holds the top-level fields this
version does not define, per 5.4, which are preserved on write whether or not a
consumer understands them.

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
  "warnings": []
}
```

A finding carries `code`, `file`, `ticket`, and `field`. `ticket` is null when
the file did not parse far enough to know its ID, and `field` is null when the
finding is about the file rather than one field. Findings are ordered by file,
then code, then field, so two reports of the same store compare directly instead
of having to be treated as sets.

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
{ "schemaVersion": 1, "kind": "instructions", "text": "## Tickets\n\n…" }
```

The block is prose, so the envelope holds it whole rather than pretending it has
structure a consumer would want to walk. In human mode the command prints the
Markdown alone, because the point of it is `git ticket instructions >> AGENTS.md`.

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
| `invalid_status` | a `status` outside the set in 6.1 |
| `invalid_type` | a `type` outside the set in 5.1 |
| `invalid_priority` | a `priority` outside the set in 5.1 |
| `archive_location_mismatch` | the status and the directory disagree; the status wins |

Warnings:

| Code | Condition |
|---|---|
| `dependency_archived_incomplete` | a live ticket depends on an archived ticket whose `from_status` is not `done` |
| `claim_expired` | a claim is past its `expires_at` |
| `reference_path_unresolved` | a `references` path does not resolve against the repository root, per 5.5 |
| `label_unknown` | a label is outside the `config.yml` allowlist |
| `in_progress_unclaimed` | a ticket is `in-progress` with no claim |

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

## 12. Interfaces

### 12.1 CLI

```text
git ticket init   [--instructions]
git ticket list   [--status S --type T --priority P --label L --assignee A --milestone M]
git ticket ready
git ticket show   ID
git ticket search QUERY [--regex]
git ticket create --title T [--type --priority --label --assignee --parent --depends-on --description]
git ticket update ID [--title --type --priority --milestone --parent --add-label --remove-label --assign --unassign]
git ticket status ID STATUS [--reason R]
git ticket claim  ID [--expires-in D] [--force]
git ticket release ID
git ticket link   ID [--depends-on OTHER | --ref proposal:x [--path P]]
git ticket unlink ID [--depends-on OTHER | --ref proposal:x]
git ticket ac     ID [--add TEXT | --check N | --uncheck N]
git ticket dod    ID [--add TEXT | --check N | --uncheck N]
git ticket note   ID TEXT
git ticket comment ID TEXT
git ticket summary ID TEXT
git ticket deps   ID [--transitive] [--dependents]
git ticket files  PATH   # the tickets that reference a path
git ticket check  [--strict]
git ticket archive ID [--reason R]
git ticket unarchive ID
git ticket instructions
git ticket schema
```

Global flags: `--json`, `--store PATH`, `--if-revision R`, `--actor ID`,
`--lock-timeout D`. They are accepted before or after the subcommand, because
`git ticket --json list` and `git ticket list --json` are both what a person
types.

The store is found in this order: `--store`, then `GIT_TICKET_STORE`, then
discovery walking up from the current directory to the Git root. `config.yml`
has no say, because it lives inside a store and cannot name one.

A `--store` or `GIT_TICKET_STORE` that names a directory holding no store is
`store_not_found`, and never a fall back to discovery. Searching elsewhere after
being told where to look is how a tool writes to the wrong store.

`instructions` prints an agent workflow block for pasting into a project's
`AGENTS.md` or equivalent, per 10.5. The block tells an agent how to find work,
claim it, record what it learned, and finish, and it names only commands this
binary has. A test holds it to that, because prose that tells a reader to run
something that does not exist is worse than no prose.

`init --instructions` writes the block to `AGENTS.md` at the repository root. It
refuses when that file already exists, naming `git ticket instructions` so the
user can append it themselves, and it checks before creating the store so a
refusal leaves nothing half-built. Without the flag `init` writes no such file.
That file is one the user maintains, and merging into it is their edit to make.

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
func (s *Store) Check(ctx context.Context) (*Report, error)
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

Package layout:

```text
git-ticket/
├── cmd/git-ticket/     the binary
├── ticket/             the public library
├── internal/cli/       flag parsing and human rendering
├── testdata/fixtures/  the corpus from Phase 0
└── docs/plan.md
```

The library must not import a CLI framework, and nothing in `ticket/` may import
terva.

### 12.3 Optional stdio adapter

`git-ticket mcp` speaks MCP over stdio against the same library. It is a
process-local adapter, not a service. It performs no permission decisions of its
own; the host decides what a caller may do.

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

Owned by terva, tracked in terva's `docs/plans/git-ticket.md`. It starts once
Phase 2 tags a release, so that terva builds against a stable module rather than
a moving one.

### Phase 4: adapters and views

The stdio MCP adapter. Import of the useful Backlog.md fields, with no runtime
dependency on Backlog.md. A local browser or TUI view, and only after the file
and agent contracts have held through at least one real project. Explicit remote
helpers evaluated separately, and never as a side effect of a mutation.

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

1. How a caller renews an existing claim rather than replacing it.
2. Whether custom statuses are worth the cost, decided after a real workflow
   asks for one.
3. Whether a later release adds sync helpers around ordinary Git commands.
4. What tool discovery the stdio adapter should expose.
5. When Backlog.md import and a local view are worth building.
6. The compatibility policy for the Go module after the first stable schema.
Six questions have left this list, and the numbers have closed up behind them.

Two were settled during Phase 1, before any reader had shipped, so adding
`status_reason` to schema 1 cost no compatibility. Where a `blocked` reason
lives is answered by `status_reason` in 5.1 and 6.2: the field holds the
current reason and `Notes` keeps the history. What a `references` path resolves
against is answered in 5.5: the root of the Git repository holding the store,
and no finding at all when the store sits outside one.

The module path was settled by publishing rather than by argument. `go.mod`
declares `github.com/terva-sh/git-ticket` and a public mirror now serves that
path, so the declared path is the real one and nothing changes. The alternative,
renaming the module to the host that serves it, was wrong on its own terms: a
private hostname does not belong in a public artifact, and a module path is the
most load-bearing string in a Go project to change later. See 12.2.

Three were settled during Phase 2, because building the CLI is what forced
them. The precedence between `--store`, `GIT_TICKET_STORE`, and `config.yml` is
answered in 12.1: the CLI is the first thing that had to resolve a store.
Whether an archive reason also goes to `Notes` is answered in 6.3, and it does,
on the same argument that puts a status reason there. Whether `update` carries
`--type` and `--parent` is answered by 12.1 listing both, because every other
field `create` sets already had an `update` flag and the library already had
`SetType` and `SetParent`.

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
- terva's `docs/plans/git-ticket.md` for Phase 3.
