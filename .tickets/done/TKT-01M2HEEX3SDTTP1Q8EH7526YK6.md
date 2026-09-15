---
schema: 2
id: TKT-01M2HEEX3SDTTP1Q8EH7526YK6
title: check passes a ticket whose criteria are prose the tool cannot see
type: bug
status: done
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
updated_at: 2026-09-15T03:04:41Z
created_by:
  id: agent:claude-code/triage
  name: ""
updated_by:
  id: agent:terva/invisible-sections
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

- [x] check reports a ticket whose body carries a `###` heading naming a section the format owns, and --strict fails on it
- [x] The report names the section and the offending heading, so the fix is obvious without reading the source
- [x] A `###` heading that does not collide with a known section name is left alone; `### Scope` and `### Trigger` stay silent
- [x] Both this store and the ketju store still pass unchanged, so the check ships without a backlog of repairs
- [x] The ## warning no longer advises ### for a heading naming a section the format owns; it says the real section was opened and the items are live

## Notes

**agent:terva/invisible-sections** at 2026-09-15T02:59:33Z

Triaged. Every claim here was checked against the tree rather than read, and all
of them hold: the reproduction runs exactly as written from a clean store, check
--strict and check --fix --dry-run --strict both report no problems at exit 0
while ac --check 1 fails with invalid_field, and the JSON envelope confirms the
cause by reporting acceptanceCriteria as an empty list. The heading counts are
exact, 14 "### Trigger", 8 "### Open questions" and 7 "### Scope", and neither
this store nor ketju's carries a single name collision, so a check keyed on the
name starts silent in both as promised.

One thing this ticket understates, and it changes the fix. The tool does not
merely fail to notice the mistake, it instructs the reader to make it. A
description carrying "## Acceptance criteria" produces a real, working section
whose items tick: that was measured, not inferred. The CLI then warns about that
heading and says, verbatim, to write "### Acceptance criteria" instead, which is
the invisible form. So for the seven names the format owns, Description,
Implementation plan, Acceptance criteria, Definition of done, Notes, Comments
and Summary, the existing advice in cli/headings.go is inverted: it takes a
ticket that works and tells the author to break it. The warning is right for
"## Scope" or "## Trigger" and wrong for exactly the names that matter. That is
why a fifth criterion was added rather than the four being reworded; the four
describe the missing diagnostic and are correct, and none of them covers the
wrong instruction that produces the defect in the first place.

A hazard for whoever builds it. The rule has to be an exact name match, never a
prefix or a substring. originRecord in ticket/import.go writes "### Summary at
the origin", "### Notes at the origin" and "### Comments at the origin" into the
work record of every ticket adopted through import --same-owner, which shipped
today, so a substring rule would fire on all three on every adopted ticket and
the check would arrive with a backlog it created itself. Exact match was
verified safe against all three.

Settled with the user before the branch: the code is section_heading_demoted,
chosen because "demoted" is unspent in the tree at zero hits while "shadow" is
already carrying two distinct senses, a binary shadowed on PATH and completion
shadowing, and it describes the mechanism rather than the symptom. The warning
for an owned name will say the heading opened the real section and the items are
live, rather than advising "###".

The reporter's own judgement is left standing in full. The argument against a
heading-level rule holds, and the argument that a --fix would be a trap is
right for the reason given: moving items between sections is a rewrite, and
--fix elsewhere in this tool only ever moves or rewrites whole files.

## Summary

Shipped as section_heading_demoted, a warning in plan section 11, plus the
correction to the advice that produced the defect.

The check is keyed on the name and never on the level, which is the reporter's
own argument and it held up under measurement: this store carries 14
"### Trigger", 8 "### Open questions" and 7 "### Scope", so a level rule would
arrive with a backlog and teach its readers to ignore it. The match is exact and
case-insensitive rather than a prefix, because originRecord in ticket/import.go
writes "### Summary at the origin" and its notes and comments forms into the work
record of every ticket adopted through import --same-owner, which shipped the
same day; a substring rule would have fired on three headings of this tool's own
making on every adopted ticket.

It is a warning rather than an error because the store is valid: nothing dangles,
nothing fails to parse, and the file is exactly what its author wrote. What is
wrong is that the author meant something else. There is no --fix, for the reason
the reporter gave and which survived review: the repair moves items from one
section into another, which is a rewrite of the body, and every other --fix here
moves or rewrites whole files.

The fifth criterion is the one the triage added, and it turned out to be the
cause rather than a second symptom. A description carrying "## Acceptance
criteria" produces a real working section whose items tick, measured rather than
inferred, and the existing warning answered that by telling the author to write
"### Acceptance criteria" instead. For the seven names the format owns the advice
was inverted: it took a ticket that worked and instructed the author to break it.
The same inverted sentence was in AGENTS.md and in cli/instructions.md, both of
which outlive any session, and all three are corrected.

Verification. The reported reproduction was run against the built binary and now
warns at exit 0 and fails --strict at exit 1, naming both the offending heading
and the real section. Both real stores pass unchanged: this one and ketju's, the
store where the mistake was originally made, each report no problems, which is
the criterion that lets the code ship without a repair queue. The corpus fixture
discriminates three ways in one ticket, a colliding heading, an ordinary
"### Trigger", and a heading inside a fenced block, and both falsifications were
run: making IsSectionName always true fires on the ordinary subheading, and
dropping the fence scanner fires on the fenced one. Each produces two findings
where the sidecar records one.

Left out deliberately. No tag: this is a minor under 12.4 by the new-surface
rule, two new exported names and a new check code, with nothing broken and no
schema move, but cutting the release is a separate decision. The eight-hundred-odd
existing tickets in other stores were not surveyed beyond ketju's, because the
check is silent on both stores available here and a survey of stores this machine
does not hold would be a claim rather than a measurement.
