---
schema: 2
id: TKT-01M2NSHN8JVH60ZJFNFC8N391D
title: Seed a conventions document from init, as terva wrote by hand
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - area/format
  - area/templates
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-16T19:01:55Z
updated_at: 2026-09-16T19:01:55Z
created_by:
  id: agent:claude/t3code
  name: ""
updated_by:
  id: agent:claude/t3code
  name: ""
extensions: {}
---

## Description

`init` should seed a conventions document, the way
TKT-01M2NQW8NZ1QY4BMHWYZF23GP8 asks it to seed a template.

### This is not a proposal, it is an observation

terva is the largest store using this tool, 123 open tickets, and it wrote
`.tickets/CONVENTIONS.md` by hand because nothing shipped one. It is not a
sketch. It carries a priority-means-triage table, the lifecycle transitions
worth memorising and which edges are missing, who may promote a draft, and the
label vocabulary with a rationale per dimension.

Its opening sentence is the argument for standardising it:

> Two other documents already cover ground, and this one does not repeat either.
> `README.md` beside this file explains the store's layout. `git ticket
> instructions` prints the tool's generic agent workflow. What follows is the
> part neither of them can know: the judgement calls terva made.

That is a third slot, and the tool ships the other two. `init` writes
`README.md` and `instructions --write` installs the workflow block, and the
per-store judgement has nowhere to live, so every store that needs one invents
both the file name and the shape.

### What the evidence says about the content

terva's most load-bearing section is the one it put first, and it is not about
labels at all: `low` on a draft means "weighed, worth keeping, not now" rather
than unimportant, because three triage pools collapsed onto `draft` when the
store was converted. Nothing mechanical could derive that, and an agent that
guessed would invert it.

So the seeded file must be mostly empty and mostly prompts. A template that
guessed at conventions would be worse than none, because a reader cannot tell a
shipped default from a decision somebody made.

### The relationship to the labels document

TKT-01M2NQVK39AD3TYKRPSD13430C proposes a generated `labels.md` with a region
the project writes. terva's experience cuts both ways on that and both halves
matter.

Against: terva says outright that what keeps its vocabulary usable is the
written section and not the store, so the generated inventory is the half that
was never the problem.

For: terva's own document lists its areas "by weight", by hand, which is exactly
the content that goes stale and exactly what a generator should own. Its
`CONVENTIONS.md` already mixes generated-shaped content with judgement, with a
person maintaining both. That is the two-region file, arrived at independently,
which is the strongest evidence the shape is right.

So the open question is whether these are one document or two: a `CONVENTIONS.md`
with a fenced generated label section, or a generated `labels.md` beside a
hand-written conventions file. Deciding that before either is built is cheaper
than merging them afterwards.

### Worth considering, not prescribed

- Whether it is opt-in like `instructions --write` or unconditional like the
  four directories. It is an opinion, which argues for opt-in.
- Whether `instructions` should point at it, since an agent reading the workflow
  block is the reader who most needs to know a store has house rules.
- What it is called. terva chose `CONVENTIONS.md`; shipping a different name now
  would strand the store that solved this first.
- Whether `doctor` should report a store that has none, which is the kind of
  thing the hygiene command is for and also the kind of opinion that annoys a
  store with nothing to say.

## Acceptance criteria

- [ ] init seeds a conventions document, mostly prompts rather than guessed defaults
- [ ] Whether it is one document with the labels section fenced inside it, or two files, is decided before either is built
- [ ] The name is settled against terva's CONVENTIONS.md rather than coined fresh
- [ ] Whether instructions points at it is decided, since that is where an agent would learn a store has house rules
- [ ] Whether it is opt-in or unconditional is decided
