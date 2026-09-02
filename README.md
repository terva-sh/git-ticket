# git-ticket

Work tracking that lives in the repository it tracks. A ticket is a Markdown
file with YAML frontmatter, committed next to the code, editable in vim and
reviewable in `git diff`. One Go library owns the format, and a `git-ticket`
binary exposes it as `git ticket …`.

> **Partly built.** The library is done and the CLI carries four commands.
> There is no release and no install path yet. See [Status](#status) before you
> plan around this.

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
priority: high
labels:
  - auth
assignees: []
milestone: null
parent: null
dependencies: []
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

```sh
go build -o git-ticket ./cmd/git-ticket
```

Put that on `PATH` and Git spells it `git ticket`. A ticket's whole life works
today:

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
git ticket note TKT-01K3ZZ2J "the skew is 40s, not the 5s we assumed"
git ticket ac TKT-01K3ZZ2J --check 1
git ticket summary TKT-01K3ZZ2J "Widened the window and pinned the clock source"
git ticket status TKT-01K3ZZ2J done
git ticket archive TKT-01K3ZZ2J --reason "shipped in v1.2"

git ticket check --strict         # safe in CI: offline, read-only
git ticket schema                 # the values and codes this binary enforces
git ticket instructions           # the agent workflow block, for an AGENTS.md
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

`link` takes one of `--depends-on` or `--ref`, and `--path` goes with the
second, because a path with no reference names nothing. `unlink` is the reverse,
and removing something that is not there succeeds rather than complaining.

`search` takes a regular expression and matches it against the title and the
body, and it takes every filter `list` takes, so you narrow by status or type
and search inside that. `ready` answers one question: which open tickets are
not blocked and have every dependency closed. That is the queue, and it is what
an agent asks for when it wants work.

`note`, `comment`, and `summary` write the three text sections of plan section
9. `note` and `comment` append with a timestamp and an actor, so they read as a
log. `summary` sets. A summary is one statement of where the ticket landed, and
a log of those is what the other two already are.

`files PATH` goes the other way, from a path to the tickets that recorded a
reference to it. A reference is written by `link --ref X --path P`, so `files`
reports what agents wrote and is only as complete as they were. Nothing derives
it from Git history, so it is a hint for a reader rather than a fact about the
tree.

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

`instructions` prints a workflow block to paste into a project's `AGENTS.md`,
telling an agent how to find work, claim it, record what it learned, and finish.
`git ticket init --instructions` writes it to `AGENTS.md` for a new project. It
refuses when that file already exists rather than touching a file you maintain,
and it checks before creating the store so a refusal leaves nothing half-built.
Without the flag, `init` writes no such file.

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
| 2 | Standalone CLI with `--json` | Done, tagged `v0.1.0`. All 24 commands, and both exit criteria met |
| 3 | Terva integration | Started. `v0.2.0` exports the `cli` package terva embeds. Tracked in terva |
| 4 | MCP adapter, Backlog.md import, a local view | Deferred |

The two questions that blocked Phase 1 are settled. A `blocked` reason lives in
the `status_reason` field and in `Notes`, per plan 5.1 and 6.2. A `references`
path resolves against the root of the Git repository holding the store, and is
not checked at all when the store sits outside one, per plan 5.5.

Phase 2 settled three more, all in the plan. Store precedence is `--store`, then
`GIT_TICKET_STORE`, then discovery, and a named store that does not exist is an
error rather than a reason to go looking elsewhere, per 12.1. Exit statuses are
Git's: 0 or 1, with the detail in the error code, per 10.2. A ticket's JSON
carries every body section as raw text and derives `checklists` from it, each
item numbered the way `ac --check N` counts, per 10.1.

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

What remains is deferred rather than open: seven questions in plan section 15,
filed as tickets in `.tickets/`. Run `git ticket list` to see them. A question
keeps its number for life, so a settled one leaves a gap rather than shifting
the rest, and each entry names the ULID of its ticket.

Q7 arrived the way the useful ones do. Filing this repository's own Phase 3
epic, with four slices under it, turned up a hole: the format validates a parent
hierarchy it gives no way to walk. `list` does not filter on `parent`, and
`deps` reads `dependencies`, so on an epic it reports that it depends on
nothing, which is true and useless.

Q8 arrived the same way, out of settling the compatibility policy. Ruling that a
store never upgrades itself leaves the migration a person would run undesigned,
and nothing should bump the schema before that exists.

## Reading order

1. [`docs/plan.md`](docs/plan.md) is the design of record. Sections 4 through 11
   are the format itself, section 13 is the phase plan.
2. [`testdata/README.md`](testdata/README.md) explains the fixture corpus and
   what an expectation sidecar means.
3. [`AGENTS.md`](AGENTS.md) is the standing rules for working in this
   repository.

`go test ./...` runs everything, including the tests that hold the fixture
corpus to the plan: every code in section 11 has a fixture, and every fixture
reproduces the findings recorded beside it.
