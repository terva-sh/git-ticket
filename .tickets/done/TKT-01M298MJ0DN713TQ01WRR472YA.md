---
schema: 2
id: TKT-01M298MJ0DN713TQ01WRR472YA
title: Build git ticket create --file for an authored ticket document
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels: []
assignees: []
milestone: null
parent: null
origin: null
dependencies:
  - TKT-01M298KEGY8KV17MCFN9V1A4WG
blocks_on: none
references:
  - ref: plan:4.2
    path: docs/plan.md
  - ref: plan:6.2.1
    path: docs/plan.md
  - ref: plan:4.3
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-11T22:15:31Z
updated_at: 2026-09-12T01:17:24Z
created_by:
  id: agent:terva/mieli
  name: Mieli
updated_by:
  id: agent:terva/mieli
  name: Mieli
extensions: {}
---

## Description

There is no way to hand the tool a whole ticket. Prose can be piped into
`--description-file`, but an agent that has written a complete ticket as
Markdown, which is the most natural thing for a model to produce, has to
decompose it back into flags and repeated `--ac` arguments.

`git am` is the only whole-document path today, and it demands the document be a
patch.

### Most of this already exists

`create --template NAME` reads a Markdown file and seeds nine fields off it:
type, priority, labels, assignees, milestone, description, acceptance criteria,
definition of done, and implementation plan. Plan 4.2 makes that loader lenient
on purpose, seeding what it recognises and ignoring every other key, which is
what lets a template be made by copying a real ticket.

The only thing that makes it a template rather than a document is that the name
resolves inside `.tickets/templates/`. A path instead of a name is most of the
feature.

### What it is not

Not `import`. That takes a foreign ticket which already has an ID and a
provenance, reminting it and reconciling its vocabulary. This takes a document
nobody has filed, with no ID and no history. They meet at the loader and part
immediately, and collapsing them would give one command two meanings.

Nor should the loader become a validator. 4.2's leniency is the property that
makes a template cheap to author, and the same leniency is what lets an agent
hand over a document it wrote without matching a schema first.

### Why it waited on the refactor

The loader is already in `ticket/template.go`, so the library half is nearly
there. What the refactor settled is the shape a host uses to file a ticket it
composed, which is the same question `PlanImport` and `ApplyImport` answer for
the foreign case. Building this first would have picked an answer the refactor
then had to reconcile with.

That refactor is TKT-01M298KEG and it shipped in v0.16.0, so the dependency is
discharged.

### Settled, and recorded in plan 4.3

Both questions this ticket held open are answered. The answers live in plan 4.3
(Authored documents) rather than here, because the plan is authoritative on a
CLI contract, and a note records the reasoning.

Two flags rather than one. `--template NAME` resolves inside
`.tickets/templates/` and `--file PATH` reads a path, because one flag taking
either would have to guess which was meant and the two are not reliably
distinguishable. `--file` is a third seed source and is refused together with
`--template` and `--from`.

Lifecycle keys are ignored and named. A document carrying `status`,
`created_at`, `updated_at`, `id`, or a `claim` or `archive` block has none of
them honoured, and each one is reported on stderr. `--status` stays the only
route to done or archived, per 6.2.1.

The document supplies the title, which makes `--title` optional with `--file`.
That one followed as a consequence rather than being asked, and the note says
so, because it is the part to challenge first if any of this is wrong.

## Acceptance criteria

- [x] A ticket written as a Markdown document is filed in one command, with its checklists and sections intact
- [x] The leniency of plan 4.2 is preserved: unrecognised keys are ignored rather than refused
- [x] Settled and recorded: one flag taking a name or a path, or two flags for two intents
- [x] Settled and recorded: what a document carrying a status or a created instant does, given 4.2 and 6.2.1 disagree about who states those
- [x] Explicit flags still win over the document, as they already win over a template
- [x] The path is a library capability the CLI calls, not logic that lives in cli/

## Notes

**agent:terva/mieli** at 2026-09-12T01:01:40Z

Both open questions are settled by the user, and recorded in plan 4.3
(Authored documents) rather than here, because the plan is authoritative on a
CLI contract. This ticket is startable.

### Two flags, not one

`--template NAME` stays and `--file PATH` is new.

The argument that decided it is mechanical rather than aesthetic. One flag
taking either would have to guess which was meant, by a slash, an `.md`
suffix, or a probe of the filesystem, and a name and a path are not reliably
distinguishable: a path need contain no slash, and a template may share its
name with a file that exists. No flag in this CLI resolves a name or a path
today, so a heuristic would be a new habit as well as a guess.

`--file` also reads correctly against what the CLI already says. `plan`,
`note`, `comment` and `summary` each take a bare `--file` meaning "this
command's primary payload, from a path", and for `create` that payload is a
whole ticket. The `--something-file` suffix is reserved for one field's prose,
which is why this is not `--template-file`.

As a third seed source it is refused together with `--template` and `--from`,
as `usage`, by the rule those two already follow.

### Lifecycle keys are ignored and named

A document carrying `status`, `created_at`, `updated_at`, `id`, or a `claim`
or `archive` block has none of them honoured, and each one that appears is
named on stderr.

The gate holds under every option considered, so this was only ever about how
loud to be: `--status` is the only route to done or archived per 6.2.1, and
everything else lands in draft. Ignoring keeps 4.2's leniency, which is what
lets a document be made by copying a real ticket, and that is the obvious way
to write one. Naming is what import already does per 12.8, where what the
receiver never agreed to does not travel and is reported rather than dropped
in silence.

The template and the document differ here, and the difference is the whole
reason to warn in one and not the other. A template is a form copied once and
reused, so a stray `status` in it is noise. A document is written for one
create, so the same key is more likely to have been meant. The warning names
`--status` and `--created`, so a real backport is told which flags carry the
intent the file could not.

Three seeding mechanisms already refuse to carry lifecycle state, which is
what made this the conservative answer rather than a new rule: `--template`
seeds nothing lifecycle-shaped, `--from` seeds title, description and criteria
text but no status and deliberately drops the ticks, and `import --adopt` sets
no status at all, so every adopted ticket lands as draft whatever the sender
had.

### A third answer, settled by consequence rather than by instruction

The title. 12.1 states `create --title T` as required, and a document carries
its own title, so the two had to be reconciled before anybody could build
this. Plan 4.3 now says the document supplies it and `--title` is optional
with `--file`, an explicit `--title` still wins, and a document with no title
refuses.

That follows from this ticket's first criterion, which asks for a document
filed in one command, and from `--from`, which already seeds a title. It is
the one part of 4.3 nobody was asked about, so it is the part to challenge
first if any of it is wrong. Changing it costs a plan edit and nothing else,
since no code exists yet.

**agent:terva/mieli** at 2026-09-12T01:07:38Z

draft to ready: Promoted by the user. Its dependency TKT-01M298KEG shipped in v0.16.0, and both open questions are settled and recorded in plan 4.3, so nothing is left to decide before building.

**agent:terva/mieli** at 2026-09-12T01:17:10Z

Built on `feat/create-file`. All six criteria hold.

`ticket/document.go` is the capability: `ReadDocument(path)` returns a
`Document` carrying `Seed` (everything a template gives), `Title`, and
`Ignored`. `CreateOptions.Document` carries it and `Store.Create` applies it
through the same per-field block a template uses, so the precedence is shared
code rather than a second rule that could drift. `cli` registers `--file`,
refuses the seed-source pairs as usage, and words `warnDocumentKeys`. Nothing
else about a document lives there.

### A null states nothing, so it is not reported

`lifecycleKeysIn` reports a key only when it carries a value. A ticket rendered
by this tool carries `status_reason: null` and `claim: null` on the ordinary
path per 5.3, and copying a real ticket is the documented way to author a
document, so reporting those would fire the warning on the common case and
teach a reader to skip it. That is a decision the plan did not make and it is
worth knowing about; the test is
`TestReadDocumentSaysNothingAboutANullKey`.

### What the smoke test found, which the suite did not ask

Running the real binary is what surfaced it: a `- [x]` in the document survives
into the filed ticket. `--from` unticks deliberately, with the reasoning that
carrying a tick claims evidence for work this ticket has not begun, and
`import --adopt` unticks too. So three seeding paths do not agree.

The behaviour follows from 4.3 sharing 4.2's loader rather than from any ruling,
which is exactly the shape of thing this project says to record instead of
invent. Plan 15 now carries it with its trigger, and the case that stops it
being obvious is the backport: 6.2.1 exists so finished work arrives finished,
and a document filed with `--status done` plausibly wants its satisfied criteria
ticked. `TestADocumentKeepsItsTicks` pins the current behaviour and says in its
own comment that a failure means the decision landed rather than a regression.

### Proven

Twelve new tests, and the real binary rather than the seam alone. `git ticket
create --file authored.md` with no `--title` filed
`TKT-01M29JX6KHE4F5E5G16KEEE2HV` from a document that carried `id`, `status:
done`, `created_at`, two nulls and an unknown `severity`. It landed as draft
with the document's title and every section intact, and the warning named the
three stated keys, skipped both nulls and the unknown key, said where it landed,
and pointed at `--status` and `--created`.

`--file` reached `completion --dump` with no work, since that vocabulary is
derived from the flag set.

**agent:terva/mieli** at 2026-09-12T01:17:21Z

Task worklog for this ticket, from the session task board.

### Tasks

- [x] task-17 A ticket written as a Markdown document is filed in one command, with its checklists and sections intact — Proven at three levels. Library: TestReadDocumentSeedsTheWholeTicket checks title, type, priority, labels, assignees, milestone and all four sections survive; TestCreateFilesAWholeDocument files one through Store.Create and finds 2 acceptance criteria, 1 definition-of-done item and the implementation plan intact, landing as draft. CLI: TestCreateFileFilesAWholeTicket runs `create --file PATH` with no --title at all and reads every section back through `show --body`. Real binary: `./git-ticket cr…
- [x] task-18 The leniency of plan 4.2 is preserved: unrecognised keys are ignored rather than refused — documentFrontmatter declares only the six seedable keys, so an unknown key unmarshals nowhere and is ignored rather than refused, which is 4.2's mechanism unchanged. TestReadDocumentIgnoresWhatItDoesNotKnow feeds severity, sprint and reviewers and requires no error plus an empty Ignored list, since an unknown key is not a lifecycle key. TestCreateFileKeepsTheLeniency drives the same through the CLI at exit 0 and asserts neither key is reported in the warning. The real binary run carried `severit…
- [x] task-19 Explicit flags still win over the document, as they already win over a template — Create applies the document through the same per-field `if o.X == ""` block a template uses, so the precedence is shared code rather than a second rule. TestAnExplicitFieldBeatsTheDocument passes Title, Type and Priority against a document carrying all three and requires the flags to win while the unnamed description still comes from the document. TestCreateFileLetsAnExplicitFlagWin proves it through the CLI: --title and --priority win, the document's type still applies because the caller named…
- [x] task-20 The path is a library capability the CLI calls, not logic that lives in cli/ — ticket/document.go holds the capability: ReadDocument(path) parses the file, seeds the Template fields plus the title, and decides which lifecycle keys it will not honour. CreateOptions.Document carries it and Store.Create applies the precedence, sharing the template seeding block rather than duplicating it. cli/ does three things only: registers --file, refuses the seed-source pairs as usage, and words warnDocumentKeys. No parsing, no precedence and no lifecycle decision lives in cli/, which mi…

## Summary

Done on `feat/create-file`, all six criteria. `git ticket create --file PATH`
files a ticket somebody wrote out in full, per plan 4.3.

`ticket/document.go` holds the capability. `ReadDocument(path)` returns a
`Document` with `Seed`, the fields a template also gives, plus `Title` and
`Ignored`. `CreateOptions.Document` carries it and `Store.Create` applies it
through the same per-field block a template uses, so one precedence rule serves
both and cannot drift. The CLI registers the flag, refuses the seed-source
pairs, and words the warning, and that is all it does.

Two things a later reader should not have to rediscover.

A lifecycle key is reported only when it states a value. A ticket this tool
rendered carries `status_reason: null` and `claim: null` per 5.3, and copying a
real ticket is the documented way to author a document, so warning on those
would fire on the common case and train a reader to skip the warning that
matters.

A `- [x]` in a document survives into the filed ticket, while `--from` and
`import --adopt` both untick. That follows from 4.3 sharing 4.2's loader rather
than from a ruling, so plan 15 carries the question with its trigger instead of
this ticket inventing an answer. The backport of 6.2.1 is what stops it being
obvious: a document filed with `--status done` plausibly wants its satisfied
criteria ticked. `TestADocumentKeepsItsTicks` pins the behaviour and says a
failure means the decision landed.

Twelve tests, and the real binary rather than the seam alone. That run is what
found the tick question: a document carrying `id`, `status: done`, `created_at`,
two nulls and an unknown `severity` filed as a draft with its title and every
section intact, and the warning named the three stated keys, skipped the nulls
and the unknown one, said where it landed, and pointed at `--status` and
`--created`.
