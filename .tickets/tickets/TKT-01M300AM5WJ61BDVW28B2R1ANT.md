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
updated_at: 2026-09-20T18:13:17Z
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
