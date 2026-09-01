# git-ticket

A Git-native work ledger: Markdown tickets in `.tickets/`, one Go library, a
`git ticket` CLI. Read `docs/plan.md` before you change anything. It is the
design of record, not background reading.

## State

Phase 0, the fixture corpus, is done. Phase 1, the core library, is in
progress: parse and render hold the round-trip property on the corpus, and the
store, `Check`, and `Apply` are still to come. Section 13 of the plan lists the
phases and their exit criteria. Do not start a later phase before an earlier one
meets its criteria.

The repository is local only, on `main`, with no remote.

## Commands

```sh
go test ./...
python3 scripts/check-corpus.py     # run from the repository root
```

Run the corpus script after touching anything under `testdata/` or section 11 of
the plan. The two overlap on purpose until Phase 1 ends; see Layout.

## Layout

`docs/plan.md` is the format, the store, the CLI, and every decision behind
them. Sections 4 through 11 are the format itself.

`testdata/` is the fixture corpus. Read `testdata/README.md` before adding to
it.

`scripts/check-corpus.py` enforces the corpus invariants until Phase 1's Go
tests can assert them with a real parser. Delete it then. Do not maintain both.

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

Renaming a code in section 11 breaks `check-corpus.py` until the sidecars
follow. That is intended. The corpus and the spec are one artifact.

Adding a frontmatter field means editing every fixture that carries one, because
5.3 renders an absent scalar as `null` rather than omitting it. The round-trip
test fails on all 32 of them until they agree with the renderer. `status_reason`
cost exactly that.
