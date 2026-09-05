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
and `instructions`, each in a human form and behind `--json`. Six more landed
after Phase 2, taking the binary to 30: `plan`, `refs`, `remove`, `config`,
`install-merge-driver`, and `merge-driver`. Phase 4's view work added `ui` and
`self-update`, taking it to 32. Every JSON kind of section 10 has a test,
ten with `self-update` per 10.7, and every write honours `--if-revision`.

The TUI is Phase 4's view: `tui/` is the rendering stack, `tui/view` the
application, and `git ticket ui` the entrypoint, wired through `Env.RunUI` so
an embedding host that imports only `ticket` and `cli` never builds the
terminal stack. Much of `tui/` is lifted from terva, and the attribution has a
regime: a derived file opens with a header pointing at NOTICE, NOTICE carries
terva's full MIT text and the file list, and `TestNoticeAgreesWithTheHeaders`
holds the two to each other in both directions. A new lift joins both or the
suite fails. Every key the list view accepts must appear in the footer or on
the `?` help page, and `TestEveryListKeyIsHintedSomewhere` enforces it: a
control the UI never prints might as well not exist.

Releases run through v0.8.1, and `self-update` (plan 12.6, with the graded
exit bucket of 10.2) is proven end to end: on 2026-09-04 the v0.8.0 release
binary applied v0.8.1 over the live GitHub API, check exit 10, apply exit 0,
and the result was byte-identical (`cmp`) to the released binary.

`schema` and `config` split the vocabulary and the split is load-bearing.
`schema` is what the binary enforces, identical in every store, and plan 10.4
requires it to read no store so it answers before `init` and outside a
repository. `config` is what one store chose, per 10.6: the label and milestone
allowlists, the actors, the `create` defaults, and the lock timeout. Do not move
a per-store value into `schema`, because that costs `schema` the one property
that makes it useful early.

An allowlist publishes as `{"values": [...], "enforced": bool}` and never as a
bare list. Per plan 4.1 an empty allowlist permits everything, so a consumer
handed `[]` reads it backwards and fails silently. `enforced` is derived from
the length, so the two cannot drift. Both allowlists are advisory: an unlisted
label still writes and `check` warns, which is where `check --strict` bites.

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
`check --fix --dry-run --strict` over this repository's own ticket store. That
last one is the verification command of plan section 11: it plans every repair,
writes nothing, and exits 1 when one is pending. `--strict` is load-bearing,
because `epics_index_stale` is a warning and without it a stale index exits 0.
CI never commits a repair. Run `just fix` and commit what it changed.

"With no network" in that exit criterion describes `check` itself, per section
11: the command performs no network access. It is not a requirement that the CI
job run without network, which is why fetching `yaml.v3` from the proxy is fine.

Section 13 of the plan lists the phases and their exit criteria. Do not start a
later phase before an earlier one meets its criteria.

There are two remotes. `origin` is the internal Forgejo and `main` tracks it.
`github` is the public mirror at `github.com/terva-sh/git-ticket`, which settles
the module path: `go.mod` already declared it, so nothing changes. Plan 12.2
holds the rule. `main` and every tag are on both, at identical SHAs. Check one
without adding a remote, because a clone has `origin` alone. Ask the proxy for
the `.info` file rather than running `go list -m`, for the reason in Gotchas:

```sh
TAG=$(git describe --tags --abbrev=0)
curl -sS "https://proxy.golang.org/github.com/terva-sh/git-ticket/@v/$TAG.info"
git rev-parse "$TAG^{commit}"   # Origin.Hash should equal this
```

The two remotes move at different speeds, and that is deliberate. Push to
`origin` often: it is the working remote, its runners are internal, and CI there
is how a branch finds out it is wrong. The mirror is not a backup and does not
want your daily commits. Push it when there is something to release, which means
a tag, and let a release carry the commits behind it.

The reason is what a push publishes, not what it costs to run. A push to the
mirror puts unfinished history in front of anyone reading the public repository,
and `main` there is what a visitor takes for the project. Runners are not the
argument: the mirror's only workflow is `.github/workflows/release.yml`, which
triggers on `push` of a `v*` tag and nothing else, so a branch push starts no
run at all. Pushing three docs commits took the mirror's run count from 2 to 2.
A tag is what spends public capacity, and a tag is the thing worth spending it
on.

`main` moves by merging a PR, so the mirror push comes after the merge, from a
local `main` that has just been fast-forwarded:

```sh
git checkout main && git pull --ff-only && git push github main --follow-tags
```

Pushing a `v*` tag publishes binaries, on both forges.
`.forgejo/workflows/release.yml` and `.github/workflows/release.yml` both fire
on it, and one `.goreleaser.yaml` is the config behind both. So the
`--follow-tags` above is what starts the public release, and
`git push origin v0.5.1` starts the internal one.

`v0.5.1` is the tag that proved this end to end, and it is worth knowing what
"proved" meant, because a green job is not the same claim. Both forges published
six assets, five archives and a `checksums.txt`. The archives were downloaded,
`sha256sum -c` passed, and the unpacked binary reported
`git-ticket v0.5.1 (dac0ff2f8b8d, ...)` on each. Sizes differ by a few kilobytes
between the forges because alpine builds with a different Go patch release than
`ubuntu-latest`. Same commit, so that is expected, and nothing here promises a
byte-identical archive across forges.

Each builds the same five archives and a `checksums.txt`, then checks that the
binary reports the tag. That check is not ceremony. Plan 12.1 reads the version
from what `go build` recorded rather than from an ldflag, so a shallow checkout
produces a binary answering `devel` and nothing else fails. On Forgejo the check
gates the upload, because that workflow does its own publishing. On GitHub
goreleaser has already published by then, so a failure turns the job red and
leaves a release to delete by hand.

The Forgejo job needs a `BOT_TOKEN` repository secret holding a token with write
access. The GitHub job uses the `GITHUB_TOKEN` that Actions provides. A tag
pushed without `BOT_TOKEN` fails loudly rather than publishing nothing quietly.

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

To read a CI result rather than guess at one, ask for the commit's combined
status. `tea api` carries the login, so there is no token to resolve and no host
to write down:

```sh
tea api "repos/terva-sh/git-ticket/commits/$(git rev-parse HEAD)/status"
```

That path is singular, and the difference is the whole rule. The plural
`/statuses` returns every row ever posted, several per context, grouped by
context and ordered newest first within each group. So any dedupe that keys on
context and keeps the row it saw last keeps the oldest row of each group. On
`6066b70` that array reads `success, pending, pending, success, pending,
pending` across two contexts, and a dict built from it in one pass reports
`pending` for a commit that had gone green four minutes earlier. PR #80 was
nearly merged on exactly that read.

The singular path collapses the history server-side. It answers with one row per
context, each already the newest, plus a top-level `state` and a `total_count`.
That count is how many contexts have reported, and on an open PR it is 1 and
stays 1. `ci.yml` triggers on `pull_request` and on `push` to `main` only, so a
branch gets the `(pull_request)` row and nothing else. The `(push)` row appears
when the commit lands on `main`, which is at merge, minutes later and by then
you are not waiting on it. Do not poll for a second row on a branch. The row
that gates a PR is the one whose context ends in `(pull_request)`, so read that
row rather than `state`, which answers for every context at once:

```sh
tea api "repos/terva-sh/git-ticket/commits/$(git rev-parse HEAD)/status" |
  python3 -c "import json,sys;[print(s['updated_at'],s['status'],s['context']) for s in json.load(sys.stdin)['statuses']]"
```

If you have a reason to read the plural path, sort by `updated_at` yourself and
take the last row. Never trust array order:

```sh
tea api "repos/terva-sh/git-ticket/commits/$(git rev-parse HEAD)/statuses" |
  python3 -c "import json,sys;[print(r['updated_at'],r['status'],r['context']) for r in sorted(json.load(sys.stdin),key=lambda r:r['updated_at'])]"
```

`actions/tasks` is the fallback when no status exists yet, and it has three
edges worth knowing before you lean on it. `?limit=N` is ignored, so a request
for 3 rows came back with 185. `conclusion` is `null` on every row, including
the failures, which makes `status` the field carrying the verdict there. And
nothing scopes the answer to your commit, so match `head_sha` yourself.

Without tea, the same call needs a token: `$TERVA_FORGE_TOKEN`, or the login
block matching the host in tea's `config.yml`, which is
`~/.config/tea/config.yml` on Linux and under `~/Library/Application Support`
on macOS. `terva/scripts/pr.sh` holds an awk parser for it. Take the forge host
from the `origin` remote rather than writing it down, so this file names no
host:

```sh
FORGE=$(git remote get-url origin | sed -E 's#.*@([^:/]+).*#\1#')
curl -sS -H "Authorization: token $TERVA_FORGE_TOKEN" \
  "https://$FORGE/api/v1/repos/terva-sh/git-ticket/commits/HEAD_SHA/status"
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

A unit of work gets its ticket filed, worked, and closed on the same branch,
so the PR carries the ticket file through its whole lifecycle and the store
never holds a claim for a branch that merged. PRs #102, #104, and #105 are the
pattern. Before labeling a new ticket, read the allowlist in
`.tickets/config.yml`: an unlisted label passes the write and then fails CI at
`check --strict`, and there is no `tui` label, which is why the TUI tickets
ship unlabeled. `update` has no `--label` flag; labels change through
`--add-label` and `--remove-label`.

`ready` is half the backlog. Everything filed lands in `draft` and nothing
promotes it, so `git ticket list --status draft` is the other half and it is
usually the larger one. Read it before reporting that there is nothing to pick
up. Promoting a draft is the user's call: name what looks startable and let them
choose.

Name a ticket with its title the first time you mention it, in a summary, a
commit message, a PR body, or a ticket note:

```text
TKT-01M1PQ7T (Build git ticket remove, per plan 9.1)
```

After that the bare ID is enough within the same piece of writing. A ULID is
unreadable on purpose, so a summary naming three bare IDs asks the reader to
look up three tickets before it says anything, and the reader is usually the
person who did not do the work. `cli/instructions.md` carries the same rule for
every project, and plan 12.1 records why.

Keep a title under 72 characters. `check` warns past that and refuses a write
past 120, per plan 5.1. Both numbers come from `git ticket schema` rather than
from memory.

An acceptance criterion is evidence of what was asked for, so do not reword one
to match what shipped. The single exception is a criterion that was
unsatisfiable as written, which is not the same as one that turned out to be
hard. A hard one stays as it is and goes unticked. Closing a ticket that way is
fine: no check reports an unticked criterion, so an honest `[ ]` costs nothing
and a false `[x]` costs the next reader their trust in every other box.

`TKT-01M1PCY3` (schema does not publish the label and milestone allowlists) is
the worked example. Its third criterion asked that a store with no allowlist be
distinguishable from one with an empty allowlist, and `permitted()` in
`ticket/config.go` treats nil and empty identically, so those are one state and
nothing could have satisfied it. It shipped unticked in PR #76 and was reworded
in #77, once the impossibility was demonstrated rather than asserted.

Rewording one is three edits, not one. Put the original wording verbatim in a
note, because a reader of the ticket should not have to run `git log` to learn
what was asked. Add a note naming the earlier note it supersedes, since `note`
appends and the old one still argues the opposite. Then reread the summary:
`summary` replaces rather than appends, so a summary still describing the old
state is how a ticket ends up disagreeing with its own checkbox.

Pass `--actor agent:terva/<name>` on every command that writes. This store
declares no `defaults.actor` in `.tickets/config.yml`, deliberately, so a write
without the flag warns on stderr and is recorded as `human:sothr`, the first
entry in `actors`. Do not silence the warning by declaring the field. It is the
opt-out for a store one person works alone, and this is not one: several agents
write here at once, and a claim signed with the wrong name tells the others a
human is holding the ticket.

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

A claim about mechanism goes into a permanent file only after you ran the thing
that would falsify it. `docs/plan.md`, this file, and a tag message all outlive
the session that wrote them, so a wrong explanation costs a later reader more
than no explanation would. PR #88 shipped a guess about `Origin.URL` that read
like a finding, and #89 withdrew it. Write what you observed, or run the test
that settles it.

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

The CLI now warns on stderr when you do it, naming the heading it found and
`###` as the fix. It covers every command that takes prose and every spelling
that carries it, so `create --description` and `create --description-file` both
warn, and so do `--plan`, `--plan-file`, `update --description`, and the `--file`
on `plan`, `summary`, `note`, and `comment`. The warning names the flag the text
arrived through. It warns rather than refuses, because passing several sections
in one string works and is sometimes meant, so the write still happens and the
exit status does not move. Read your stderr.

Writing the prose to a file and passing `--description-file` is the better habit
anyway, and not only for the quoting. A file is where you notice you typed `## `.

Nothing downstream catches it, which is why the warning is at the point of
writing. `check` passes, `show` prints every section in order, and the file reads
correctly, which is why `TKT-01M1HVMQ` was filed that way and had to be filed
again. Repairing it after the fact is worse than it sounds: `update
--description` replaces the description alone, so the stray sections survive and
the content ends up duplicated.

So catch it before the commit, remove the ticket, and create it again. `git
ticket remove ID` is the repair, per plan 9.1.

`rm` on the file does the same job for a ticket nobody has referenced, and that
is the point: deleting an unreferenced ticket leaves `check --strict` green, so
the deletion was never the hard part. What `remove` adds is the refusal. It
stops with `ticket_referenced` when another ticket names this one in
`dependencies` or `parent`, naming the tickets that do, because `rm` there
leaves a `dependency_missing` or `parent_missing` that `check --fix` will not
repair. It stops with `ticket_touched` when the ticket carries notes, comments,
a summary, a claim, or an archive record, because that is work somebody did and
`archive` is the operation for it.

`--force` overrides both and names every dangling reference it created on
stderr, with the command that repairs it. The repair differs by field: `unlink
--depends-on` drops a dependency and does nothing to a parent, which `update
CHILD --parent ""` clears.

A release tag has to agree with the module path, or the binary is quietly wrong
about itself. `go build` derives the version from the tag reachable at HEAD, and
a tag whose major version has no matching path suffix is not a version of this
module at all, so Go falls back to a pseudo-version. Tagging `v9.9.9` on
`github.com/terva-sh/git-ticket` produced
`v0.5.1-0.20260904151039-e40405b4aac2` rather than the tag. Nothing failed: the
build succeeded, the archives packed, and only the binary disagreed with the
release it shipped in. Both release workflows check `--version` against the tag
because of this. Going to `v2` means the module path gains `/v2` first, per 12.4.

Goreleaser's `dist/` is gitignored for the same reason, not for tidiness. Go
counts an untracked file as a modified tree, and goreleaser creates `dist/`
before it builds, so without that line every published binary would report
`modified: true`. A tagged run in a scratch clone reports `modified: false`,
which is the check worth repeating if the ignore rule ever moves.

GitHub does not fire a workflow for a tag pushed in the same operation that
first adds the workflow file. `git push github main --follow-tags` carried
`.github/workflows/release.yml` and `v0.5.1` together, and GitHub registered the
workflow from the `main` update and dispatched nothing for the tag. Deleting the
tag on the mirror and pushing it alone fixed it, and the identical tag object
came back. This is a first-release-only trap, so it will not recur while the
file stays put, and it is written down because the evidence points the wrong
way: every check says the setup is correct.

What makes it findable is that the run count is zero rather than one. A job-level
`if:` that evaluates false still creates a run and marks it skipped, so a guard
like `github.server_url == 'https://github.com'` cannot produce zero. Check the
count before suspecting the guard. `gh` is installed and logged in, and it is to
the mirror what `tea` is to the instance:

```sh
gh api repos/terva-sh/git-ticket/actions/runs --jq '.total_count'
gh api repos/terva-sh/git-ticket/actions/permissions   # enabled, allowed_actions
gh api repos/terva-sh/git-ticket/actions/workflows --jq '.workflows[].state'
```

A release is not proven by a green job. Read the assets back, verify
`sha256sum -c`, and run the unpacked binary, because the failure this catches is
a build that succeeds while shipping a binary that disagrees with its own tag.

`go list -m` answers from the local module cache before it asks the proxy, so it
can report a hash that was never published. Verifying v0.6.0 returned
`Origin.Hash = e40405b`, a commit on neither remote and in no clone, which reads
exactly like a botched tag. The published tag was right the whole time. Ask for
`@v/$TAG.info` over HTTP instead, which has no local cache to answer from. Clear
a poisoned entry by removing
`$(go env GOMODCACHE)/cache/download/github.com/terva-sh/git-ticket/@v/$TAG.*`,
which is read-only and needs `chmod u+w` on the directory and the files first.

The proving run above is where that entry came from. It tagged a scratch clone
`v0.6.0` at `e40405b` to watch goreleaser stamp a real version, and a
`v0.6.0.info` naming that commit appeared in the shared module cache. The clone
was thrown away and the commit reached no remote, but the cache entry outlived
both and was still sitting there when the real v0.6.0 was tagged seven hours
later. Prove a release with a number you will never ship. The same run also used
`v9.9.9` and that left no entry behind, because nothing ever asks the cache for
it again.

Do not try to tell a poisoned entry from a good one by reading it. The obvious
tell is wrong: `Origin.URL` is absent from 68 of the 72 entries cached here, and
both a proxy fetch and a `GOPROXY=direct` fetch write it, so its absence
separates nothing. Which invocation writes a `URL`-less entry was never pinned
down. Compare the hash against `git rev-parse` and do not read the tea leaves.

A release number comes from plan 12.4, not from the size of the diff. Under
`v0.x` the operative precedent has two steps. A break moves the minor, which
is why v0.6.0 is a minor: `title_too_long` refuses a write that used to work,
not because 19 commits landed. A new surface also moves the minor, by the
user's ruling on v0.7.0 and repeated on v0.8.0: a new command or package is a
larger promise than a patch carries, even with nothing broken. Flags beside
old ones and fixes stay patches, which is v0.5.1 and v0.7.1. Read what an
earlier release decided with `git tag -l v0.7.0 -n99` before picking one.

An interface contract that 12.4 will cover, an exit status, a flag spelling, a
JSON kind, is settled with the user before it ships, because re-shipping a
covered surface costs a break. The `self-update` exit bucket came out of
exactly that question, and the plan records the decision with its precedent.
