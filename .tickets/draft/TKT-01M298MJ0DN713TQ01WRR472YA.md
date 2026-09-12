---
schema: 2
id: TKT-01M298MJ0DN713TQ01WRR472YA
title: Build git ticket create --file for an authored ticket document
type: task
status: draft
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
updated_at: 2026-09-12T01:01:57Z
created_by:
  id: agent:terva/mieli
  name: Mieli
updated_by:
  id: agent:terva/mieli
  name: ""
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

- [ ] A ticket written as a Markdown document is filed in one command, with its checklists and sections intact
- [ ] The leniency of plan 4.2 is preserved: unrecognised keys are ignored rather than refused
- [x] Settled and recorded: one flag taking a name or a path, or two flags for two intents
- [x] Settled and recorded: what a document carrying a status or a created instant does, given 4.2 and 6.2.1 disagree about who states those
- [ ] Explicit flags still win over the document, as they already win over a template
- [ ] The path is a library capability the CLI calls, not logic that lives in cli/

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
