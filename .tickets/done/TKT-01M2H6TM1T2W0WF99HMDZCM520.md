---
schema: 2
id: TKT-01M2H6TM1T2W0WF99HMDZCM520
title: Carry parent and criteria state through git ticket import --adopt
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels: []
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references:
  - ref: origin-ticket:TKT-01M2DQPQS36PP4TV2M4SHX4T8W
    path: null
  - ref: origin-store:ledger
    path: null
claim: null
archive: null
created_at: 2026-09-15T00:17:48Z
updated_at: 2026-09-15T01:56:48Z
created_by:
  id: agent:claude/skill-bundle-2
  name: ""
updated_by:
  id: agent:terva/same-owner-import
  name: ""
extensions: {}
---

## Description

Filed from a reflect backlog item, 2026-09-13. Adopting a ticket into a project store with git ticket import --adopt drops the parent link and every acceptance-criteria tick, so the adopted copy lands in draft with every box unchecked while the origin copy is ticked and done. Observed moving TKT-01M2CM7X1KAEXZY6ERVFNTXQR3 into warricksothr/agent-session. This belongs to terva-sh/git-ticket; export it there when picked up. Suggested mechanism: a move operation, or an --adopt option, that carries parent, criteria state, and provenance, and closes the origin with a summary naming the adopted id. Also add a one-paragraph procedure to docs/workspace-layout.md.

## Acceptance criteria

- [x] import --adopt --same-owner carries every acceptance-criteria and definition-of-done tick as the sender wrote it, and names the carry rather than leaving it silent
- [x] A ticket that arrived done or archived lands in that status; every other status lands draft and is named, because 6.2.1 refuses the rest
- [x] A parent the export left behind is recorded as an origin-parent: reference, so the link survives where the edge cannot
- [x] The command prints the summary and status commands that close the origin, and writes nothing in the sending store
- [x] --same-owner without --adopt previews exactly what the adopt would carry out, from the one ImportPlan both read
- [x] Without --same-owner, import --adopt behaves exactly as before, proven by TestAdoptCarriesTheStatementOfTheWork passing unchanged
- [x] Plan 12.8 carries the ruling and section 15 records it settled with a reopen trigger, and the authored-document entry no longer says import --adopt unticks unconditionally
- [x] A handoff document in docs/ carries the cross-store move procedure the ticket asks for in ledger's docs/workspace-layout.md
- [x] The reported case is reproduced end to end in the built binary: a done ticket with one of two boxes ticked and a parent left behind arrives done, one box ticked, with check --strict green

## Definition of done

- [x] just ci is green: gofmt, vet, go test -race ./..., and check --fix --dry-run --strict

## Notes

**agent:terva/same-owner-import** at 2026-09-15T00:53:12Z

Groomed. Verified every claim against the tree at f57f7b4 by building the binary
and running an export and an import between two scratch stores, rather than by
reading the code. Two of the three claims need correcting, and the correction is
what shapes the work.

The parent claim is half right. A parent travelling in the same export is
carried and reminted: exporting a parent and its child together lands the child
with its parent edge pointing at the newly minted parent ID, with no warning,
which the run confirmed. A parent left behind is dropped and named twice, once
by export as a warning naming the edge and again by import as "parent ... not
carried, this export does not include it". The reported case exported one
ticket, so its parent stayed at the origin. The edge genuinely cannot survive a
remint, because it would name an ID this store has never seen and
parent_missing is an error. What is missing is not the edge but any record that
the parent existed at all, and that is the repairable part.

The tick claim is right and the behaviour is deliberate. Plan 12.8 argues it:
a tick is evidence about the sender's work and says nothing about whether this
store has met the criterion, which is the same argument that files every
imported ticket as a draft. TestAdoptCarriesTheStatementOfTheWork pins it, with
a comment saying so at the assertion. So this ticket is not a bug report
against that rule; it is the report that settles whether the rule has a second
case.

The origin claim is right. The sending store is untouched by an import and
holds no pointer to the adopted ID, which the run confirmed by reading the
origin ticket's references back as empty after the export.

What the three add up to is that import --adopt answers a different question
than the one the report asks. It models somebody else's contribution arriving,
where the receiver never agreed to the sender's evidence. The reported case is
one owner moving their own ticket between their own stores, where the ticks are
that owner's evidence about that owner's work and unticking them destroys a
record nothing else holds. So the fix is a second case on the existing rule,
asserted by the person who knows what the tool cannot, rather than a change to
what --adopt means on its own.

Settled with the user before the branch, per the rule that a plan question is
not settled in code: the mechanism is a --same-owner flag on import rather than
a new move operation. The vocabulary counts decided the name. "move" is spent
217 times across the tree, including the store's own sense of a status change
moving a file between directories, so a move spelling would collide with a
meaning the store already has; "same-owner" and "carry-state" are both unspent,
and "mine" has fifteen hits that are all test-local variables. --same-owner
follows the grammar --adopt already set, a sentence the user asserts that the
tool cannot work out for itself.

The last line of the description asks for a paragraph in
docs/workspace-layout.md. That file is in the ledger repository, not this one,
and the policy here forbids writing into a sibling repository. It ships as a
handoff document in docs/ for the user to carry over instead.

## Summary

Shipped as `import --same-owner`, settled with the user before the branch and
recorded in plan 12.8 under "One owner, two stores", with the reopen trigger in
section 15.

The ruling. The interchange rule was not wrong; it was answering a different
question. `import --adopt` models somebody else's contribution arriving, and its
argument for unticking rests on the tick being the sender's evidence. That
premise fails when one person keeps both stores, so the flag is a second case on
the rule rather than an exception to it: the evidence travels and the vocabulary
does not. Ticks, status and the filing instant travel. Labels, milestones, due
dates and blocks_on reconcile exactly as before, which is what keeps
`check --strict` green on arrival. Status stops where 6.2.1 already stops it,
done and archived through and everything else into draft and named, because
widening CreateOptions.Status would walk around the gate that makes promotion a
human call.

The parent half was answered differently from how it was asked, because the
grooming run showed the report was half right there: a parent travelling in the
same export is already carried and reminted, and only one left behind is dropped.
The edge genuinely cannot survive a remint, so what ships is an origin-parent:
reference, the link rather than the edge, and only under --same-owner because a
stranger's parent ID resolves nowhere the receiver can follow.

Closing the origin is printed advice and never action. import runs in the
receiving store, 7.3 forbids a sync helper rewriting another worktree, and the
sending store may be open in another session. Measured rather than assumed: the
sending store's files hash identically before and after an adopt, and its
worktree stays clean.

What the work taught, both of them worth carrying. The report was measured before
it was believed and one of its three claims did not survive, which is the whole
reason the parent is answered with a reference. And the first real run of the
finished flag found a defect the suite would have missed: the parent was reported
as "kept as an origin-parent reference" and then as "not carried" one line later,
two true sentences that contradict each other. PlannedTicket.keptParent now
exists so droppedEdges reads the decision rather than recomputing the condition,
and TestSameOwnerCarriesTheEvidence asserts the contradiction cannot come back.

Every criterion is ticked and each was proven by a run rather than by the code
that should have satisfied it. Both falsifications were checked: disabling the
carry fails three of the five CLI tests, and restoring the parent double-naming
fails the fourth. The handoff procedure in
docs/handoff-ledger-cross-store-move.md was run verbatim, including the
--help check it tells its reader to run.

Two things deliberately left. The release itself is a separate decision: this is
a minor under 12.4 by the new-surface rule, five new exported ChangeKind values
and a new flag, with nothing broken and no schema move, but no tag is cut here.
And the preview's "1 ticket already use a series" disagrees with itself about
number; it predates this branch and belongs in its own ticket rather than riding
into this diff.
