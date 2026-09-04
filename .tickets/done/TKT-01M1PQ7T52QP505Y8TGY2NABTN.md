---
schema: 1
id: TKT-01M1PQ7T52QP505Y8TGY2NABTN
title: Build git ticket remove, per plan 9.1
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels: []
assignees: []
milestone: null
parent: null
dependencies:
  - TKT-01M1PCYC14VD3TJYDE1EWC671Y
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-04T17:25:07Z
updated_at: 2026-09-04T18:15:02Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Plan section 9.1 settles the design and this is the work.

`git ticket remove ID` deletes a ticket file, refusing two cases and naming what
it found in each. Another ticket naming this one in `dependencies` or `parent`
is the first, because the store fails `check` afterwards and `--fix` cannot
repair it. Carrying `Notes`, `Comments`, or a `Summary`, or holding a claim, is
the second, because `archive` is the operation for work that happened.

`--force` overrides both and reports every dangling reference it created, naming
`unlink` as the repair.

The refusal is the product. Deleting an unreferenced ticket file already leaves
`check --strict` green, so the deletion is the part that was never hard, and a
command that only wrapped one unlink would earn nothing.

### Shape

It sits on the `Store` beside `Create` rather than being a `Mutation`, because
every mutation rewrites a ticket in place and this deletes the file. It runs no
Git command, per 7.4, so the deletion is left in the working tree for a person
to stage.

It returns the ticket it removed, which is the last state rather than a new one,
so `--json` hands a caller the content back without going to Git for it. That
needs a decision recorded in section 10 about which kind carries it, since 10.1
describes a `ticket` kind for a ticket that exists.

### Watch for

`resolveID` first, per 5.5, because every command taking an ID accepts a unique
prefix and a mutation that skips it rejects a valid one.

The refusal needs to scan the store for tickets naming this ID, which is a read
of every ticket rather than a lookup. `deps --dependents` already walks that
direction and is the thing to reuse.

Removing an epic leaves `epics.md` stale. That is `epics_index_stale`, which
`check --fix` already repairs, and this command should not rewrite the index
itself.

## Notes

**agent:terva/mieli** at 2026-09-04T17:44:07Z

Parked mid-build to take the ticket-title work first. Claimed and in-progress on
`feat/remove-command`, which holds one commit and nothing else.

Done so far: plan section 10 records which kind carries a removed ticket.
`remove` emits `mutation-result` with the id, the revision the ticket had, and
the path it no longer occupies. It keeps the two-key `ticket` stub even though
`remove` is the one command that breaks the stub's stated justification, since
`show` afterwards is `ticket_not_found`. Adding a key that carries the body is
additive under 12.4 and removing one is a break, so the smaller envelope is the
reversible way round. The sharp edge is recorded there too: a ticket created and
removed without an intervening commit is gone, because Git never saw it.

Not started: `Store.Remove`, the CLI command, the tests, and the doc updates
that turn 9.1 from "not built" into built.

Two findings from the survey worth keeping, so they are not re-derived:

`Deps` with `Dependents` walks `dependencies` only and never `parent`, so the
referential refusal needs its own scan over both rather than reusing it.

`Finding` carries exactly `Code`, `File`, `Ticket`, `Field`, with `Message`
already `json:"-"`. That precedent is what lets a human-only field be added to a
finding without touching the fixture sidecars.

## Summary

Shipped. `git ticket remove ID` deletes a ticket file and refuses the two cases
plan 9.1 names, with `--force` overriding both and reporting what it broke.

`Store.Remove` sits beside `Create` and returns a `RemoveResult`, not a
`Result`. A `Result` carries the ticket a write produced and a removal produces
none, and `Dangling` has nowhere to live on a `Result` because no other
operation can create one.

Two new codes, `ticket_referenced` and `ticket_touched`, added to plan section
10's stable list and to `OperationCodes`, so `schema` publishes them without
further wiring. They are separate codes rather than one `validation_failed`
apiece because the repairs differ, and a caller reading the code should know
which to do. `claim_conflict` is the precedent: a refusal belonging to one
command and overridden by the same `--force`.

Three things this ticket's own description got wrong, all corrected in the code
and the plan.

The description said to reuse `deps --dependents`. It cannot be reused:
`Deps` with `Dependents` walks `dependencies` and never `parent`, so an epic's
children would not have counted as referrers. `referencesTo` scans both.

The 9.1 table and its prose disagreed about what blocks a removal. The table
listed notes, comments, summary, and a claim; the prose added `archive` and then
said the body must carry "a `Description` and nothing else". That last part
would refuse any ticket filed with `create --plan` or `create --ac`, which seed
those sections at filing time, so it would have refused exactly the mistakes
that were filed most carefully. Settled as: notes, comments, summary, claim, or
an archive record block it, and a plan, acceptance criteria, and a definition of
done do not. `TestRemoveTakesAFiledTicketWithItsPlan` holds that.

Both the refusal message and the `--force` warning named `unlink` as the repair
for every reference. `unlink --depends-on` drops a dependency and does nothing
to a parent, so for an epic it sent the reader to a command that reports success
and changes nothing. Each now names the repair its field actually takes, and a
test asserts the parent case does not offer `unlink`.

21 tests across `ticket/remove_test.go` and `cli/remove_test.go`, including the
constraint that makes the `--force` warning safe: it goes to stderr, so stdout
stays a parseable envelope.

Docs followed: 9.1 is no longer "Not built", the 12.1 usage block lists the
command, and the AGENTS.md gotcha that told a reader to `rm` the file now names
`remove`. README says why it is the one operation that does not undo.
