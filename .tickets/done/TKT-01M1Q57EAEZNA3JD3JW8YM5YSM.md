---
schema: 1
id: TKT-01M1Q57EAEZNA3JD3JW8YM5YSM
title: Fix the 7.4 allowlist parse leaking 7.5 table rows
type: bug
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - policy
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-04T21:29:35Z
updated_at: 2026-09-04T21:38:34Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

`planAllowedGitCommands` in `ticket/gitpolicy_test.go` reads the 7.4 command
table to build the allowlist `TestGitCommandsAreReadOnly` checks code against.
It slices the plan from `### 7.4 The Git commands this code runs` to
`## 8. Query surface`, which is not the end of 7.4. It is the end of 7.5.

7.5's merge-resolution table has a `| Field | Rule |` shape, and its rows are
backtick-quoted field names. Three of them are lowercase with no underscore, so
they match `planGitCommandPattern` exactly as a command row does. The allowlist
the guard actually builds today:

    ['rev-parse', 'symbolic-ref', 'config', 'claim', 'archive', 'extensions']

Only the first three were ever approved. `claim`, `archive` and `extensions` are
ticket fields that leaked in through the range.

`archive` is the one that matters, because `git archive` is a real command. A
call to `runGit(dir, "archive", ...)` passes the guard today with no row, no
reason, and no review. `claim` and `extensions` are not Git commands, so they
widen the allowlist without currently naming anything reachable, but they are
the same defect.

Nothing runs any of the three, so this is a latent hole rather than a live
escape. It is worth fixing on its own rather than in passing, because the guard's
whole value is that its allowlist is exactly what somebody wrote down, and an
allowlist nobody has read in full is the state it exists to prevent.

The fix is the end boundary. `### 7.5` is the next heading, so bounding there
confines the parse to 7.4. Worth adding an assertion that survives the next
person moving a section, rather than trusting the new boundary string: the
parsed slice should not contain 7.5's heading, and the three known field names
should not appear in the allowlist.

Found while writing the cross-branch decisions into section 8, by counting the
7.4 rows to confirm the table had not grown and getting 6 where 3 was correct.

## Acceptance criteria

- [x] The allowlist built from the plan is exactly the commands in 7.4's table
- [x] claim, archive and extensions are absent from it, asserted by a test that names them
- [x] A test fails if the parsed slice ever extends past the end of 7.4 again

## Summary

Fixed by bounding the parse to 7.4's own section. `plan74Section` now slices
from just after the 7.4 heading to the next heading of any level, found with
`planHeadingPattern`, instead of running to the literal `## 8. Query surface`.
That string is the end of 7.5, not of 7.4.

The allowlist the guard builds, before and after, against the same file:

    old bound ->  ['archive', 'claim', 'config', 'extensions', 'rev-parse', 'symbolic-ref']
    new bound ->  ['config', 'rev-parse', 'symbolic-ref']

The three that leaked are field names from 7.5's merge table. Its rows have the
same `| \`name\` |` shape a command row has, and `claim`, `archive` and
`extensions` are lowercase with no underscore, so they matched a pattern written
for commands. `git archive` is a real command, which is what made this an open
door rather than untidiness: `runGit(dir, "archive", ...)` passed the guard with
no row, no reason, and no review.

`TestPlanAllowlistIsSection74Alone` holds all three criteria. It asserts the
slice contains neither `### 7.5` nor `## 8.`, that the three leaked names are
absent from the allowlist, and that the allowlist is exactly `config`,
`rev-parse`, `symbolic-ref`.

That last assertion duplicates the plan on purpose, which is worth defending
because this file's own comment argues the opposite for the allowlist itself.
The difference is what each one is for. Reading the allowlist from the plan
keeps the plan authoritative. Pinning the expected result makes growing 7.4 fail
a test, which is the review the guard exists to force and which a test reading
its answer from the thing it checks cannot force. `gitHelpers` above is
hardcoded for the same reason.

Proved rather than assumed. Adding an `archive` row to 7.4 made the test fail on
both assertions, naming the offending entry and printing the exact diff, and
reverting the row returned the suite to green.

Found while writing the cross-branch decisions into section 8, by counting 7.4's
rows to confirm the table had not grown and getting 6 where 3 was correct.
