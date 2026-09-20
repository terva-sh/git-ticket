---
schema: 2
id: TKT-01M300AM5WJ61BDVW28B2R1ANT
title: Write board rules from the command line
type: task
status: in-progress
status_reason: null
priority: normal
due_on: null
labels:
  - area/cli
  - area/format
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
claim:
  actor: agent:claude/t3code-a6d0ff31
  branch: t3code/canvas-write
  worktree: /home/sothr/workspace/git.local.sothr.com/terva-sh/git-ticket
  commit: 23452cc1d40f2c31d9b809b8426ce2001039ab77
  claimed_at: 2026-09-20T18:13:17Z
  expires_at: null
archive: null
created_at: 2026-09-20T18:12:48Z
updated_at: 2026-09-20T18:43:11Z
created_by:
  id: agent:claude/t3code-a6d0ff31
  name: ""
updated_by:
  id: agent:claude/t3code-a6d0ff31
  name: ""
extensions: {}
---

## Description

The write half of git ticket canvas, per plan 12.10 and the canvas repository's TKT-01M2ND1RMSJ1KAEZM7HZX5DEHR (Write board rules from the command line), which holds the design and the interface decisions. Words under one command: pen add, pen rm, pen order, place, release, frame add, and inbox, each writing .tickets/canvas/<board>.yml through the layout package and refusing rather than writing a board check would reject. check reads the same file in the same pass as the tickets, and --fix repairs the one finding with exactly one correct repair. No command computes a card position: place writes the coordinate its caller chose, and everything else writes rules. Release as a minor under 12.4, since it adds command words and finding codes.

## Acceptance criteria

- [x] pen add, pen rm, pen order, place, release, frame add and inbox write the layout file and answer with canvas-board under --json
- [x] git ticket check reports a layout that does not parse or validate, a card or member naming a ticket that does not exist, and a pen label outside the allowlist, with a store fixture per new code
- [x] check --fix rewrites a valid layout that is not in canonical form and touches nothing else in it
- [x] No command computes a card position; only place writes one, from its argument
- [x] A write refuses rather than producing a layout that check would reject, and the file is unchanged after a refusal
- [x] Plan 11, 12.1 and 12.10 and the corpus README name the codes and the words

## Implementation plan

Source read on 2026-09-21: layout/{layout,pens,frames,resolve}.go, cli/canvas.go, ticket/{check,fix,errors,id}.go, testdata/README.md, plan 10.3, 10.10, 11, 12.1, 12.10.

Import direction. check must reach layout.Parse, so ticket imports layout, and layout drops its one import of ticket (ValidID, for frame members). The ID grammar moves to internal/ids; ticket.ValidID and layout call it. Rejected: a registration hook layout installs on import, which hides the dependency and leaves a library caller without the check; and composing the layout pass in cli, which the fixture corpus and TestCorpusCoversEveryPlanCode could not reach, so the codes would be published with nothing holding them.

Codes, per plan 11. layout_invalid, an error: a file under canvas/ whose name is outside the board grammar, or that does not parse or validate, one finding per file with the message from layout. layout_ticket_missing, a warning: a card or a frame member names a ticket the store does not have; field is cards.ID or frames.ID.members. layout_not_canonical, a warning: a valid file whose bytes differ from render, which is the one layout finding with exactly one correct repair, so check --fix rewrites it the way it rewrites epics.md. label_unknown is reused for a pen label outside the allowlist, field pens.ID.requiredLabels, because the condition is the same one. A store fixture per new code under testdata/stores/layout-*, each with canvas/default.yml.

Writer. layout.Store gains Modify(board, func(*Board) error), which loads under the package mutex, applies, validates, normalizes and saves by rename; a validation error leaves the file as it was. Every CLI write goes through it. Locking stays the package's own, per 12.10: the canvas does not take the store lock of section 7 for layout writes, so taking it in the CLI alone would serialize CLI against CLI and nothing else; recorded as the gap it is.

Words, in cli/canvas.go, dispatched from the one runCanvas over the one FlagSet: pen add ID --title T --label L... --at X,Y --size W,H [--color C] [--pin X,Y]; pen rm ID; pen order ID...; place ID --at X,Y; release ID...; frame add ID --title T --at X,Y --size W,H [--color C] --member ID...; inbox --at X,Y. A flag the word does not take is a usage error. pen add appends to ruleOrder; --status, --type and --parent arrive with the match record ticket. pin defaults to the pen's origin, since the reference resolver reads no pin and a person who wants one names it. frame add takes an ID and geometry, which the design sketch left out: the record requires w and h, and nothing here computes them. release refuses a card that is not pinned rather than answering yes to nothing. Ticket IDs resolve through the store, so a prefix works. Under --json every write answers with canvas-board, the board after the write, and 10.10 says so.

Docs: plan 11 tables and the fixture note, 12.1 lines, 12.10, 10.10; testdata/README.md store-scoped list; README. Release as v0.23.0, a minor under 12.4.

## Notes

**agent:claude/t3code-a6d0ff31** at 2026-09-20T18:22:29Z

Built on 2026-09-21; just ci green. Two things worth knowing that the plan does not say. The corpus requires every ticket ID unique across store fixtures, so each layout-* case is one ticket minted for it rather than a copy of clean. And a same-millisecond pair of tickets share ten characters, so the CLI tests pass full IDs to place and --member where a person would type a prefix. Tried and rejected: registering the layout check from layout's init, because the dependency would be invisible and a library caller of Store.Check would silently lack it.

**agent:claude/t3code-a6d0ff31** at 2026-09-20T18:29:45Z

Terva review 50 reviewed 056b41a37474: two medium findings. Concurrent CLI processes could overwrite each other through the in-process mutex alone: accepted. Every layout writer now takes a file lock on the canvas directory, git-ticket/canvas.lock under the common Git directory beside the store lock, through internal/filelock, which the ticket store's lock now also uses; the canvas takes it the moment it builds against this version, so the gap 12.10 had recorded as open is closed for both writers rather than half-closed for one. A pen label outside the allowlist is written and then reported by check: accepted in part. The allowlist is advisory under plan 11 and create files a ticket under an unlisted label without a word, so pen add writes the pen and warns on stderr naming the label, and a test holds that. Refusing would make a rule harder to write than the ticket it catches.

**agent:claude/t3code-a6d0ff31** at 2026-09-20T18:31:54Z

Terva review 52 reviewed b565c1f74b48: one medium, one low, both accepted. Under --json the ticket listing ran after the rename, so a listing that failed would report a committed write as failed; it now runs before the write, since a write changes no ticket. The unlisted-label warning printed before Modify could refuse; it is now said only after the write lands, and the test holds that a refused duplicate says nothing about labels.

**agent:claude/t3code-a6d0ff31** at 2026-09-20T18:35:00Z

Terva review 53 reviewed ef53d2d1e2af: one high, one low, both accepted. check --fix wrote the planner's canonical bytes with os.WriteFile, outside the layout lock, so a canvas save between planning and repair would have been overwritten; the repair now calls layout.Store.Canonicalize, which re-reads and re-renders under the lock, and a test saves between the two and checks the save survives. An explicit empty flag such as --at= read as absent; string flags now record whether they were passed, and --at= on show and --title= on place are usage errors under test.

**agent:claude/t3code-a6d0ff31** at 2026-09-20T18:40:32Z

Terva review 56 reviewed 23ee2ac82fc8: one medium, one low. Archived tickets reported as missing from a layout: declined. The existence callback reads the map Check builds from load(), which walks every store directory including archive/, so an archived ticket is present there; the name live means parsed, not open. The layout-ticket-missing fixture now carries an archived ticket as a frame member and its expectation is unchanged, which holds the answer. A repair reported when a concurrent save had already made the file canonical: accepted. Canonicalize's changed result now decides whether the repair is in the report, through one applyRepairs that every repair kind goes through, with a test that canonicalises between planning and applying.

**agent:claude/t3code-a6d0ff31** at 2026-09-20T18:43:11Z

Terva's fifth round found nothing at 65c57709099a and the clean status is on the head; Forgejo CI is green. PR 215 is ready for the maintainer's merge, to release as v0.23.0 together with the match record on the stacked branch t3code/match-record if that lands first.
