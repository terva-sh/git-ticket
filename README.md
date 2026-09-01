# git-ticket

Work tracking that lives in the repository it tracks. A ticket is a Markdown
file with YAML frontmatter, committed next to the code, editable in vim and
reviewable in `git diff`. One Go library owns the format, and a `git-ticket`
binary exposes it as `git ticket …`.

> **Nothing is built yet.** The format is frozen and covered by 35 test
> fixtures, but there is no Go code, no module, and nothing to install. See
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

## Status

| Phase | What | State |
|---|---|---|
| 0 | Format and fixtures | Done |
| 1 | Core library: parse, render, validate, query, `Apply` | Next |
| 2 | Standalone CLI with `--json` | Not started |
| 3 | Terva integration | Tracked in terva, starts after Phase 2 tags |
| 4 | MCP adapter, Backlog.md import, a local view | Deferred |

Two questions are open and recorded in section 15 of the plan rather than
guessed at: where a `blocked` reason is stored, and what a `references` path
resolves against. Both affect Phase 1.

## Reading order

1. [`docs/plan.md`](docs/plan.md) is the design of record. Sections 4 through 11
   are the format itself, section 13 is the phase plan.
2. [`testdata/README.md`](testdata/README.md) explains the fixture corpus and
   what an expectation sidecar means.
3. [`AGENTS.md`](AGENTS.md) is the standing rules for working in this
   repository.

`python3 scripts/check-corpus.py` from the repository root checks the corpus
against the plan. It is scaffolding, and Phase 1's Go tests replace it.
