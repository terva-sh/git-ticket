---
schema: 2
id: TKT-01M2HS32AK5DP6BZPYC72KTBQ1
title: init reports no adoption when the arrived tickets are all done
type: bug
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
references: []
claim: null
archive: null
created_at: 2026-09-15T05:36:59Z
updated_at: 2026-09-15T05:37:13Z
created_by:
  id: agent:terva/adoption-count
  name: ""
updated_by:
  id: agent:terva/adoption-count
  name: ""
extensions: {}
---

## Description

`git ticket init` adopting a `.tickets` directory whose arrived tickets are all
`done` or `archived` prints no `Adopted N tickets` line and no undeclared-series
advice. The adoption itself is correct and the store is usable; what is lost is
the output that tells the reader it happened and how to finish.

Found by the v0.18.0 release verification, running the shipped binary rather
than a working tree, which is what that step is for.

```sh
# an export carrying one done ticket, landed by git am into a bare repository
git am ./out/*.patch
git ticket init --actor human:sothr
# Initialized a ticket store at .tickets
# (no Adopted line, and no `git ticket series add` even for a foreign series)
```

### Cause

`reportAdoption` asks the store to list its tickets, and a bare `Filter` answers
with the open ones. Section 8 leaves `done` and `archived` out of a listing
because a list of work is about work that is still live. Here that rule is
exactly inverted: a cross-store move usually carries work somebody finished, so
the arrived tickets are the ones the default hides, and `len(all) == 0` returns
early.

### How it got in

TKT-01M2HR1M822YGTKQBK2SBAVEX3 replaced a file-walking `adoptedTickets` with the
listing `reportAdoption` already read. The reasoning was sound, that a store
`Init` just created holds tickets only if `Init` adopted them, and it removed a
copy of the store's directory layout from the CLI. What it missed is that the
two do not answer the same question: the file walk counted all four directories
and the listing counts the open ones.

`TestGitAmOfAnExportReachesAUsableStore` did not catch it because `crossCreate`
files a draft, so its adopted ticket was open and the default filter found it.
The test was written against the route rather than against the statuses that
travel it.

### Severity

Cosmetic, and on the case the release is about. `--same-owner` exists for moving
finished work between one owner's stores, so a `done` ticket is the expected
cargo, and for an export carrying a series the receiver has not declared the
reader loses the only pointer to `git ticket series add`.

## Acceptance criteria

- [x] init counts and reports an adopted ticket whose status is done or archived
- [x] The undeclared-series advice appears for a finished ticket too, since it sits behind the same early return
- [x] A test drives a ticket to done through the real transitions and adopts it, so the statuses that travel are covered rather than only the route

## Implementation plan

Pass ticket.Statuses rather than a bare Filter. That is the full exported list, so a status added later is included without another edit here, which is the same reason OpenStatuses is derived rather than written out.

## Notes

**agent:terva/adoption-count** at 2026-09-15T05:37:13Z

Not reverting to the file walk. The listing is still the right source: it goes through the library rather than duplicating the store's directory layout in the CLI, which is what TKT-01M2HR1M822YGTKQBK2SBAVEX3 removed and should stay removed. The defect was the filter, not the decision to ask the store.

**agent:terva/adoption-count** at 2026-09-15T05:37:13Z

Falsified by reverting to the bare Filter. TestAdoptionCountsFinishedWork fails and TestGitAmOfAnExportReachesAUsableStore still passes, which is the coverage gap stated rather than assumed.

## Summary

reportAdoption passes ticket.Statuses instead of a bare Filter, so a listing that section 8 would trim to open work answers with everything the store holds. Verified against the exact failing case from the release verification: a done ticket landed by git am is now counted and its series named.

The listing stays the source rather than going back to walking the directories. It goes through the library instead of duplicating the store's layout in the CLI, which is what the previous ticket removed and should stay removed. The defect was the filter, not the decision to ask the store.

TestAdoptionCountsFinishedWork drives a ticket to done through the real transitions before exporting it, so the statuses that travel are covered rather than only the route. Falsifying it also shows TestGitAmOfAnExportReachesAUsableStore still passing with the bug in place, which is the coverage gap measured rather than assumed.
