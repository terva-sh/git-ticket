# git-ticket

Work tracking that lives in the repository it tracks. A ticket is a Markdown
file with YAML frontmatter, committed next to the code, editable in vim and
reviewable in `git diff`. One Go library owns the format, and a `git-ticket`
binary exposes it as `git ticket …`.

> **Partly built.** The library and the 28-command CLI are done and tagged
> `v0.5.0`. That tag predates the release pipeline and carries no binaries; the
> next one publishes archives for Linux, macOS and Windows. See
> [Status](#status) before you plan around this.

## The problem

A project outlives an agent session. Over one ticket's life it may be touched by
Claude Code, by Codex, by terva, by a shell script, and by a person with an
editor open. They should all be working on the same files, and none of them
should have to scrape another tool's terminal output to do it.

A session task list cannot do this: it dies with the session. A hosted tracker
cannot either, because it is not in the clone, so it is not in the diff and not
in the review.

## What a ticket is

`.tickets/tickets/TKT-01K3ZZ2JH000GHB4EE6SNRE6MD.md`:

```markdown
---
schema: 1
id: TKT-01K3ZZ2JH000GHB4EE6SNRE6MD
title: Refresh fails when the clock jumps backward
type: bug
status: ready
status_reason: null
priority: high
due_on: null
labels:
  - auth
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-08-31T12:01:00Z
updated_at: 2026-08-31T12:01:00Z
created_by:
  id: human:sothr
  name: Drew Short
updated_by:
  id: human:sothr
  name: Drew Short
extensions: {}
---

## Description

The token refresh compares wall-clock times and assumes the second is later.
```

The filename is the ID and nothing else. Putting the title in the filename means
a title change renames the file, which breaks `git log` on the old path.

When an agent picks the ticket up, it writes a claim:

```yaml
claim:
  actor: agent:terva/session-123
  branch: fix/clock-jump
  worktree: /Users/sothr/wt/clock-jump
  commit: 9f8e7d6c5b4a
  claimed_at: 2026-09-01T09:00:00Z
  expires_at: null
```

A ticket sits in the directory its status implies, so `ls` answers what is worth
looking at:

```text
.tickets/
├── config.yml
├── README.md
├── epics.md   generated index of the open epics
├── draft/     filed, not yet worth starting
├── tickets/   the working set: ready, in-progress, blocked, review
├── done/      finished recently, still worth reading
└── archive/   retired, swept out of done from time to time
```

That split is for a person, not the tool. `list` already answers with open work,
so a binary never needed it. Somebody reading the store in a forge web UI has a
directory as the only query they get for free, and `tickets/` holding the working
set alone is what makes that answer worth having.

The directories key on status, which leaves type with no answer at all in a file
browser. `epics.md` is that answer for the one type worth browsing: a table of
the epics that are not done or archived, each linking to its file. `check`
reports it stale when it disagrees with the tickets and `check --fix` rewrites
it, so it is generated output rather than something to maintain.

The working statuses share one directory because a directory each would make
every ordinary transition a rename. A ticket moves three times: when somebody
decides it is worth starting, when it is finished, and when it is archived. The
status wins if the two ever disagree, so `check` reports a file in the wrong
place as `location_mismatch` and `check --fix` moves it, which is also how an
older store migrates.

## The decisions worth knowing

**A claim is advisory, never a lock.** It records who is working, on what
branch, from which commit. A claim made in a clone you have not fetched is
evidence, not a reservation, and the tool says so rather than pretending to
coordinate across machines it cannot see.

**Staleness is a hash of the file bytes, not a field.** Every read returns
`revision`, a SHA-256 of the file as it sits on disk. Nothing stores it. A
stored counter or an `updated_at` fails silently the moment a person edits the
file by hand, which is the case that matters most here, and a counter has no
correct value after two branches both increment it.

**Hand-editing is a supported operation.** Not tolerated, supported. That is why
the format is Markdown and accepts Markdown's merge behaviour instead of using a
database, and it is why `git ticket check` exists as the safety net. Conflict
markers in a ticket are reported as `merge_conflict` rather than as a YAML
syntax error, because that is what actually happened.

**A merge driver is optional and worth installing.** Two agents editing
different fields of the same ticket still collide in a line merge, because every
write touches `updated_by`. `git ticket install-merge-driver` resolves those
per field and marks only the ones that genuinely disagree.

**Nothing touches the network.** No fetch, push, merge, branch switch, or commit
happens as a side effect of a mutation. Publishing a ticket change is `git
commit` and `git push`, the same as publishing code.

**An unknown field is preserved, not dropped.** A reader one minor version
behind keeps a field it does not understand and writes it back out. `check`
reports it, an ordinary read warns. Dropping it would corrupt the ticket for
everyone else sharing the repository.

## How it differs from the neighbours

[git-bug](https://github.com/MichaelMure/git-bug) stores records in Git objects
rather than worktree files. That gets clean merges, at the cost of records no
one can open in an editor. This project takes the opposite trade.

[Beads](https://github.com/gastownhall/beads) is agent-first, with a Dolt
database as the source of truth and JSONL as an export. The database is the part
this project rejects: an export is not something you review in a pull request.

[Backlog.md](https://github.com/MrLesk/Backlog.md) is the strongest Markdown
workflow of the three, and the acceptance criteria and definition of done fields
come from it. It puts the title in the filename, which this format does not.

## Using it

A release carries archives for Linux, macOS and Windows, on amd64 and arm64.
Download one, unpack it, and put `git-ticket` on your `PATH`. The binary sits at
the root of the archive, and `checksums.txt` beside it verifies what you
downloaded. Nothing else is needed: no Go toolchain, no runtime.

With a Go toolchain, this gets you the same command:

```sh
go install github.com/terva-sh/git-ticket/cmd/git-ticket@latest
```

That lands `git-ticket` in your `GOBIN`, or `GOPATH/bin` without one. Git spells
a binary named `git-ticket` on `PATH` as `git ticket`. From a clone, `just
install` does the same thing and `just build` puts the binary in the repository
root instead. That root copy is gitignored and nothing rebuilds it for you, so
prefer `just ready` and `just check`, which depend on `build`.

`git ticket --version` reports the tag and the commit the binary was built from,
and says so honestly: a build from a tree with no tag reachable answers `devel`
rather than inventing a version.

A ticket's whole life works today:

```sh
git ticket init --actor human:you
git ticket create --title "Refresh fails when the clock jumps backward" \
    --type bug --priority high --label auth
git ticket list --status ready --type bug
git ticket search "clock jump" --type bug
git ticket ready                  # what could be started right now
git ticket files auth/verify.go   # which tickets reference this path
git ticket show TKT-01K3ZZ2J      # a unique prefix, with or without TKT-

git ticket update TKT-01K3ZZ2J --priority urgent --add-label crypto
git ticket ac TKT-01K3ZZ2J --add "The verifier accepts either key"
git ticket link TKT-01K3ZZ2J --depends-on TKT-01K4001C
git ticket link TKT-01K3ZZ2J --ref proposal:git-ticket --path docs/plan.md
git ticket deps TKT-01K3ZZ2J --transitive

git ticket status TKT-01K3ZZ2J ready
git ticket claim TKT-01K3ZZ2J     # records your branch and HEAD
git ticket status TKT-01K3ZZ2J in-progress
git ticket plan TKT-01K3ZZ2J "1. Reproduce with a stepped clock
2. Widen the window
3. Pin the source"
git ticket note TKT-01K3ZZ2J "the skew is 40s, not the 5s we assumed"
git ticket ac TKT-01K3ZZ2J --check 1
git ticket summary TKT-01K3ZZ2J "Widened the window and pinned the clock source"
git ticket status TKT-01K3ZZ2J done
git ticket archive TKT-01K3ZZ2J --reason "shipped in v1.2"

git ticket check --strict         # safe in CI: offline, read-only
git ticket check --fix            # repair the two findings that have one repair
git ticket schema                 # the values and codes this binary enforces
git ticket instructions           # the agent workflow block, for an AGENTS.md
git ticket instructions --write   # put it in AGENTS.md, or refresh it in place

git ticket install-merge-driver   # per-field merges instead of line merges
```

`release` and `unarchive` undo the two that are undoable, and `dod` edits the
definition of done exactly as `ac` edits the acceptance criteria.

`update` takes as many flags as you like and applies them as one write. Either
all of them land or none do, so an update that fails partway leaves a ticket in
a state somebody typed rather than half of one. An empty value clears a field
and an absent flag leaves it alone, so `--milestone ""` and no `--milestone` at
all are different instructions.

Every field `create` sets, `update` can change, `--type` and `--parent`
included. A bug that turns out to be a chore, and a ticket that belongs under a
different epic, are fixable without opening the file.

The number `ac --check N` takes counts checkbox lines from one, not lines and
not array positions. A section can hold prose above its list and still number
its items 1, 2, 3, and editing a box leaves that prose alone.

`ac` and `dod` take `--add`, `--check`, `--uncheck`, and `--remove`, each
repeatable and all combinable in one call that lands as one write. Every index
means the item you read when you typed it, so
`ac ID --check 3 --remove 1 --add "a fourth"` ticks the third box you can see,
drops the first, and appends. Removals run highest first for that reason, and
naming one twice removes one item.

`create` seeds the same two sections with `--ac` and `--dod`, both repeatable,
so a ticket can be filed complete in one command rather than one call per
criterion.

`link` takes one of `--depends-on` or `--ref`, and `--path` goes with the
second, because a path with no reference names nothing. `unlink` is the reverse,
and removing something that is not there succeeds rather than complaining.

`list` answers with open work: every status except `done` and `archived`. Naming
a status brings it back, so `list --status done` still works, and `--all` drops
the exclusion. The rule is an exclusion rather than a list of the open statuses,
so a status added later is included without an edit. `git ticket schema`
publishes the open set as `openStatuses` rather than making a consumer hard-code
it.

`search` takes a regular expression and matches it against the title and the
body, and it takes every filter `list` takes, so you narrow by status or type
and search inside that. It does not take the open default, though. Search is how
you find what was already decided, and a decision lives in a done ticket.

`ready` answers one question: which open tickets are not blocked and have every
dependency closed. That is the queue, and it is what an agent asks for when it
wants work.

Every ticket also carries a `readiness`, derived from the whole store at read
time and never stored. It holds the verdict `ready` filters on plus what stands
in the way, so `show` prints a `blocked by` line and a consumer drawing a board
can grey out a card and say why without calling `ready` and then `deps` per row.
`ready` filters on that same verdict rather than restating the rule, so the two
cannot come to disagree.

Blocked means dependencies. A draft, and a ticket somebody else is holding, are
both unready with nothing in the way but their own state. A dependency that
resolves to nothing, or to two files claiming one ID, blocks rather than
counting as satisfied, because a prerequisite nobody can point at is not one
anybody met.

`note`, `comment`, `plan`, and `summary` write the text sections of plan section
9. `note` and `comment` append with a timestamp and an actor, so they read as a
log. `plan` and `summary` set. Each is one statement rather than a log: a plan
says how the work will go and a summary says where it landed, and a log of
either is what the first two already are.

`--json` splits that comment log back out as `comments`, one record with an
actor, a time, and the text. An entry runs from its stamp to the next one, so a
comment can have several paragraphs. Text with no stamp above it, which is what
somebody editing the file by hand leaves, comes back with a null actor rather
than being dropped.

`files PATH` goes the other way, from a path to the tickets that recorded a
reference to it. A reference is written by `link --ref X --path P`, so `files`
reports what agents wrote and is only as complete as they were. Nothing derives
it from Git history, so it is a hint for a reader rather than a fact about the
tree.

`refs REF` answers the same shape of question for the rest of the namespaces.
`git ticket refs jira:PROJ-1234` finds the ticket tracking that work item, and
`git ticket refs jira:` lists everything in that namespace. The namespace is
compared without regard to case and the identifier exactly, so `JIRA:` and
`jira:` are one type while `PROJ-1234` and `proj-1234` stay two references,
because only the system that issued an identifier knows whether its case means
anything.

Use it instead of `search` when you want the ticket for a tracker item. A search
matches substrings across the whole body, so it also returns a ticket that only
mentions the number in passing. A ref with no namespace at all is still stored,
but nothing can look it up that way, and `check` says so with
`reference_untyped`.

Text that opens with a dash goes after a bare `--`, so
`git ticket note TKT-01K3ZZ2J -- "--force was the wrong default"` records the
note rather than failing on an unknown flag.

`deps` walks dependencies, `--transitive` follows the chain, and `--dependents`
walks it backwards to what is waiting on this ticket. A dependency cycle is a
state a store can genuinely be in, so the walk terminates on one and reports
what it found; `check` is where you are told the cycle exists.

Anywhere an ID is taken, a unique prefix works, with or without the `TKT-`
part, and four characters is the minimum. A listing shortens each ID to the
fewest characters that still resolve, so what you read off the screen is what
you can type back.

A status moves along the table in plan 6.2 and refuses anything else, naming
where the ticket may go instead. Entering `blocked` needs `--reason`, and so
does reopening from `done`, because a ticket that silently un-finishes makes the
status mean nothing. The reason goes in `status_reason` for a query to read and
into `Notes`, which survives the next transition clearing the field.

A claim is advisory. It records who is working, on which branch, from which
commit, and it reserves nothing. Claiming a ticket somebody else holds is
`claim_conflict`; `--force` takes it and writes the displaced claim into `Notes`.

`archive` is its own command rather than a status because it also moves the file
to `.tickets/archive/`, and it records `from_status`. That is what stops an
archived ticket from silently blocking its dependents: a dependency is satisfied
by a ticket archived out of `done`, and by nothing else.

An `archive --reason` lands in the archive block and in `Notes`, the same two
places a status reason lands. `unarchive` deletes the block, so without the note
a ticket archived as "shipped in v1.2" and then restored would keep nothing that
says why it was ever closed out.

Every command takes `--json` and answers with one envelope on stdout. A failure
exits 1 and puts the reason in an `error` envelope with a stable `code`, so a
script switches on the code rather than parsing the message. `--store PATH`
names a store, `GIT_TICKET_STORE` does the same from the environment, and
without either the store is discovered upward to the Git root.

Every write takes `--if-revision R` and refuses if the ticket moved since you
read it, answering `stale_revision`. Nothing stores that revision: it is a
SHA-256 of the file's bytes, so it notices a hand edit too.

`check` validates every ticket against section 11 of the plan: broken
dependencies, cycles, duplicate IDs, a filename that no longer matches its ID,
conflict markers left by a merge. It separates errors from warnings, and
`--strict` makes a warning fail the run too. A store with findings still gets a
`check-report`, not an `error`, because the check ran fine and the answer is no:

```json
{ "schemaVersion": 1, "kind": "check-report", "ok": false,
  "errors": [{ "code": "dependency_missing", "file": ".tickets/tickets/TKT-….md",
               "ticket": "TKT-…", "field": "dependencies" }],
  "warnings": [] }
```

`ok` is true exactly when the command exited zero, so CI gates on one field
whether or not it passed `--strict`.

`check --fix` repairs the two findings that have one correct repair, and
nothing else. A file named something other than its ID is renamed, and a file
in the wrong directory is moved to the one its status implies, because plan 6.3
already rules that the status wins. It moves the file and never rewrites the
ticket, since both findings are about where a file sits.

It is the only command that writes without being told which ticket to write, so
`--dry-run` shows the moves and makes none, and the report names every path it
touched in `pathsChanged` the way a mutation does.

Everything else is reported and left alone. A duplicate ID has to choose which
file is the real ticket, a missing dependency is repaired either by dropping the
edge or by creating what it names, and an unknown label is either a typo or a
gap in the allowlist. Each needs a person, so `--fix` does not guess. That also
covers a move onto a path already taken: the repair is dropped and the finding
stays.

`instructions` prints a workflow block to paste into a project's `AGENTS.md`,
telling an agent how to find work, claim it, record what it learned, and finish.
`git ticket instructions --write` puts it there for you, and
`git ticket init --instructions` does the same for a new project. Without the
flag, `init` writes no such file.

The block is fenced by `<!-- git-ticket:begin -->` and `<!-- git-ticket:end -->`,
so `--write` can replace it later and leave every other byte of your file alone.
Run it again after upgrading and the block catches up. A file with no markers is
appended to rather than refused, which is what a project that already has an
`AGENTS.md` needs, and it is refreshable from then on. A file already current is
not rewritten at all, so a no-op stays out of your diff.

One marker without its partner is the one case it refuses. There is no honest
reading of where the block ends, and guessing would delete prose you wrote.

A test holds the block to the commands and flags this binary actually has, so it
cannot tell you to run something that does not exist. It caught the first draft
telling agents to run `git ticket files ID --add PATH`.

`schema` prints what the binary enforces: the statuses, types, and priorities,
the transition table, every error code, and every check finding paired with its
severity. Each list is read from the code that enforces it, not copied into the
command, so a value the library accepts and the plan forgot still shows up.
Write a consumer against `schema` rather than against a hard-coded list, and it
keeps working when the sets grow. It reads no store, so it answers outside a
repository and before `init`.

## Status

| Phase | What | State |
|---|---|---|
| 0 | Format and fixtures | Done |
| 1 | Core library: parse, render, validate, query, `Apply` | Done |
| 2 | Standalone CLI with `--json` | Done, tagged `v0.1.0`. All 24 commands it scoped, and both exit criteria met |
| 3 | Terva integration | Started. `v0.2.0` exports the `cli` package terva embeds, `v0.3.0` adds the `--parent` filter its board needs, `v0.4.1` reports the files a query could not read, `v0.5.0` gives every unready ticket a reason and merges concurrent edits per field. Tracked in terva |
| 4 | MCP adapter, Backlog.md import, a local view | Deferred |

The two questions that blocked Phase 1 are settled. A `blocked` reason lives in
the `status_reason` field and in `Notes`, per plan 5.1 and 6.2. A `references`
path resolves against the root of the Git repository holding the store, and is
not checked at all when the store sits outside one, per plan 5.5.

Phase 2 settled three more, all in the plan. Store precedence is `--store`, then
`GIT_TICKET_STORE`, then discovery, and a named store that does not exist is an
error rather than a reason to go looking elsewhere, per 12.1. Exit statuses are
Git's: 0 or 1, with the detail in the error code, per 10.2. A ticket's JSON
carries every body section as raw text and derives two views from it, per 10.1:
`checklists`, each item numbered the way `ac --check N` counts, and `comments`,
one record per stamp so a reader draws a thread without parsing Markdown.

Wiring `check` settled a fourth, recorded as 10.3. `--strict` moves no finding
between `errors` and `warnings`, because those arrays report severity as the
format defines it. Strictness is a policy on top, visible in `ok` and the exit
status alone, so `ok` can be false with an empty `errors` array.

Writing the commands raised two more, and both are now settled. An archive
reason goes to `Notes` as well as the `archive` block, per 6.3, because
`unarchive` deletes the block and a status reason lands in two places for
exactly that reason. And `update` carries `--type` and `--parent`, per 12.1,
because every other field `create` sets already had an `update` flag.

The module path is settled too, by publishing rather than by argument. `go.mod`
declares `github.com/terva-sh/git-ticket`, a public mirror serves that path, and
so the import path a consumer writes is the one that was already there.

The compatibility policy is settled as well, in 12.4. The module version, the
file `schema`, and the JSON `schemaVersion` move independently; everything a
machine reads is covered and the human output is not; and a store never upgrades
itself, so a new binary cannot make a repository unreadable to a colleague still
on the old one. The module stays `v0.x` until Phase 3 lands.

The parent hierarchy is settled too, in section 8. `list --parent ID` gives an
epic's direct children that are still open, and `list --parent none` gives the
tickets with no parent, which is what a board needs for its top level. `--parent`
takes the open default like every other filter, so add `--all` for an epic's
whole roster. Nothing that reasons about children reads a listing: `readiness`
computes `blockingChildren` from the whole store, so a done child still counts
as satisfied. `deps` still walks `dependencies` alone, but when its answer is
empty and the ticket has children it now names the count and points at that
filter, because "It depends on nothing." is true and useless on an epic.

That question shrank when it was run rather than argued. `list --json` already
carried every `parent` edge, so the tree was always reconstructable from one
call and nothing was blocked. What was missing was the asking, not the data,
which is why the answer is a filter and not a new field on the ticket.

What remains is deferred rather than open. The parked questions are filed as
tickets carrying the `question` label, so `git ticket list --label question`
answers with the current set rather than a count somebody has to maintain. Plan
section 15 carries the prose for each question it has settled or parked, naming
the subject and the ULID of its ticket. Numbering was tried and dropped, because
a number assigned by hand collides when two agents work at once, and a ULID is
generated.

The schema migration question arrived the same way, out of settling the
compatibility policy. Ruling that a store never upgrades itself leaves the
migration a person would run undesigned, and nothing should bump the schema
before that exists.

## Reading order

1. [`docs/plan.md`](docs/plan.md) is the design of record. Sections 4 through 11
   are the format itself, section 13 is the phase plan.
2. [`testdata/README.md`](testdata/README.md) explains the fixture corpus and
   what an expectation sidecar means.
3. [`AGENTS.md`](AGENTS.md) is the standing rules for working in this
   repository.

`just test` runs everything, including the tests that hold the fixture corpus to
the plan: every code in section 11 has a fixture, and every fixture reproduces
the findings recorded beside it. `just ci` adds the race detector, gofmt, vet,
and `check --fix --dry-run --strict` over this repository's own store, which is
what CI runs. That plans every repair and writes nothing, so a pending one fails
the build and `just fix` settles it locally. Plan section 11 is why CI reports
the repair rather than committing it.
The recipes are plain `go` invocations, so working without just costs you
nothing but typing.

## License

MIT, in [`LICENSE`](LICENSE). Every release archive carries a copy beside the
binary, so something downloaded from a release page ships with its own terms and
needs nothing from this repository to be used.
