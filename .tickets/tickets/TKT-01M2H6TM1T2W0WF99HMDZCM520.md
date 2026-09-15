---
schema: 2
id: TKT-01M2H6TM1T2W0WF99HMDZCM520
title: Carry parent and criteria state through git ticket import --adopt
type: task
status: in-progress
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
claim:
  actor: agent:terva/same-owner-import
  branch: feat/same-owner-import
  worktree: /home/sothr/workspace/git.local.sothr.com/terva-sh/git-ticket
  commit: 697276ef8a93bc0212e7d3910bd537d9dbd5878d
  claimed_at: 2026-09-15T00:53:43Z
  expires_at: null
archive: null
created_at: 2026-09-15T00:17:48Z
updated_at: 2026-09-15T00:53:44Z
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

- [ ] import --adopt --same-owner carries every acceptance-criteria and definition-of-done tick as the sender wrote it, and names the carry rather than leaving it silent
- [ ] A ticket that arrived done or archived lands in that status; every other status lands draft and is named, because 6.2.1 refuses the rest
- [ ] A parent the export left behind is recorded as an origin-parent: reference, so the link survives where the edge cannot
- [ ] The command prints the summary and status commands that close the origin, and writes nothing in the sending store
- [ ] --same-owner without --adopt previews exactly what the adopt would carry out, from the one ImportPlan both read
- [ ] Without --same-owner, import --adopt behaves exactly as before, proven by TestAdoptCarriesTheStatementOfTheWork passing unchanged
- [ ] Plan 12.8 carries the ruling and section 15 records it settled with a reopen trigger, and the authored-document entry no longer says import --adopt unticks unconditionally
- [ ] A handoff document in docs/ carries the cross-store move procedure the ticket asks for in ledger's docs/workspace-layout.md
- [ ] The reported case is reproduced end to end in the built binary: a done ticket with one of two boxes ticked and a parent left behind arrives done, one box ticked, with check --strict green

## Definition of done

- [ ] just ci is green: gofmt, vet, go test -race ./..., and check --fix --dry-run --strict

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
