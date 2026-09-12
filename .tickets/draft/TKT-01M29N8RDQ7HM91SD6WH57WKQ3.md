---
schema: 2
id: TKT-01M29N8RDQ7HM91SD6WH57WKQ3
title: Decide what a CRLF checkout does to an imported export
type: spike
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - question
  - integration
  - format
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references:
  - ref: plan:12.8
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-12T01:56:15Z
updated_at: 2026-09-12T01:56:15Z
created_by:
  id: agent:terva/mieli
  name: Mieli
updated_by:
  id: agent:terva/mieli
  name: Mieli
extensions: {}
---

## Description

Plan 12.8 promises a receiver can land an export with `git am DIR/*.patch` in a repository that has never heard of git-ticket. On Windows that promise has an unwritten caveat, and the caveat makes the arriving files unreadable.

git for Windows installs with `core.autocrlf=true`. Applying an export's patch under that config writes the ticket files with CRLF line endings. Plan 5.3 requires LF, and `parse` refuses a file whose fence line is `---\r` with:

    parse_error: file does not start with a --- frontmatter fence

So the receiver lands the patch cleanly, `git am` reports success, and every ticket it just added is unparseable.

### How it was found

Not by a user. `TestAHunkWithNoTrailingNewlineAppliesWithRealGit` landed in the no-trailing-newline fix, and the Windows lane went red on it in run 34666101851: `git apply produced "one\r\ntwo" (8 bytes), want "one\ntwo" (7 bytes)`. Both rows failed, the trailing-newline control included, which is what identified line-ending conversion rather than the marker.

That test has since pinned `core.autocrlf=false`, because it is about the marker and not about line endings. The pin is correct for that test and settles nothing here.

### Why this is not v0.14.3 again

v0.14.3 made `init` write `.tickets/**/*.md text eol=lf` beside the merge-driver line, which protects a store that git-ticket created. An export does not carry that line. `exportTicketFiles` writes a hunk for each ticket file and nothing else, so a repository receiving its first ticket by `git am` has no `.gitattributes` covering `.tickets/`, and nothing tells git to leave those bytes alone.

The overlap with TKT-01M1X70GCG5RHGT0ZHVDC09D62 (how a store predating the eol attribute gets repaired) is real but the trigger differs: that one is about a store that already exists, this one is about files arriving into a repository that has no store yet.

### The decision this needs

Whether export should carry a `.gitattributes` hunk, and if so what it may claim about a file it does not own. Adding one means an export writes a file outside `.tickets/`, which is a larger promise than "the ticket files themselves" and could collide with what the receiver already has.

Whether import should instead repair on the way in. `import --adopt` re-renders through the library rather than applying a patch, so it may already be immune, and that should be measured rather than assumed before anything is built.

Whether the honest answer for v1 is documentation: name the caveat in 12.8 and tell a Windows receiver to set `core.autocrlf=false` or `--no-autocrlf` for the operation.

### Trigger

A person on Windows landing an export and finding unparseable tickets, or a decision to state 12.8's promise without a platform caveat.</description>
<parameter name="acceptance_criteria">["Whether import --adopt is affected is measured rather than assumed, and the result recorded", "The chosen repair is recorded with its reasoning, or the caveat is written into plan 12.8", "A test reaches the behaviour from any platform, the way TestAStoreSurvivesACRLFClone does, rather than needing a Windows runner"]
