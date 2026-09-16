---
schema: 2
id: TKT-01M2NQW8NZ1QY4BMHWYZF23GP8
title: init should create templates/ with a worked example
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - area/templates
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references:
  - ref: ticket:TKT-01M240H2M7EZ2QSJ7A7T3RB36H
    path: null
claim: null
archive: null
created_at: 2026-09-16T18:32:45Z
updated_at: 2026-09-16T19:00:20Z
created_by:
  id: agent:claude/t3code
  name: ""
updated_by:
  id: agent:claude/t3code
  name: ""
extensions: {}
---

## Description

`init` should create `templates/`, put one worked example in it, and say in the
store README how to use a template, write another, and refer to one.

### Templates already exist, and that is most of the point

This is not a request to build templates. Plan 4.2 defines them,
`ticket/template.go` loads them from `.tickets/templates/NAME.md`,
`Store.Templates()` and `Store.Template(name)` are library API, `create
--template` seeds from one, `config` and `config --json` publish the sorted
names, and the TUI create picker offers them.

Every part of that is built and none of it is discoverable. `storeDirs` is
`draft`, `tickets`, `done`, `archive`, so `init` never creates `templates/`.
The store README written by `init` describes the four directories, `epics.md`
and `config.yml`, and does not mention templates once. A person who has run
`init` has a store whose template support is invisible unless they read the plan
or the CLI reference.

### The empty directory will not survive on its own

`storeDirs`' own comment says why this cannot be fixed by creating a directory:
"Git tracks no empty directory, so a store whose `tickets/` happens to be empty
loses it on the next clone". An empty `templates/` would be gone the first time
anybody cloned the repository, and the feature would be invisible again.

That is an argument for the example template rather than a complication of it.
The example is the file that makes the directory exist in Git, so the two halves
of this ask are one mechanism: ship the example and the directory keeps itself.

### Not the same ticket as TKT-01M240H2M7EZ2QSJ7A7T3RB36H

That one is about turning an existing ticket into a template and discovering
which template fits new work, and it carries open interface questions about
template identity, usage metadata and a discovery command. It is authoring.

This is bootstrap: what a store has on the day it is created. They touch the
same directory and nothing else, and this one should not wait on those
decisions. If the identity question there changes how a template is referred to,
the example shipped here is one file to update.

### The example is the opinion, so it is the hard part

A template nobody would use teaches worse than no template, because it says this
is what we think a ticket looks like. Whatever ships has to be a ticket shape
this project would actually file, with the acceptance criteria unchecked, and it
has to survive being the first thing a new user reads.

Worth deciding at the same time: whether `init --instructions` and the bare
`init` differ here. The instructions block is already opt-in, and an example
template is closer to that kind of opinion than to the four directories that are
the format.

## Acceptance criteria

- [ ] init creates templates/ and writes one example template into it
- [ ] The example survives a clone, since the directory alone would not
- [ ] The store README init writes says how to use a template, write another, and refer to one
- [ ] The example is a ticket shape this project would file, with its checklists unchecked
- [ ] Whether the example is opt-in like the instructions block, or unconditional like the four directories, is decided
- [ ] An existing store that predates this gets a recorded answer: repaired, left alone, or told how to catch up
