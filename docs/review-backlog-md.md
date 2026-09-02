# Backlog.md, reviewed against this format

Section 16 of `docs/plan.md` already names Backlog.md as the source of the
acceptance criteria and definition of done fields. This is the full read of it:
what else is worth taking, what belongs to a UI rather than to us, and what we
should keep refusing.

Read on 2026-09-02 against `MrLesk/Backlog.md@main`: `README.md`,
`CLI-INSTRUCTIONS.md`, `ADVANCED-CONFIG.md`, `src/types/index.ts`,
`src/guidelines/agent-guidelines.md`, `src/core/cross-branch-tasks.ts`,
`src/core/statistics.ts`, and the `src/mcp/` tree.

The premise of this review is that terva draws the board and we do not. So a
feature only counts as ours if a UI cannot supply it from what `--json` already
returns. Everything else is terva's, and our job for those is to make sure the
data is there.

## The one-line verdict

Backlog.md is ahead of us on the authoring surface and behind us on everything
concurrent. It can write every section of a task from the CLI and we cannot
write two of ours. It has no claim, no revision precondition, and no store lock,
which is the half of the problem this format exists for.

Three gaps are real defects rather than missing features, and they are in
section A below. The rest is a menu.

## A. Defects: fields the format defines and nothing can write

These are not comparisons with Backlog.md. They are places where our own plan
promises a field and our own code cannot reach it. Backlog.md is what made them
visible, because its workflow leans on the first one hardest.

**The implementation plan has no mutation.** Plan 5.2 lists
`## Implementation plan` as a known section. `ticket/parse.go` reads it,
`ticket/render.go` writes it, `ticket/query.go` searches it, `show` prints it,
and `cli/json.go` carries it as `body.implementationPlan`. There is no
`SetImplementationPlan` and no CLI flag. The only way to fill the section is to
hand-edit the file, which is what the whole ledger exists to stop.

This matters more than its size. Backlog.md's argument for the format is three
review checkpoints, and the middle one is that the agent writes its plan into
the ticket and waits before writing code. Our `cli/instructions.md` cannot tell
an agent to do that, because there is no command to name.

**`update` has no `--description`.** `SetDescription` exists at
`ticket/mutation.go:472` and nothing calls it. `create --description` sets the
field through `CreateOptions` instead. So a description is write-once from the
CLI: a typo in the first sentence of a ticket is unfixable without hand-editing.

**`create` has no `--milestone`.** `update` has one. Every other field `create`
sets has an `update` flag, which plan 15 records as the argument that settled
`--type` and `--parent`. The reverse asymmetry never got checked.

## B. Baseline: what a UI cannot supply

### B1. Readiness per ticket, and the reason

`git ticket ready` is a query that returns a filtered list. Backlog.md puts the
same verdict on every task as a field, `isReady`, and adds a `readiness` object
to task detail with `isBlocked`, `blockingDependencies`, and
`missingDependencies`.

The difference shows up the moment something draws a list. A board that greys
out a blocked card and says what it waits on has to call `ready`, diff two ID
sets, then call `deps` per card. The data is derived either way, so this costs
the format nothing: it is a computed field on the `ticket-list` and `ticket`
kinds, alongside `revision` and `path`, which plan 7.1 already computes rather
than stores.

Their fail-closed rule is worth copying verbatim. A dependency that resolves to
nothing, or to more than one record, blocks rather than counting as satisfied.
Our `check` reports `dependency_missing`, so we agree about the finding, but
`ready` should agree about the verdict.

### B2. Comments are prose to us and records to them

Their comment is `{ index, body, createdAt, author }`, appended through
`--comment "..." --comment-author @sara`, delimited in the file by standalone
`---` lines. Our `## Comments` is one string, and `comment ID TEXT` stamps an
entry into it.

Plan 10.1 already draws this distinction and lands on the other side for
checklists: `body` is the document and `checklists` is a one-way view of it. A
`comments` view derived the same way would cost the format nothing and let a UI
render a thread. Without it, terva has to re-parse our stamp format out of
prose, which is a second parser for a shape we already wrote.

An `--author` on `comment` is a separate question. We stamp with the actor,
which covers attribution for agents. A human quoting a colleague is the case it
would serve, and it is thin.

### B3. Seeding a ticket takes one call to them and N to us

`backlog task create "Feature" -d "..." --ac "One" --ac "Two" --dod "Run tests"
--plan "..." --notes "..." --dep task-1` is one command. Ours is `create`
followed by an `ac --add` per criterion.

For a person this is friction. For an agent it is worse: every extra call is a
round trip and another chance to abandon the ticket half-built. `create` should
take `--ac`, `--dod`, and `--milestone`, and `--plan` once A1 lands.

Repeatable index flags on the checklist commands are the same argument from the
other end. They accept `--check-ac 1 --check-ac 3 --uncheck-ac 2 --remove-ac 4`
in one edit and document that removals process high to low so the indexes stay
stable. Ours takes one operation per invocation. Note that they also have a
remove, which our `ac` and `dod` lack entirely: `--add`, `--check`, `--uncheck`,
and no way to delete a criterion that turned out to be wrong.

### B4. Milestones are a string to us and files to them

We store `milestone` as a bare scalar with no registry. Nothing validates it, so
`v1.2` and `v1.2.0` are two milestones and `check` cannot tell a typo from a new
one. There is no rename, so correcting one means editing every ticket that names
it.

Backlog.md keeps a milestone file with an id, title, description, and due date,
and gives it `add`, `rename`, `remove`, `archive`, and `list`. `rename` cascades
into every task by default and `--no-update-tasks` opts out. `remove` makes the
caller say what happens to the tasks: `clear`, `keep`, or `reassign --reassign-to`.

That cascade is the part a UI cannot supply. Everything else about milestones is
grouping and progress bars, which terva computes from `list --json` today.

The cheap half is a `milestones` allowlist in `config.yml` next to `labels`,
which plan 4.1 already establishes as advisory vocabulary that `check` warns
about and never errors on. That catches the typo. The expensive half is the file
per milestone with its own dates and lifecycle, and it should wait for a store
that wants dates on a milestone.

### B5. Due dates

They carry `dueDate` on tasks and milestones, UTC at minute precision, rejecting
date-only values, with `--clear-due-date`. We have nothing.

This is a frontmatter addition, so plan 5.3 makes it cost every fixture that
carries frontmatter. `status_reason` cost exactly that and the note in AGENTS.md
records it. The question is whether a repository-native ledger wants deadlines
at all, or whether a date belongs on a milestone and never on a ticket. I lean
toward the milestone, which folds this into B4 and buys the field once instead
of on every ticket.

### B6. Manual ordering

`ordinal` on a task, with `src/core/reorder.ts` maintaining it, is what makes
drag-and-drop persist. `Store.List` sorts by ID and nothing else, which plan 5.5
makes chronological because a ULID sorts by creation time. `priority` is a
filter and never an order. So the only sequence we can express is the order
tickets were filed in, and there is no way to say that one ready ticket should
be picked up before another.

Two different wants hide in that sentence and they deserve different answers. A
sort by priority is display: every consumer has `priority` on every row and can
order on it, and if we ever want it in human output it is a `--sort` flag with
no format change behind it. A hand-set sequence is not display, because no UI
can persist a field the format does not have.

So `ordinal` is the deferred half, and it earns its place only when something
reorders. The trigger: terva ships a board with reorderable columns, or somebody
asks for an ordering within one priority level.

### B7. Repair for the findings `check` reports

Plan section 11 defines fourteen error codes and five warnings, and `check`
repairs none of them. Backlog.md's `doctor` diagnoses and repairs, and
`src/core/duplicate-task-repair.ts` is 27 KB of it.

Most of that file is a tax on their ID choice, which is section E. The useful
question is narrower: which of our findings have exactly one correct repair, so
that a tool can apply it without guessing at intent? Two do.

`filename_id_mismatch` is a rename to `<id>.md`, and plan 4 fixes the target
name with no room for a second reading. `archive_location_mismatch` is a file
move, because plan 6.3 already rules that the status wins.

The rest look repairable and are not. `duplicate_id` has to pick which file
keeps the ID, which is a judgement about which ticket is the real one.
`dependency_missing` has a repair in `unlink --depends-on`, which AGENTS.md
notes resolves against the ticket's own list precisely so it works on a
dangling ID, but dropping the edge and creating the missing ticket are both
plausible and only a person knows which. `label_unknown` is either a typo in
the ticket or a gap in the allowlist.

So a `check --fix` is worth having and its scope is two codes, not fourteen. It
is also the first command that would write without being told which ticket to
write, so it needs `--dry-run` and it needs to name every path it touched.

## C. Display: terva's job

None of these need a format change. Listed with what they read, so we can check
that the data is already there.

| Feature | What it reads from us |
|---|---|
| Terminal Kanban board and web board | `list --json`, grouped on `status` |
| Drag and drop between columns | `status`, plus `ordinal` from B6 if the order must persist |
| Task detail pane, editing forms | `show --json`, then the mutation commands |
| Acceptance criteria checkbox editor | `checklists` from 10.1, then `ac --check N` |
| Project statistics and health | `list --json`: counts by `status` and `priority`, completion rate, `updatedAt` for staleness, `createdAt` for age |
| Dependency tree drawing | `deps --transitive --dependents` |
| Milestone progress bars | `list --milestone M`, counting done against total |
| Search result highlighting | `search --json`, positions computed by the UI |
| Date display formats | `createdAt` and `updatedAt`, always RFC 3339 from us |
| Board export to Markdown | `list --json` |
| Colour, icons, column widths, hiding empty columns | nothing |

Their `backlog overview` is the interesting entry. `src/core/statistics.ts`
computes status counts, priority counts, completion percentage, average task
age, the five stalest tickets, the five blocked ones, and recent activity. Every
one of those falls out of `list --json` because we return whole tickets rather
than a compact row. So we owe terva nothing here, and that is worth stating
before somebody proposes a `git ticket stats`.

Two entries in that table are the same argument as B1 seen from the UI side.
Blocked tickets in their statistics are computed by scanning every task's
dependencies against every other task, which is what any consumer of ours has to
do today.

## D. Operations worth lifting

**A refreshable instruction block.** `init --instructions` writes to `AGENTS.md`
and refuses when the file exists, naming `git ticket instructions` so the user
can paste it themselves. Backlog.md's `agents --update-instructions` rewrites
the block in place across `CLAUDE.md`, `AGENTS.md`, `GEMINI.md`, and
`.github/copilot-instructions.md`, preserving everything around it. A
marker-delimited block we can rewrite is the difference between shipping an
instructions change and asking every user to re-paste one. We should keep
writing only `AGENTS.md`, because the rest is their vendor list and not ours.

**Named workflow guides.** They split instructions into `overview`,
`task-creation`, `task-execution`, and `task-finalization`, and the short block
in `AGENTS.md` says only to run `backlog instructions overview`. Our block is one
piece and it is already long enough that `TestInstructionsWorkflowRuns` has to
check its ordering. Splitting keeps the always-loaded part small.

**Shell completion.** `completion install` for bash, zsh, fish, and PowerShell,
completing command names, live ticket IDs, and enum values. Our enums are fixed,
so most of it is a static script, and the dynamic half is IDs. Small, and it
pairs well with `shortestUnique`.

**Bulk archive by age.** `backlog cleanup` moves old done tasks out of the
board's way. We have `archive ID`, one at a time. An `archive --status done
--before DATE` with `--dry-run` is the same operation, and it wants the same
care `check --fix` does about naming what it touched.

**Cross-branch visibility.** This is their most distinctive Git idea and the one
closest to our actual problem. `src/core/cross-branch-tasks.ts` lists branches
touched in the last 30 days, reads the ticket directory out of each with
`git ls-tree`, compares last-modified times, and shows the newest state. So a
board reflects work in flight in another worktree.

AGENTS.md describes exactly that situation: several agents in separate
worktrees, each holding a `main` it believes is current. Everything the read
needs is read-only, so plan 7.4 permits it, though `git ls-tree` and
`git log -1 --format` are not in the table and would have to be added with their
reasons. The catch is that their implementation resolves conflicts by file
mtime, which is a guess, and `taskResolutionStrategy: most_recent |
most_progressed` is a configuration knob for which guess to make. We should not
copy the guess. We should copy the question, which is a deferred entry in
section 15.

## E. Declined, and why

**Freeform statuses.** `TaskStatus = string` in their types, with the column set
in `config.statuses`. Our plan parks this as `TKT-01M1F7Z2Y5H1ZJAHRGF3XE6F91`,
and reading their code is evidence for the cost side rather than against.
`src/core/statistics.ts` hardcodes `task.status === "Done"` to count completions,
and `src/core/cross-branch-tasks.ts` resolves lifecycle state from which
directory a file sits in rather than from its status, because the status string
means nothing to the tool. Configurable statuses do not remove the need for the
tool to know which state means finished. They just move that knowledge somewhere
it cannot be checked.

**`autoCommit`.** They commit ticket changes for you, and `bypassGitHooks` exists
because those commits fight repositories with pre-commit hooks. That second flag
is the argument. Our policy leaves publishing to the user's ordinary Git
workflow, and plan 15 records that a sync helper is answered no by two
independent mechanisms.

**`onStatusChange`.** A shell command in `config.yml`, run on every status
change, with `$TASK_ID` and friends substituted, overridable per ticket in
frontmatter. Their own example launches `claude` in the background. This makes
a ticket file in a repository a way to run arbitrary commands on a machine that
checks it out, which is a far bigger authority grant than anything else in the
format. terva can hook status changes host-side where the permission model
already lives.

**Fuzzy search.** Theirs finds "authentication" from "auth" and returns match
positions for highlighting. Ours is case-insensitive substring with `--regex`.
Fuzzy is better for a person browsing and worse for an agent, which wants to
know whether a string is present. Predictable beats forgiving here, and `--regex`
covers the cases substring misses.

**A separate drafts directory with `promote` and `demote`.** They move files
between `drafts/` and `tasks/`. We have `draft` as a status with transitions in
6.2, so the same state change costs no file move and no path in `git log`. Their
`demote` is our `status ID draft`.

**Sequential integer IDs.** `TASK-1`, with `zeroPaddedIds` and a configurable
prefix. Plan 5.5 already argues for ULIDs, and their repository is the proof:
`duplicate-task-repair.ts` is 27 KB, `doctor` exists to run it, and their
dependency graph documentation has to define what `ambiguous task ID` means
because two branches can both mint `task-42`. Two of our disconnected agents
cannot collide.

**Editing several tickets in one call.** `backlog task edit 42 43 44 -s "In
Progress"`. It reads well until `--if-revision`, which is one precondition
against many tickets. Either the flag becomes a list positioned by argument
order, or a batch edit is quietly the unchecked kind of write. Declined for now,
and the reason is worth keeping if somebody asks again.

## F. Where we are already ahead

Recorded so nobody borrows backwards.

Backlog.md has no claim, no lease, no expiry, and no conflict when two agents
pick up the same task. It has no revision precondition, so two writes to one
task are last-writer-wins with no way to find out. It has no store lock. Its
concurrency answer is `autoCommit` and cross-branch reads, which is a way to see
a collision after it happened rather than to prevent one. Sections 6.4 and 7 are
the reason this format exists, and nothing here should be traded for a feature.

Our filenames are the ID alone. Theirs are `task-42 - Add GraphQL resolver.md`,
so renaming a task renames the file and `git log` on the old path stops. Plan 4
already records this decision and it holds up.

Our references are one typed list with a namespace. They have three untyped
string arrays: `references`, `documentation`, and `modifiedFiles`. Their split is
semantic and ours is general, and `git ticket files PATH` answers the one query
the split was for. Their convention is still worth adopting inside our
namespaces, so `file:` means a path this ticket touched and `doc:` or `proposal:`
means context to read first.

Our JSON contract has eight kinds to their three, plus `unreadable` on every
list so a consumer can tell a short answer from a complete one, plus stable error
codes, plus the exit status rules in 10.2. Their `--json` covers `task list`,
`task view`, and `search`, and their mutations answer in text.

Our `check` has a finding code table in plan section 11, a fixture corpus, and
`corpus_test.go` holding the two to each other. Their validation is inline and
undocumented outside the code.

Our rendering is deterministic with a round-trip test over every fixture. Theirs
uses `<!-- AC:BEGIN -->` and `<!-- DOD:END -->` markers in the body to find the
checklists, which is a parser that fails when a person removes a comment they had
no reason to keep.

## G. What to do next

Section A first, because those are defects and the fix for each is small.

1. Add `SetImplementationPlan` and a CLI path to it. Decide append against
   replace on the same argument 9 uses for `summary`, which is that a plan is one
   statement and `Notes` is the log.
2. Add `update --description`, which wires up a mutation that already exists.
3. Add `create --milestone`, `--ac`, and `--dod`.
4. Add a remove to `ac` and `dod`, and let all four operations repeat in one
   invocation.

Then B1, readiness as a computed field, because it is additive under 12.4, it
needs no format change, and it is the piece terva's board most obviously wants.

Then B2, a derived `comments` view, on the same argument.

B4 splits: the `milestones` allowlist in `config.yml` is cheap and catches the
typo; the milestone file and the rename cascade wait.

B5, B6, and the cross-branch question in D go to section 15 with their triggers
written down. Each of those is a frontmatter field or a new Git read, and plan 15
exists so that a question with no trigger yet does not get answered by whoever
happens to touch the code next.
