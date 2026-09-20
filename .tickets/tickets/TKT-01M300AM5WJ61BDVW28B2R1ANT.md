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
updated_at: 2026-09-20T18:13:43Z
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

- [ ] pen add, pen rm, pen order, place, release, frame add and inbox write the layout file and answer with canvas-board under --json
- [ ] git ticket check reports a layout that does not parse or validate, a card or member naming a ticket that does not exist, and a pen label outside the allowlist, with a store fixture per new code
- [ ] check --fix rewrites a valid layout that is not in canonical form and touches nothing else in it
- [ ] No command computes a card position; only place writes one, from its argument
- [ ] A write refuses rather than producing a layout that check would reject, and the file is unchanged after a refusal
- [ ] Plan 11, 12.1 and 12.10 and the corpus README name the codes and the words

## Implementation plan

Source read on 2026-09-21: layout/{layout,pens,frames,resolve}.go, cli/canvas.go, ticket/{check,fix,errors,id}.go, testdata/README.md, plan 10.3, 10.10, 11, 12.1, 12.10.

Import direction. check must reach layout.Parse, so ticket imports layout, and layout drops its one import of ticket (ValidID, for frame members). The ID grammar moves to internal/ids; ticket.ValidID and layout call it. Rejected: a registration hook layout installs on import, which hides the dependency and leaves a library caller without the check; and composing the layout pass in cli, which the fixture corpus and TestCorpusCoversEveryPlanCode could not reach, so the codes would be published with nothing holding them.

Codes, per plan 11. layout_invalid, an error: a file under canvas/ whose name is outside the board grammar, or that does not parse or validate, one finding per file with the message from layout. layout_ticket_missing, a warning: a card or a frame member names a ticket the store does not have; field is cards.ID or frames.ID.members. layout_not_canonical, a warning: a valid file whose bytes differ from render, which is the one layout finding with exactly one correct repair, so check --fix rewrites it the way it rewrites epics.md. label_unknown is reused for a pen label outside the allowlist, field pens.ID.requiredLabels, because the condition is the same one. A store fixture per new code under testdata/stores/layout-*, each with canvas/default.yml.

Writer. layout.Store gains Modify(board, func(*Board) error), which loads under the package mutex, applies, validates, normalizes and saves by rename; a validation error leaves the file as it was. Every CLI write goes through it. Locking stays the package's own, per 12.10: the canvas does not take the store lock of section 7 for layout writes, so taking it in the CLI alone would serialize CLI against CLI and nothing else; recorded as the gap it is.

Words, in cli/canvas.go, dispatched from the one runCanvas over the one FlagSet: pen add ID --title T --label L... --at X,Y --size W,H [--color C] [--pin X,Y]; pen rm ID; pen order ID...; place ID --at X,Y; release ID...; frame add ID --title T --at X,Y --size W,H [--color C] --member ID...; inbox --at X,Y. A flag the word does not take is a usage error. pen add appends to ruleOrder; --status, --type and --parent arrive with the match record ticket. pin defaults to the pen's origin, since the reference resolver reads no pin and a person who wants one names it. frame add takes an ID and geometry, which the design sketch left out: the record requires w and h, and nothing here computes them. release refuses a card that is not pinned rather than answering yes to nothing. Ticket IDs resolve through the store, so a prefix works. Under --json every write answers with canvas-board, the board after the write, and 10.10 says so.

Docs: plan 11 tables and the fixture note, 12.1 lines, 12.10, 10.10; testdata/README.md store-scoped list; README. Release as v0.23.0, a minor under 12.4.
