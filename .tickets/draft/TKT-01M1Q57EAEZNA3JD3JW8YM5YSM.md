---
schema: 1
id: TKT-01M1Q57EAEZNA3JD3JW8YM5YSM
title: Fix the 7.4 allowlist parse leaking 7.5 table rows
type: bug
status: draft
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
updated_at: 2026-09-04T21:29:35Z
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

- [ ] The allowlist built from the plan is exactly the commands in 7.4's table
- [ ] claim, archive and extensions are absent from it, asserted by a test that names them
- [ ] A test fails if the parsed slice ever extends past the end of 7.4 again
