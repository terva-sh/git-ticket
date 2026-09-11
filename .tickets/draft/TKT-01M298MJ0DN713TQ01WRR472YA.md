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
claim: null
archive: null
created_at: 2026-09-11T22:15:31Z
updated_at: 2026-09-11T22:15:31Z
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

### Why it waits on the refactor

The loader is already in `ticket/template.go`, so the library half is nearly
there. What the refactor settles is the shape a host uses to file a ticket it
composed, which is the same question `PlanImport` and `ApplyImport` answer for
the foreign case. Building this first would pick an answer the refactor then has
to reconcile with.

### Open, and worth settling before building

Whether `--file` and `--template` stay two flags or become one that takes either
a name or a path. Two flags are honest about two intents and cost a reader a
choice; one flag is fewer words and hides which resolution happened. Either way
it is a CLI contract under 12.4, so it is settled before it ships.

What happens to lifecycle-shaped keys in the document is the other one. 6.2.1
says a backport states status and created instant explicitly, and 4.2 says the
template loader seeds neither. A document that carries `status: done` should
probably be refused rather than quietly filed as a draft, but that is a decision
and not an assumption.
</description>
<parameter name="acceptance_criteria">["A ticket written as a Markdown document is filed in one command, with its checklists and sections intact", "The leniency of plan 4.2 is preserved: unrecognised keys are ignored rather than refused", "Settled and recorded: one flag taking a name or a path, or two flags for two intents", "Settled and recorded: what a document carrying a status or a created instant does, given 4.2 and 6.2.1 disagree about who states those", "Explicit flags still win over the document, as they already win over a template", "The path is a library capability the CLI calls, not logic that lives in cli/"]
