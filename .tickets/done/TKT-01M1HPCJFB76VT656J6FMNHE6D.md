---
schema: 1
id: TKT-01M1HPCJFB76VT656J6FMNHE6D
title: Give check a fix for the two findings that have one correct repair
type: task
status: done
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
updated_at: 2026-09-02T19:43:49Z
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

## Implementation plan

1. Add ticket.Repair, FixOptions, FixResult, and Store.Fix, holding the store
   lock across planning, moving, and the re-check.
2. Repair by moving the file, never by re-rendering it, since both findings
   are about where a file sits.
3. Drop any repair whose destination is occupied or contested, which is the
   duplicate_id judgement it declines to make.
4. Add check --fix and --dry-run, with repairs and pathsChanged in the report.
5. Record the repairable pair in plan 11, the report fields in 10.3.

## Summary

Added check --fix over the two findings that have one correct repair.

filename_id_mismatch is a rename to <id>.md and archive_location_mismatch is a
move to the directory the status implies. Both are the destination rule
writeTicket already follows, so the repair was already written down as a rule
and only needed applying.

The repair moves the bytes rather than re-rendering. Both findings are about
where a file sits, and a pass that also rewrote contents would canonicalize a
file the caller only asked to move. A test reads the file before and after and
compares them.

Two collision cases fall out and both refuse: a destination that already
exists, and two files wanting the same one. Each means a second ticket is
involved and which is the real one is exactly the duplicate_id judgement this
does not make. The repair is dropped and the finding stays, so the store still
says what is wrong.

--dry-run reports the moves and makes none. Under it pathsChanged is empty
because nothing was written, while repairs is not, and the findings it would
have cleared are still in errors. --dry-run without --fix is refused, because
it would report nothing and read as a clean store.

The lock is held across planning, moving, and the re-check, so the report
describes the store the repairs left.
