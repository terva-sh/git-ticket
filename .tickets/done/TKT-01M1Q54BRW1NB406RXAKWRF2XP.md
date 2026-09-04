---
schema: 1
id: TKT-01M1Q54BRW1NB406RXAKWRF2XP
title: Build cross-branch reads for list and ready
type: task
status: done
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
updated_at: 2026-09-04T21:54:00Z
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

- [x] list --cross-branch shows a ticket that exists only on another ref, with branch naming that ref
- [x] ready --cross-branch omits a ticket whose live claim is on another ref, and never picks between two claims
- [x] branch is null on every row when the flag is absent, so an ordinary query is unchanged
- [x] The scan covers refs/heads and refs/remotes both, proven by a test where the only copy is on a remote-tracking ref
- [x] Plan 7.4 gains rows for ls-tree and for-each-ref in the same commit as the code that runs them

## Summary

Built as plan section 8 specifies. `--cross-branch` on `list` and `ready`, off
by default, taken by no mutation. `ticket/crossbranch.go` is the scan and
`Filter.CrossBranch` and `ReadyWith` are how a caller asks for it.

The design needed one correction on contact. Section 8 named two Git commands,
`for-each-ref` and `ls-tree`, and that set cannot work: `ls-tree` lists files
and their blob names without opening them, so reading a ticket off a ref needs a
third. Plan 7.4 gained three rows rather than two, and section 8 now says so.
`cat-file` addresses a blob rather than a path and ref, which is also what makes
the scan affordable. A ticket nobody has touched is the same blob on every
branch, so `blobReader` reads one version once instead of once per ref.

Resolution follows 7.5 as decided: later `updated_at` wins, ties fall to the
working tree, and a claim is never adjudicated. Keeping the claim set separate
from the winning copy turned out to matter more than it looked. When two copies
carry the same second, the working tree wins the display and the claim is still
honoured, because `ClaimedElsewhere` is a set of IDs and not a claim anybody
picked.

Two defects came out of probing a real two-branch repository, and neither would
have shown up in a unit test written from the design.

Printed IDs went ambiguous. `storeAbbreviations` shortened against the working
tree while the listing also printed cross-branch rows, so a local ID could
shorten to a prefix matching the row beneath it, and a branch-only row printed
its whole ID because nothing shortened it. Both halves are now fed the rows
actually being printed.

Worse, a branch-only row came back `isReady: false, isBlocked: false, reason:
""`. That is the exact complaint section 15 records against `readiness` before
it carried a reason at all, reintroduced with a new cause: a working-tree
readiness map has no entry for a ticket that lives only on a ref, so the row got
the zero value. `ReadinessWith` fixes it and `listView` carries the flag to the
one place that explains a listing.

The tests were falsified rather than trusted. Reverting the readiness fix makes
both readiness tests fail with `isReady = true, want false` and `reason = , want
claimed`, and restoring it returns them to green. Commit dates in the tests are
pinned to `2026-09-25` so the 30-day ref window is measured against the injected
`referenceInstant` and never the machine's clock.

On the fifth criterion, 7.4 gained a row for `ls-tree` and one for
`for-each-ref` in this commit, alongside the code that runs them, which is what
was asked. It also gained `cat-file`, which the criterion did not anticipate
because the design had not noticed the command was needed.

The guard did its job on the way through. Adding the scan failed
`TestGitCommandsAreReadOnly` on three unlisted commands, and the pinned
allowlist from the previous ticket failed too, forcing both the rows and a
deliberate edit saying why the table grew. That is the sequencing those two
tests exist to enforce, and this is the first change to test it.
