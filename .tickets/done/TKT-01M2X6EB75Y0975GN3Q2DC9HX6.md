---
schema: 2
id: TKT-01M2X6EB75Y0975GN3Q2DC9HX6
title: Install targeted Terva PR reviews
type: chore
status: done
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
claim: null
archive: null
created_at: 2026-09-19T16:01:59Z
updated_at: 2026-09-19T18:09:19Z
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
- [x] A manual dispatch against a real PR completes and publishes a review or a clean summary, proving the repo-level BOT_TOKEN reaches the private reviewer
- [x] The review's findings on that PR carry dispositions in the ticket

## Implementation plan

Copy .forgejo/workflows/terva-review.yml from git-ticket-canvas at 0021a8d byte for byte; nothing in it names the repository. Copy docs/pr-reviews.md and change the one sentence that names the canvas's parity and release checks to name this repository's lint, race tests, and store check. Append the canvas's PR review workflow section to AGENTS.md. Ride on PR 208 so the first real dispatch reviews a substantive diff. The comment trigger activates only after the workflow is on main.

## Notes

**agent:claude/t3code-a6d0ff31** at 2026-09-19T16:03:54Z

First dispatch worked: run against PR 208 head ef11378209e9d680643565f82c8213fc682fc1b1 checked out the private reviewer with the repo-level BOT_TOKEN, ran Terva, and published review 32 with one medium finding, so criterion 2 is met. The finding's disposition is on TKT-01M2WD5XZKM5XBYP41ZRF2P0SK and deferred to TKT-01M2X6HV.

## Summary

Landed in PR 208, merged as c70a6b9. The workflow, docs/pr-reviews.md, and the AGENTS.md section are the canvas's, with the docs page naming this repository's CI. The first dispatch, run against PR 208 head ef11378, checked out the private reviewer with the repo-level BOT_TOKEN and published review 32, whose finding carries a disposition on TKT-01M2WD5XZKM5XBYP41ZRF2P0SK. Comment-triggered reviews are live now that the workflow is on main.
