---
schema: 2
id: TKT-01M1WTBJFM26T481B15T6MGCWT
title: Say which transition unblocks a claim refused for its status
type: bug
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - claims
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-07T02:15:03Z
updated_at: 2026-09-07T02:22:18Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

terva's third handoff, finding 1. Measured against v0.14.0.

Claiming a draft refuses with `validation_failed` and the message "a ticket in
draft cannot be claimed". Every word of that is true and the refusal itself is
right, because nobody should work a ticket nobody has promoted. What it does
not say is the step that clears it, so an agent that has just filed a ticket
and wants to start work has to find `status ready` somewhere else. terva lost
a few minutes to exactly that.

The refusal is generic, and that is what makes the fix less obvious than it
looks. `ClaimTicket.apply` in `ticket/mutation.go` allows ready, in-progress,
blocked and review, so draft, done and archived all reach the same sentence.
terva suggested appending "promote it with `status ready` first". That is
right for a draft and wrong for the other two, and would teach a wrong step to
two of the three callers who see it. The transition table answers differently
for each: draft goes to ready, done goes to in-progress, and archived goes
back to ready, which `unarchive` is the command for.

Naming the remedy is already the house convention, which makes this message
the outlier rather than a new ask. `invalid_root` names the parent to pass,
`migration_incomplete` says to run migrate, `epics_index_stale` says to run
`check --fix`, and `remove` names both `unlink` and `--force`.

## Acceptance criteria

- [x] Claiming a draft names status ready as the step that clears the refusal
- [x] Claiming a done ticket names status in-progress rather than status ready
- [x] Claiming an archived ticket names the way back to a claimable status
- [x] The message still states the status it refused on, so a reader of the old sentence loses nothing
- [x] A test covers all three refused statuses and fails if a remedy goes missing

## Notes

**agent:terva/mieli** at 2026-09-07T02:22:18Z

Built and shipped in v0.14.1. All five criteria tick.

The build deviated from terva's suggested wording, and the deviation is the
substance of the ticket. They proposed appending "promote it with `status
ready` first" to the message. That message is reached by draft, done and
archived alike, so one sentence would have taught the wrong step to two of the
three callers. `claimRemedy` returns a different transition for each, and an
unrecognised status returns the empty string so the message stays as it was
rather than guessing.

Running the real binary caught what the unit test could not. The first version
told a done ticket to run `status ID in-progress` and stopped there, which
earns a second refusal, `invalid_field` on `status_reason`, because reopening
from done needs a reason. A remedy that lands the reader on another refusal
has not saved the trip it exists to save. The message now names `--reason R`.

The test was rewritten because of that miss. It performed only string checks,
which prove a sentence exists and say nothing about whether following it
works. Each case now applies the remedy it describes and claims afterwards.
TestReopeningFromDoneNeedsAReason pins the constraint behind the wording, so a
later tidy-up that makes all three remedies look alike fails the build.

## Summary

Shipped in v0.14.1. Claiming a ticket whose status forbids it now names the
move that clears it, and the move differs by status: `status ID ready` for a
draft, `status ID in-progress --reason R` for a done ticket, and `unarchive
ID` for an archived one. An unrecognised status keeps the old message rather
than guessing a remedy.

TestClaimRefusalNamesTheTransition performs each remedy and claims afterwards,
so the advice is held to working rather than to existing. That check is what
the done wording came from: naming `status ID in-progress` alone earns a
second refusal, because reopening from done needs a reason.
