---
schema: 2
id: TKT-01M2HEEX3SDTTP1Q8EH7526YK6
title: check passes a ticket whose criteria are prose the tool cannot see
type: bug
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - format
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-15T02:31:12Z
updated_at: 2026-09-15T02:31:12Z
created_by:
  id: agent:claude-code/triage
  name: ""
updated_by:
  id: agent:claude-code/triage
  name: ""
extensions: {}
---

## Description

`git ticket create` accepts a Description containing a `###` heading that names
a real section, and nothing afterwards notices. The items under it render, read
and review as if they were the section they name, but no mutation can reach
them and `check --strict` calls the store clean.

Reproduced from a clean store:

```sh
git ticket init
printf 'Body.\n\n### Acceptance criteria\n\n- [ ] first\n- [ ] second\n' > d.md
git ticket create --title Reproduction --description-file d.md

git ticket check --strict            # No problems found.       exit 0
git ticket check --fix --dry-run --strict  # No problems found. exit 0
git ticket ac <id> --check 1
# invalid_field: there is no item 1; the section has 0 (field Acceptance criteria)
```

`git ticket show` prints the heading and both checkboxes verbatim, so the only
thing distinguishing a real section from an impostor on screen is one `#`. A
reviewer reading the ticket sees acceptance criteria. The tool sees prose.

Not specific to acceptance criteria: `### Definition of done` behaves the same
way, and by inspection so will any `###` naming a known section.

### Where this was hit

Filing `TKT-01M2BXKF8F` in the `terva-sh/ketju` store. The criteria were
written as a `###` inside the Description at creation and the mistake surfaced
only when an agent tried to tick one, several sessions later, at the point of
closing the ticket. In between, the ticket passed every `check --fix --dry-run
--strict` run in that repository's CI, and was read by a human who had no
reason to doubt it.

The affordance was never missing — `create --ac` is right there and repeatable.
What is missing is anything that says the mistake was made.

### Why a heading-level rule would be wrong

`###` sub-headings inside a Description are ordinary and wanted. This
repository's own store carries 14 `### Trigger`, 8 `### Open questions` and
7 `### Scope`; ketju's carries a long tail of one-off ones. Flagging `###` as
such would be unusable noise.

The signal is the NAME COLLISION, not the level: a sub-heading whose text
matches a section the format already owns is almost certainly a section that
was meant to be real. Neither store has a single such collision today, so a
check keyed that way starts silent in both.

### Worth considering, not prescribed

A `check` diagnostic is the smallest fix and the one that catches tickets
already filed. Refusing the heading at `create` time would also work but leaves
existing stores unexamined, and is a harder sell for `--file`, where the body
is authored elsewhere. A `--fix` that promotes the heading and moves its items
into the real section looks attractive and is probably a trap: the items would
have to move between sections, which is a rewrite rather than a repair, and
`--fix` elsewhere in this tool only ever moves or rewrites whole files.

## Acceptance criteria

- [ ] check reports a ticket whose body carries a `###` heading naming a section the format owns, and --strict fails on it
- [ ] The report names the section and the offending heading, so the fix is obvious without reading the source
- [ ] A `###` heading that does not collide with a known section name is left alone; `### Scope` and `### Trigger` stay silent
- [ ] Both this store and the ketju store still pass unchanged, so the check ships without a backlog of repairs
