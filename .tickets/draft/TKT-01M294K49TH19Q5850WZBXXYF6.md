---
schema: 2
id: TKT-01M294K49TH19Q5850WZBXXYF6
title: Decide whether git format-patch joins the 7.4 table
type: spike
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - policy
  - question
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references:
  - ref: plan:7.4
    path: docs/plan.md
  - ref: plan:12.8
    path: docs/plan.md
  - ref: handoff:flywheel-export-import
    path: null
claim: null
archive: null
created_at: 2026-09-11T21:04:49Z
updated_at: 2026-09-11T21:04:49Z
created_by:
  id: agent:terva/mieli
  name: Mieli
updated_by:
  id: agent:terva/mieli
  name: Mieli
extensions: {}
---

## Description

`git ticket export` reserves patch numbers 0 and 1 and leaves the rest, so code
travels by composing `git format-patch --start-number 2 -o DIR RANGE` from
outside. An `export --patch RANGE` flag would fold that in, and it needs a row in
the 7.4 table for `format-patch`.

Ruled against on 2026-09-11, with the reasoning below, so that a later reader
does not have to re-derive it.

### What was measured, not argued

Forgetting `--start-number 2` is harmless. It produces two files numbered
`0001`, and `git am *.patch` still applies both, exit 0, with only the commit
order inverted: the code lands before the tickets and nothing is lost. That was
the strongest practical argument for owning the call and it did not survive the
test.

`-o` beats `format.outputDirectory`, so a config cannot redirect output somewhere
the caller did not name. But the rest of `format.*` shapes the output even with
`-o` given: with `format.subjectPrefix` and `format.signature` set, the same
command produced `Subject: [PATCH flywheel 1/1] fix: the thing` and a trailing
signature. Owning the call therefore means owning a row of `-c format.x=`
overrides to neutralize it, and `TestGitCommandsAreReadOnly` checks the command
name rather than the flags, so none of that neutralization is protected by the
thing the table buys.

### Why no

The table earns its keep by being short enough to read in one breath, and by the
sentence under it: every row but `config` only reads. That sentence already
carries one exception, narrowed to a single command where the user typed the
request. A second makes it a list, and a list of exceptions is a rule nobody
checks. Against that, the gain is small.

The asymmetry decides it. Adding a row later is a minor under 12.4 and costs
nothing. Taking one back is a break. There is no hurry to spend a promise that
can be spent later on better evidence.

### What was done instead

The discoverability gap is the real cost of composing, and it is closed without
a table row: `export --help` and the block in `cli/instructions.md` name the
composition, and export prints the exact `format-patch` line with the directory
filled in beside the `git am` line it already printed.
</description>
<parameter name="acceptance_criteria">["A real report exists of somebody shipping an export that should have carried code and did not, or one broken by the numbering", "The format.* neutralization is settled: which keys are overridden, and how a test holds the override list", "12.4 is re-read, since admitting the row is a minor and removing it later is a break"]
