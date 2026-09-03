# git-ticket

A Git-native work ledger: Markdown tickets in `.tickets/`, one Go library, a
`git ticket` CLI. Read `docs/plan.md` before you change anything. It is the
design of record, not background reading.

## State

Phase 0 and Phase 1 are done and their exit criteria hold. The library parses,
renders, validates, queries, and mutates, with the store lock and the revision
precondition.

The store partitions by status into four directories, per plan section 4:
`draft/` for what somebody filed, `tickets/` for the working set of `ready`,
`in-progress`, `blocked` and `review`, `done/` for recent finished work, and
`archive/` for what has been swept out of it. One `statusDir` function in
`ticket/store.go` is the whole mapping, and `files()`, `Init`, `writeTicket`,
`check`, and `relTarget` all read it. An unknown status falls through to
`tickets/`. A misplaced file is `location_mismatch`, which `check --fix` repairs
by moving it, and that is also how a store migrates: no schema bump, because a
ticket file does not change when its path does.

`list` answers with open work, meaning every status except `done` and
`archived`, per section 8. Naming a status brings it back and `--all` drops the
exclusion. `search` deliberately does not take that default, because finding
what was already decided means reading a done ticket. In the library `Filter{}`
means open work and `Filter{All: true}` is everything.

Phase 2, the standalone CLI in section 12.1, has all 24 commands. `cmd/git-ticket`
and `cli` carry `init`, `create`, `update`, `show`, `list`, `search`,
`ready`, `status`, `claim`, `release`, `link`, `unlink`, `deps`, `files`, `ac`,
`dod`, `note`, `comment`, `summary`, `archive`, `unarchive`, `check`, `schema`,
and `instructions`, each in a human form and behind `--json`. All seven JSON
kinds of section 10 have a test, and every write honours `--if-revision`.

The agent workflow block lives at `cli/instructions.md`, embedded with
`go:embed`. Edit the Markdown, not a Go string.
`TestInstructionsNameRealCommands` holds every command and flag it names to what
the binary has, because prose telling a reader to run something that does not
exist is worse than no prose. `TestInstructionsWorkflowRuns` goes further and
runs the sequence against a real store, in the order the block prints it,
because every command can exist and the order still be wrong. It was: the block
said claim before ready, and a draft cannot be claimed.

Both Phase 2 exit criteria are met. The scripted end-to-end run is
`TestLifecycle` in `cli/lifecycle_test.go`. The other is `git ticket
check --strict` green in this repository's own CI, which run 4 did on `45b73c0`
in 29 seconds. Runs 1 through 3 were red, every one of them for the instance
convention in Gotchas below rather than for anything the workflow runs.

`.forgejo/workflows/ci.yml` runs gofmt, vet, `go test -race ./...`, and
`check --strict` over this repository's own ticket store.

"With no network" in that exit criterion describes `check` itself, per section
11: the command performs no network access. It is not a requirement that the CI
job run without network, which is why fetching `yaml.v3` from the proxy is fine.

Section 13 of the plan lists the phases and their exit criteria. Do not start a
later phase before an earlier one meets its criteria.

There are two remotes. `origin` is the internal Forgejo and `main` tracks it.
`github` is the public mirror at `github.com/terva-sh/git-ticket`, which settles
the module path: `go.mod` already declared it, so nothing changes. Plan 12.2
holds the rule. `main` and every tag are on both, at identical SHAs. Check one
without adding a remote, because a clone has `origin` alone:

```sh
TAG=$(git describe --tags --abbrev=0)
go list -m -json "github.com/terva-sh/git-ticket@$TAG"   # Origin.Hash
git rev-parse "$TAG^{commit}"                            # what it should equal
```

The two remotes move at different speeds, and that is deliberate. Push to
`origin` often: it is the working remote, its runners are internal, and CI there
is how a branch finds out it is wrong. The mirror is not a backup and does not
want your daily commits. Push it when there is something to release, which means
a tag, and let a release carry the commits behind it.

The reason is where the work runs. `origin` builds on the instance's own
runners, which cost nothing but their own capacity. The mirror is on public
infrastructure, so every push there spends somebody else's runners on work that
is not finished.

`main` moves by merging a PR, so the mirror push comes after the merge, from a
local `main` that has just been fast-forwarded:

```sh
git checkout main && git pull --ff-only && git push github main --follow-tags
```

A fresh clone has `origin` alone. Add the mirror when you need it, which is at
release time and not before:
`git remote add github git@github.com:terva-sh/git-ticket.git`.

The mirror is a plain `git push`, not a built tree: same commits, same tags, one
history. That works because the prose never names the internal hosts. Write "the
internal Forgejo" and "the internal registry" and take a real host from `origin`
at need. One exception ships knowingly, the container image in
`.forgejo/workflows/ci.yml`, because it is a working reference rather than
prose. `TKT-01M1FAFS` holds that decision. Before adding a hostname anywhere
else, check what you are about to publish:

```sh
git grep -n -E "$(git remote get-url origin | sed -E 's#.*@([^:/]+).*#\1#')" -- . ':!.forgejo'
```

## Branches and pull requests

Work lands on `main` through a pull request. Do not commit to `main` and do not
push to it, not even for a one-line documentation fix.

The reason is concurrency, not ceremony. Several agents work this repository at
once, in separate worktrees, and each one holds a `main` it believes is current.
A direct push makes every other agent's next push a rebase over commits they
cannot see, and the ticket store is the worst place for that: two agents each
adding a file under `.tickets/draft/` merge cleanly and each editing the same
ticket does not. A status change now also moves the file between directories, so
two agents moving one ticket to different statuses collide as a rename conflict
rather than as a content conflict on `status:`. A PR gives the collision one
place to happen and a reviewer to notice it.

The loop:

```sh
git switch -c fix/some-slug        # off an up-to-date main
just ci                            # the same steps the PR will run
git push -u origin HEAD
```

Then open the PR with `tea`, which is installed and logged in to the instance.
Name the ticket in the body, because a reviewer arriving from `git ticket show`
should find the review from the ticket and the ticket from the review:

```sh
tea pr create --base main --title "..." -d "$(cat body.md)"$'\n'
tea pr ls                          # open PRs
tea pr 1                           # one PR, body and all
tea pr merge 1 --style rebase      # server-side, once CI is green
```

That trailing `$'\n'` is not superstition. Bash's `$(...)` strips trailing
newlines, so the body arrives one byte short of the file. tea itself is exact:
a body posted with `curl` and the same body sent through `tea pr edit` compare
byte for byte on the server.

Write the body to a file and pass the file. A PR body is prose, and prose
assembled in a shell argument is prose nobody proofread.

A branch prefix says what the change is: `fix/`, `feat/`, `docs/`, `build/`,
`test/`, matching the commit type it will carry.

Merge server-side with `tea pr merge` or the web UI. Pushing a local merge
commit can leave the PR open with its commits already in `main`, which is a
state somebody then has to clean up by hand.

If you branched after committing to `main` by mistake, move the commits rather
than re-doing them, and use `git branch -f` rather than `reset --hard` so
nothing is destroyed if you got it wrong:

```sh
git switch -c fix/some-slug        # carries the commits
git branch -f main origin/main     # main is where the remote has it
```

## Commands

`just` drives the local loop. `just --list` names every recipe.

```sh
just ci        # lint, test-race, check: the same steps CI runs, in the same order
just test      # the whole suite, without the race detector
just install   # git-ticket into GOBIN, so `git ticket` resolves outside this tree
```

The recipes wrap plain `go` invocations and hold no logic of their own, so
`go test ./...` still works when you want one. `just ci` is the gate to run
before you push.

To read a CI result rather than guess at one, ask for the commit's statuses.
`tea api` carries the login, so there is no token to resolve and no host to
write down:

```sh
tea api "repos/terva-sh/git-ticket/commits/$(git rev-parse HEAD)/statuses"
tea api "repos/terva-sh/git-ticket/actions/tasks?limit=5"   # when there is no status yet
```

One context carries several rows at different timestamps, `pending` then
`success`, so read the newest and not the first. The row that gates a PR is the
one whose context ends in `(pull_request)`.

Without tea, the same call needs a token: `$TERVA_FORGE_TOKEN`, or the login
block matching the host in tea's `config.yml`, which is
`~/.config/tea/config.yml` on Linux and under `~/Library/Application Support`
on macOS. `terva/scripts/pr.sh` holds an awk parser for it. Take the forge host
from the `origin` remote rather than writing it down, so this file names no
host:

```sh
FORGE=$(git remote get-url origin | sed -E 's#.*@([^:/]+).*#\1#')
curl -sS -H "Authorization: token $TERVA_FORGE_TOKEN" \
  "https://$FORGE/api/v1/repos/terva-sh/git-ticket/commits/HEAD_SHA/statuses"
```

The suite includes the tests that hold the corpus to the plan. Run it after
touching anything under `testdata/` or section 11 of the plan.

`.forgejo/workflows/ci.yml` keeps its steps inline rather than calling `just`,
because the runner image is `golang:1.25-alpine` and installing just there would
buy nothing. That makes the justfile a mirror, so a change to one belongs in the
other.

`just build` and `just check` put the binary in the repository root, where the
README tells a reader to find it. It is gitignored for that reason.

Do not call `./git-ticket` directly. It is a build artifact with no sense of its
own age. Run `just ready` and `just check`, which depend on `build` and so
answer from the code in your tree.

This repository uses its own tool. Work that outlives a session belongs in a
ticket rather than in a comment: `git ticket ready` says what is startable, and
`git ticket create --title "..."` files what you found on the way. After
`just install` those work as Git subcommands; `just ready` runs the first one
from the tree without an install.

`ready` is half the backlog. Everything filed lands in `draft` and nothing
promotes it, so `git ticket list --status draft` is the other half and it is
usually the larger one. Read it before reporting that there is nothing to pick
up. Promoting a draft is the user's call: name what looks startable and let them
choose.

## Layout

`docs/plan.md` is the format, the store, the CLI, and every decision behind
them. Sections 4 through 11 are the format itself.

`testdata/` is the fixture corpus. Read `testdata/README.md` before adding to
it.

`cmd/git-ticket/` is a thin `main` and holds no decisions. Every choice about
flags, output, and exit status lives in `cli/`, where the tests can
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
Directories stay plural: `.tickets/`, `draft/`, `tickets/`, `done/`, `archive/`.

Fixtures are static files a person reviewed. Never generate one at test time.
Every fixture carries an `.expected.json` sidecar even when it is clean,
because a missing expectation cannot be told apart from a forgotten one.

Time-dependent behaviour uses the fixed reference instant
`2026-09-30T00:00:00Z`. Inject it. Never read the system clock in a test.

## Policy

This repository owns the format, the store, validation, and the CLI. Nothing
here imports terva. The terva integration is Phase 3 and is terva's to build.

**Do not write into a sibling repository.** Other agents work in those trees
at the same time you do, and a commit, a branch, or a PR you leave there
collides with work you cannot see. This applies to `../terva` and to every
other repository under `terva-sh/`, and it holds even when the change is
obviously correct and only touches documentation.

When the other side needs something, write a handoff document here and give it
to the user to carry over. `docs/handoff-terva-phase-3.md` is the worked
example. Verify the code in a handoff by compiling it against the published
module, because a wrong field name in a handoff costs the reader more time than
it saved. Writing that one caught two.

This rule arrived late. terva already holds `docs/plans/git-ticket.md` and PR
#880 from before it was set, which is exactly the collision it exists to
prevent. Both are terva's to keep, rewrite, or close.

The library reaches no network and runs no Git command that writes. No fetch,
push, merge, commit, or branch switch. Publishing is the user's ordinary Git
workflow, and a helper that does it for them is out of scope for v1.

`TestGitCommandsAreReadOnly` enforces that, against the command table in plan
7.4. Every git call goes through one helper per package, `runGit` in `ticket`
and `readGit` in `cli`, and running a command the table does not list fails the
suite. To add one, add the row and its reason to 7.4 first. Test code is exempt
and runs `git init` freely.

## Gotchas

CI actions resolve against this Forgejo instance, not GitHub. A bare
`uses: actions/checkout@v4` is fetched from the instance's own
`actions/checkout` path and 404s before a single step runs, which is how the
first version of `.forgejo/workflows/ci.yml` died. Use the public mirrors:
`Actions-Mirrors/forgejo-actions-checkout@v6`. Go comes from the internal
registry's `golang:1.25-alpine` rather than `actions/setup-go`, which fails the
same way for the same reason. `.forgejo/workflows/ci.yml` has the exact image
reference and is the only file that should, because it is the only place the
literal string has to work. That registry mirror is curated, so do not assume
an arbitrary `library/*` image exists. On alpine, `-race` needs cgo:
`apk add --no-cache gcc musl-dev`. The sibling repositories are the reference,
and `terva/.forgejo/workflows/ci.yml` is the fullest example.

A stale `./git-ticket` answers with a straight face. It is gitignored, so it
survives a branch switch, a rebase, and a pull, and nothing in its output says
which commit built it. A review of this repository's own store once opened with
`./git-ticket ready` against a binary six hours old whose JSON carried no
`readiness` field at all, so every ticket read `null` and a shipped feature
looked unbuilt. `just ready` and `just check` depend on `build`, and `just
build` prints the short SHA it built from.

`go:embed` silently skips any path whose name starts with `.` or `_`. That is
why store fixtures live under `store/` and not `.tickets/`. The realistic name
would embed nothing and leave a suite of tests passing against an empty corpus.

Renaming a code in section 11 fails `TestCorpusCoversEveryPlanCode` until the
sidecars follow. That is intended. The corpus and the spec are one artifact.

The standard library's `flag.Parse` stops at the first non-flag word, so a
naive parse never sees the `--json` in `git ticket show ID --json`. `parseFlags`
in `cli/cli.go` loops instead, consuming one positional per pass. Plan
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
the repository root. `findings()` in `cli/json.go` converts. A test
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

A `##` heading inside text passed to `--description` becomes a section rather
than part of the description. `parseBody` splits on any line starting with
`## `, so a long description written with Markdown subheadings comes back with
`body.description` holding only the prose above the first one and everything
after it in `body.extra`. Write `###` instead: the prefix test carries the
trailing space, so three hashes do not match. A `## ` inside a fenced code block
is already safe, because `parseBody` tracks fences.

Nothing catches that. `check` passes, `show` prints every section in order, and
the file reads correctly, which is why `TKT-01M1HVMQ` was filed that way and had
to be filed again. Repairing it after the fact is worse than it sounds: there is
no delete command, and `update --description` replaces the description alone, so
the stray sections survive and the content ends up duplicated. Catch it before
the commit, remove the file, and create the ticket again.
