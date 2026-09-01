# git-ticket

A Git-native work ledger: Markdown tickets in `.tickets/`, one Go library, a
`git ticket` CLI. Read `docs/plan.md` before you change anything. It is the
design of record, not background reading.

## State

Phase 0 and Phase 1 are done and their exit criteria hold. The library parses,
renders, validates, queries, and mutates, with the store lock and the revision
precondition.

Phase 2, the standalone CLI in section 12.1, is nearly done. `cmd/git-ticket`
and `internal/cli` carry 22 of the 24 commands: `init`, `create`, `update`,
`show`, `list`, `search`, `ready`, `status`, `claim`, `release`, `link`,
`unlink`, `deps`, `files`, `ac`, `dod`, `note`, `comment`, `summary`,
`archive`, `unarchive`, and `check`, each in a human form and behind `--json`.
All five JSON kinds of section 10 have a test, and every write honours
`--if-revision`.

What remains in 12.1 is `instructions` and `schema`. `instructions` prints an
agent workflow block for a project's `AGENTS.md`, and what that block says is a
content decision. Ask before writing it.

The exit criterion about a scripted end-to-end run is met by `TestLifecycle` in
`internal/cli/lifecycle_test.go`. The other, `git ticket check` green in CI,
needs this repository to keep its own tickets in `.tickets/`, which it does not
yet do.

Section 13 of the plan lists the phases and their exit criteria. Do not start a
later phase before an earlier one meets its criteria.

The repository is local only, on `main`, with no remote.

## Commands

```sh
go test ./...
```

That is the whole suite, including the tests that hold the corpus to the plan.
Run it after touching anything under `testdata/` or section 11 of the plan.

## Layout

`docs/plan.md` is the format, the store, the CLI, and every decision behind
them. Sections 4 through 11 are the format itself.

`testdata/` is the fixture corpus. Read `testdata/README.md` before adding to
it.

`cmd/git-ticket/` is a thin `main` and holds no decisions. Every choice about
flags, output, and exit status lives in `internal/cli/`, where the tests can
reach it: `Run(args, Env)` takes the directory, the environment, both streams,
and the clock as arguments rather than reading process state.

`ticket/` is the library. `ticket/corpus_test.go` holds the corpus to its own
rules: every fixture pairs with a sidecar, every code in section 11 has a
fixture, and no sidecar names a code the plan does not define. It replaced
`scripts/check-corpus.py`, which did the same job from outside until there was a
real parser to do it from inside.

## The plan is authoritative

Do not settle a format question in code. If the code needs the format to
change, change `docs/plan.md` in the same commit and say why in the message.

When the plan does not answer a question, add it to section 15 instead of
inventing an answer. The two that blocked Phase 1 are settled: a `blocked`
reason lives in `status_reason` and in `Notes`, per 5.1 and 6.2, and a
`references` path resolves against the Git repository root, per 5.5. The end of
section 15 records both.

## Conventions

Names are singular: `git ticket`, the `ticket_*` tools, the `ticket` package.
Directories stay plural: `.tickets/`, `tickets/`, `archive/`.

Fixtures are static files a person reviewed. Never generate one at test time.
Every fixture carries an `.expected.json` sidecar even when it is clean,
because a missing expectation cannot be told apart from a forgotten one.

Time-dependent behaviour uses the fixed reference instant
`2026-09-30T00:00:00Z`. Inject it. Never read the system clock in a test.

## Policy

This repository owns the format, the store, validation, and the CLI. Nothing
here imports terva. The terva integration is Phase 3 and is tracked in terva's
own `docs/plans/git-ticket.md`.

The library reaches no network and runs no Git command that writes. No fetch,
push, merge, commit, or branch switch. Publishing is the user's ordinary Git
workflow, and a helper that does it for them is out of scope for v1.

## Gotchas

`go:embed` silently skips any path whose name starts with `.` or `_`. That is
why store fixtures live under `store/` and not `.tickets/`. The realistic name
would embed nothing and leave a suite of tests passing against an empty corpus.

Renaming a code in section 11 fails `TestCorpusCoversEveryPlanCode` until the
sidecars follow. That is intended. The corpus and the spec are one artifact.

The standard library's `flag.Parse` stops at the first non-flag word, so a
naive parse never sees the `--json` in `git ticket show ID --json`. `parseFlags`
in `internal/cli/cli.go` loops instead, consuming one positional per pass. Plan
12.1 requires both orders.

A path printed to a person goes through `displayPath`, never `filepath.Rel`
alone. On macOS a temporary directory is reached through `/var`, a symlink to
`/private/var`, so `git rev-parse --show-toplevel` answers in one name space and
the store path is in another. `displayPath` retries through `EvalSymlinks`.

The library takes canonical IDs; the CLI turns what a person typed into one.
Plan 5.5 says any command taking an ID accepts a unique prefix, so anything
passing an ID into a mutation goes through `resolveID` first. `create` shipped
without that and rejected a valid prefix with `dependency_missing`.

`unlink --depends-on` is the exception: it resolves against the ticket's own
dependency list, not the store. A dependency naming a ticket that does not
exist is the `dependency_missing` that `check` reports, and `unlink` is the
repair, so resolving through the store would make it unrepairable.

A listing never abbreviates an ID to a fixed width. A ULID opens with ten
characters of timestamp, so tickets created in the same millisecond are
identical that far in and a fixed eight printed the same prefix on several
rows. `shortestUnique` shortens to what actually resolves across the store.
For the same reason, a test must not assume ID sort order matches creation
order within a millisecond: compare as a set.

A flag whose zero value is a legal instruction needs `fs.Visit`, not a check
for emptiness. `update --milestone ""` clears the field and no `--milestone` at
all leaves it alone, and both arrive as an empty string. `runUpdate` and
`runChecklist` capture the `*flag.FlagSet` in their register closure to read
which flags were actually given.

`displayPath` handles a path that no longer exists, through `evalExisting`. A
mutation that moves a file reports the old location after deleting it, and
`EvalSymlinks` fails on a path that is not there, so archiving used to emit one
relative and one absolute path in the same `pathsChanged`. A test for this needs
a real repository, because with no root at all an absolute path is correct and
the assertion passes for the wrong reason.

A `ticket.Finding` names its file relative to the store, and the `.expected.json`
sidecars record that form, but every path in the JSON contract is relative to
the repository root. `findings()` in `internal/cli/json.go` converts. A test
that only matches the path suffix passes either way, so
`TestCheckAgreesWithTheCorpus` stats the path from the repository root instead.

`reference_path_unresolved` is the one check whose result depends on where the
store sits. The library tests inject the fixture's case directory as the root;
the CLI takes the root from git and gets this repository. The two disagree about
the `reference-unresolved` fixture on purpose, and the CLI corpus test skips
comparing its findings for that reason.

Adding a frontmatter field means editing every fixture that carries one, because
5.3 renders an absent scalar as `null` rather than omitting it. The round-trip
test fails on all 32 of them until they agree with the renderer. `status_reason`
cost exactly that.
