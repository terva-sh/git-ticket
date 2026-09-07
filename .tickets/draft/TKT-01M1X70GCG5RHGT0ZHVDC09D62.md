---
schema: 2
id: TKT-01M1X70GCG5RHGT0ZHVDC09D62
title: Decide how a store predating the eol attribute gets repaired
type: spike
status: draft
status_reason: null
priority: low
due_on: null
labels:
  - question
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-07T05:56:12Z
updated_at: 2026-09-07T05:56:12Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Plan 7.5 now has `init` write two `.gitattributes` lines, one of which pins the store to LF. A store created before that line existed carries only the merge line, and nothing in the tool adds the other one.

`install-merge-driver` deliberately does not. That command repairs the tracked half of the merge driver, which is what its name promises, and quietly writing an unrelated rule beside it would make the name a lie. `TestInstallMergeDriverLeavesTheEOLLineAlone` holds it to that.

So the repair today is manual: add `.tickets/**/*.md text eol=lf` to `.gitattributes`, commit it, and re-checkout the ticket files. That works, and it requires knowing the problem exists, which is exactly what a person meeting it does not know. The symptom is `list` reporting nothing.

### The question

What, if anything, should offer the repair.

`check` is the repair mechanism this project already has, and `check --fix` is where a person looks when the store is wrong. Against that: every finding in section 11 is about the store's own contents, and `.gitattributes` sits at the repository root, outside the store. A finding about a file the store does not own would be the first of its kind, and section 11 would have to say what that means.

A second option is a flag on `install-merge-driver`, which keeps the default honest while making the repair one command. A third is that `init` is enough, and a store predating this is rare enough that the manual line in the documentation is the whole answer.

### The trigger

Somebody meets this on a real store. That means a store created before the eol line, cloned on Windows, reporting no tickets. Until then the population is small and known: this repository's own store predates the line and is covered by its repository-wide `* text=auto eol=lf`, and terva's store is the other one, which its own reply document already tells it about.

Do not build a `check` finding on the strength of the reasoning above. The cost is a new category in section 11, and it is worth paying only against a real report.
