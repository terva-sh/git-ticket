---
schema: 2
id: TKT-01M2X6EB75Y0975GN3Q2DC9HX6
title: Install targeted Terva PR reviews
type: chore
status: in-progress
status_reason: null
priority: normal
due_on: null
labels:
  - area/ci
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
claim:
  actor: agent:claude/t3code-a6d0ff31
  branch: t3code/layout-package
  worktree: /home/sothr/.t3/worktrees/git-ticket/t3code-a6d0ff31-layout
  commit: 70d2a379482a7a39571d5723be618de714a2a33c
  claimed_at: 2026-09-19T16:01:59Z
  expires_at: null
archive: null
created_at: 2026-09-19T16:01:59Z
updated_at: 2026-09-19T16:02:17Z
created_by:
  id: agent:claude/t3code-a6d0ff31
  name: ""
updated_by:
  id: agent:claude/t3code-a6d0ff31
  name: ""
extensions: {}
---

## Description

git-ticket-canvas installed a targeted Terva review workflow under its TKT-01M2W22SBF554JHMVPMDWPFTY4 (Install targeted Terva PR reviews) and has been taking findings on every PR since. This repository has CI but no model review, so a PR here gets lint, race tests, and the store check, and nobody reads the diff for the kind of thing those miss.

Install the same workflow here, unchanged where it can be: the pinned reviewer commit, the checksum-pinned Terva binary, the same provider, model, profile, and publication policy, the same maintainer allowlist. The docs page and the AGENTS.md section come with it, adapted to what this repository's CI is.

Two facts about this repository that the canvas did not have to think about. Its Forgejo secrets hold a repo-level BOT_TOKEN, which shadows the org-level one the canvas uses; whether it can check out the private reviewer is proven by the first run, not by reading. And its CI image is golang:1.25-alpine while the review job uses golang:1.27-alpine, which the canvas runs prove is present on the registry; the review job builds nothing from this repository, so the Go version there is only what the runner needs.

## Acceptance criteria

- [x] The terva-review workflow, docs/pr-reviews.md, and the AGENTS.md section are in the tree, with the reviewer, Terva binary, and checkout helper pinned to the same revisions the canvas uses
- [ ] A manual dispatch against a real PR completes and publishes a review or a clean summary, proving the repo-level BOT_TOKEN reaches the private reviewer
- [ ] The review's findings on that PR carry dispositions in the ticket

## Implementation plan

Copy .forgejo/workflows/terva-review.yml from git-ticket-canvas at 0021a8d byte for byte; nothing in it names the repository. Copy docs/pr-reviews.md and change the one sentence that names the canvas's parity and release checks to name this repository's lint, race tests, and store check. Append the canvas's PR review workflow section to AGENTS.md. Ride on PR 208 so the first real dispatch reviews a substantive diff. The comment trigger activates only after the workflow is on main.
