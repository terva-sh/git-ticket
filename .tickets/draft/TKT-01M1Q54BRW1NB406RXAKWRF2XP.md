---
schema: 1
id: TKT-01M1Q54BRW1NB406RXAKWRF2XP
title: Build cross-branch reads for list and ready
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - claims
assignees: []
milestone: null
parent: null
dependencies:
  - TKT-01M1HPCJJ1FFHG7HXC8QG1JRAG
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-04T21:27:54Z
updated_at: 2026-09-04T21:27:54Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Build what plan section 8 specifies under "Reading tickets from other branches",
with the JSON in 10.1 and the compatibility note in 12.4. The design is settled,
so this is implementation and not a second round of decisions.

`TKT-01M1HPCJJ1FFHG7HXC8QG1JRAG` (Decide whether a query reads tickets from
other branches) is the spike that settled it. Its Implementation plan carries
the two Git command rows, already worded, that plan 7.4 does not yet have.

The shape, restated so this ticket stands alone:

`--cross-branch` on `list` and `ready`, off by default, taken by no other
command and by no mutation. `for-each-ref` over `refs/heads/` and
`refs/remotes/` both, skipping any ref whose last commit is older than 30 days.
`ls-tree` per surviving ref over the whole `.tickets/` subtree, not
`.tickets/tickets/`, because the store partitions by status and a file's
directory is its status.

Resolution follows 7.5 and does not invent a second rule. The later `updated_at`
wins for display, ties go to the working tree, and a claim is never adjudicated:
a live claim on any scanned ref makes the ticket not ready. Every row carries
`branch`, the short ref name its winning copy came from, null when that is the
working tree.

The 7.4 rows land in this commit and not before it. They were deliberately kept
out of the table while nothing ran them, because `planAllowedGitCommands` builds
its allowlist by reading 7.4 and `TestGitCommandsAreReadOnly` only checks code
against that list. A row with no caller passes and widens the allowlist, which
would hand this change a green guard and skip the review the test exists to
force. The rows are `ls-tree` and `for-each-ref`. Not `log`: `for-each-ref`
returns a branch name and its commit date in one call, and `log`'s other use was
the file mtime guess this format declines.

Worth knowing before starting. The trigger named in the spike has not fired: no
two agents here have claimed the same ticket blind, and the pull request rule is
still absorbing the pressure. So this is specified and ready rather than urgent.

## Acceptance criteria

- [ ] list --cross-branch shows a ticket that exists only on another ref, with branch naming that ref
- [ ] ready --cross-branch omits a ticket whose live claim is on another ref, and never picks between two claims
- [ ] branch is null on every row when the flag is absent, so an ordinary query is unchanged
- [ ] The scan covers refs/heads and refs/remotes both, proven by a test where the only copy is on a remote-tracking ref
- [ ] Plan 7.4 gains rows for ls-tree and for-each-ref in the same commit as the code that runs them
