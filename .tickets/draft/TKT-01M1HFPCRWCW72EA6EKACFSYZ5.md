---
schema: 1
id: TKT-01M1HFPCRWCW72EA6EKACFSYZ5
title: Decide how a store archives for the long term
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
dependencies: []
blocks_on: none
references:
  - ref: plan:deferred-questions
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-02T16:37:04Z
updated_at: 2026-09-03T17:00:32Z
created_by:
  id: agent:terva/dev-loop
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

The archive grows without bound. Every ticket ever closed stays in .tickets/archive/ as its own file, and nothing prunes, rolls up, or compresses it.

The sketch to start from: tar and compress anything in archive/ older than 30 days, grouped by the month it was archived in, and generate an index the tooling can read so something exploring the cold archive does not have to open every tarball.

What makes this harder than it looks is that the archive is not inert. Plan 6.3 says a dependency is satisfied by a ticket archived out of done and by nothing else, so check and deps read archived tickets to answer questions about live ones. check also scans archive/ for archive_location_mismatch and for a duplicate ID across both directories. Cold storage here is still queried.

Open, and this ticket is where they get settled:

- What the index has to carry. If it holds each archived ticket's ID and from_status, dependency resolution answers from the index and never opens a tarball. If it holds less, every check unpacks a month.
- Whether a tarball is compatible with the point of the format. The whole design is Markdown a person edits and git diffs. A .tar.gz is a binary blob: one late arrival rewrites the whole month, and review sees an opaque object.
- Determinism, if it does ship. tar records mtimes and entry order, so a naive rebuild produces a different blob from identical inputs and every regeneration is a diff.
- What 30 days counts from. archive.archived_at is the candidate, not updated_at.
- Who runs it. 12.4 says a store never upgrades itself, and the same argument says a mutation must not silently rewrite the archive as a side effect. That points at a person-run command.
- Whether the index alone is the answer. It is the separable half, it costs the format nothing, and it delivers most of the value for tooling. Compression is the half that trades the format's main property for bytes.

Nothing needs this yet. This repository's own archive is 7 files. It gets decided when a real store makes it hurt, and the size at which that happens is itself worth recording when somebody sees it.

## Notes

**agent:terva/mieli** at 2026-09-03T17:00:32Z

Groomed after `refs` shipped. The trigger has not fired: the archive is still 7 files and the store is 312K. But the cost analysis above is now demonstrably too small, and that is worth fixing before somebody sizes an index from it.

The description names check and deps as the commands that read archived tickets, and the first open bullet sizes the index for them: ID and from_status, so dependency resolution answers without opening a tarball. Three more commands read the archive by content, and I verified each against a real store rather than inferring it. A ticket archived out of done is still found by `refs jira:PROJ-1234`, by `files README.md`, and by `search`. `list` correctly does not find it.

Two of those three predate today. `files` and `search` have always spanned the archive and this ticket never accounted for them; `refs` makes it three and is what made me look. So this is a correction to the ticket, not only news about the new command. Plan section 8 now states the rule for all three, which is where the requirement should be read from.

That changes the first bullet rather than adding a fourth. An index carrying ID and from_status leaves `refs`, `files`, and `search` unpacking every month, or answering wrong, and answering wrong is worse: each one reports that nothing matches, which reads exactly like an item nobody ever tracked. To keep them working the index has to carry every reference, every path, and enough text to substring-match a title, description, notes, comments and summary, which section 8 lists as the search surface.

At that point the index holds most of what the tickets hold, and the sixth bullet gets sharper rather than weaker. If the index has to be nearly a copy of the archive to keep three commands honest, compression is buying bytes against a duplicate somebody still has to store, and the separable half stops looking like most of the value for a fraction of the cost. Whoever settles this should price the index against the archive before assuming a tarball wins.
