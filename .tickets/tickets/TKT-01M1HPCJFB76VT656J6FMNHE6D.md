---
schema: 1
id: TKT-01M1HPCJFB76VT656J6FMNHE6D
title: Give check a fix for the two findings that have one correct repair
type: task
status: ready
status_reason: null
priority: low
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
references:
  - ref: review:backlog-md
    path: docs/review-backlog-md.md
claim: null
archive: null
created_at: 2026-09-02T18:34:03Z
updated_at: 2026-09-02T18:34:29Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Plan section 11 defines fourteen error codes and five warnings, and check repairs none of them. Backlog.md ships doctor for the same job, and src/core/duplicate-task-repair.ts is 27 KB of it. See section B7 of docs/review-backlog-md.md.

Most of that file is a tax on their sequential IDs and buys us nothing. The narrow question is which of our findings have exactly one correct repair, so a tool can apply it without guessing at intent. Two do.

filename_id_mismatch is a rename to <id>.md. Plan 4 fixes the target name and leaves no second reading.

archive_location_mismatch is a file move, because plan 6.3 already rules that the status wins when the status and the directory disagree.

The rest look repairable and are not. duplicate_id has to choose which file keeps the ID, which is a judgement about which ticket is the real one. dependency_missing can be repaired by unlink --depends-on, and AGENTS.md notes that unlink resolves against the ticket own dependency list precisely so it works on a dangling ID, but dropping the edge and creating the missing ticket are both plausible and only a person knows which. label_unknown is either a typo in the ticket or a gap in the allowlist.

Scope: check --fix covering those two codes and refusing to touch the others. It is the first command that writes without being told which ticket to write, so it needs --dry-run, and it names every path it touched in pathsChanged the way a mutation does. Take the store lock once for the whole pass.
