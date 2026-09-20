---
schema: 2
id: TKT-01M2Y0H9YDK3325C55QHYNM4K4
title: Read the board from the command line
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
  branch: t3code/canvas-read
  worktree: /home/sothr/.t3/worktrees/git-ticket/t3code-a6d0ff31-layout
  commit: e568e5af38ffab0c2eccd76fe5551fd396c12b1f
  claimed_at: 2026-09-19T23:37:59Z
  expires_at: null
archive: null
created_at: 2026-09-19T23:37:58Z
updated_at: 2026-09-20T00:00:18Z
created_by:
  id: agent:claude/t3code-a6d0ff31
  name: ""
updated_by:
  id: agent:claude/t3code-a6d0ff31
  name: ""
extensions: {}
---

## Description

The read half of `git ticket canvas`, per plan 12.10 and the canvas repository's TKT-01M2ND1RJAGTH8QBF1QK79XPC0 (Read the board from the command line), which holds the source-inspected plan and the interface decisions the user settled on 2026-09-19.

Three words under one command: `canvas show [--board B]` prints each pen with its rule and the tickets it catches, then the Inbox and what falls through; `canvas pens` prints the rules in resolution order; `canvas explain ID` says where a card is and why. Behind them, `layout` gains a pure resolver: a card with a saved coordinate is pinned and routing does not apply; otherwise the first pen in `ruleOrder` whose `requiredLabels` the ticket all carries; otherwise Inbox. That resolver is the reference implementation of routing from here on.

Two things are true only for now and the output says so. The canvas still places every unpinned card in status lanes, so `explain` states that first and reports routing under a heading that names it as not yet applied. And the rule is schema 3's label conjunction; the wider `match` record is the next ticket's.

A board with no layout file is a fact about the store, not a failure: one line saying so, exit 0, and the JSON says the same. JSON kinds are `canvas-board` for show and pens and `canvas-explain` for explain, each with a section under plan 10. Release as v0.21.0, a new command being a minor under 12.4.

## Acceptance criteria

- [ ] git ticket canvas show prints each pen, its rule, and the tickets it catches, then the Inbox and what falls through
- [ ] git ticket canvas pens prints the rules in resolution order
- [ ] git ticket canvas explain ID names the pin or the status-lane placement, then the winning pen and every earlier rule it beat with the labels each lacked, under a heading saying routing is not yet applied
- [ ] The commands run on a checkout with no canvas process, and a board with no layout file is reported in one line with exit 0
- [ ] canvas-board and canvas-explain are published kinds with sections under plan 10, and plan 12.1 and 12.10 name the commands
- [ ] Route has table tests for pinned, first match, order tie, no match, no pens, and a legacy board without routing; the CLI has tests over a store with a layout file and one without

## Implementation plan

The plan is on the canvas repository's TKT-01M2ND1RJAGTH8QBF1QK79XPC0 and is not repeated here. In this tree: layout/resolve.go with Match and Route and their table tests; cli/canvas.go dispatching show, pens, explain over parseFlags and openStore, reading the layout through layout.New(store.Path()); cli/canvas_test.go; docs/plan.md gains 10.10 for the two kinds, three lines under 12.1, and a revised 12.10; envelopeKinds and the completion dump gain the entries. Plan text first, then code, per AGENTS.md.

## Notes

**agent:claude/t3code-a6d0ff31** at 2026-09-19T23:45:15Z

show lists every status, archived included, because the canvas board does: its status filter is empty by default and CardView styles archived cards rather than hiding them. A CLI that dropped them would disagree with the board it reads. Fixture pen colors must be one of the three hex values pens.go accepts; the tests use them rather than names.

**agent:claude/t3code-a6d0ff31** at 2026-09-19T23:45:15Z

The board and explain envelopes carry applied:false so a consumer reads the fact rather than a release number; when the canvas places by rules the flag flips and nothing changes shape. A pinned card's explain still reports a destination: where it would go if released, which is the question asked before releasing it.

**agent:claude/t3code-a6d0ff31** at 2026-09-19T23:48:19Z

PR 211 https://git.local.sothr.com/terva-sh/git-ticket/pulls/211. Terva review 43 reviewed ef95affeac8688e686d00ef7a72cd09d239eb692 against e568e5af38ffab0c2eccd76fe5551fd396c12b1f with two medium findings. Trailing --board parsed as positional: declined, parseFlags re-parses after positionals and the form exits 0 against a real store; a test now holds it. missingLabels null on a match: accepted, fixed in 53d59bb with a marshal test. Fresh review dispatched on 53d59bb.

**agent:claude/t3code-a6d0ff31** at 2026-09-19T23:53:29Z

Terva review 44 reviewed 96d2f8279629 against e568e5af38ff. Board-name traversal (high): declined, layout.Load applies boardNameOK before any path is built; five names tested at the CLI. pens fails on listing failure (medium): accepted in part, the page form no longer lists, the JSON form must because canvas-board carries catches; Store.List tolerates malformed and missing tickets so the described failure did not occur. Fixed in 4ba9e4b and 0d726e3. Lesson recorded: an && chain does not stop a push when just ci fails inside a pipeline; gate the push on the recipe's own exit.

**agent:claude/t3code-a6d0ff31** at 2026-09-19T23:56:36Z

Terva review 45 reviewed 11fc2a0bb334 against e568e5af38ff. exists from a second stat after Load could describe a different file than the board, because a save is a rename (medium): accepted. layout.Read returns the board and whether a file was read from the one read; Load wraps it; the CLI drops its stat. Rule for the next round: ticket notes go in the same commit as the fix, because a push after the dispatch fails the review run.

**agent:claude/t3code-a6d0ff31** at 2026-09-19T23:58:36Z

Terva review 46 reviewed 3200e28b5b75 against e568e5af38ff. Pinned cards still consult rules (medium): accepted in part. The resolver's contract said no rule is consulted while 10.10 promised a pinned card its destination if released; the words contradicted, the behaviour is the one settled for explain. Reworded the contract in resolve.go and plan 12.10: no rule places a pinned card, the rules' answer on a pin is hypothetical, and a placing consumer reads the pin first. Behaviour unchanged.

**agent:claude/t3code-a6d0ff31** at 2026-09-20T00:00:18Z

Terva review 47 reviewed 7d37233134ec against e568e5af38ff: gate passed, one low finding. explain on a missing layout prints three lines where a test comment promised one: accepted in part. The one-line promise was about show and pens; explain answers about a ticket and names it first, then the absent file, then the placement. The comment now says so; output unchanged. PR 211 is ready for the maintainer's merge.
