---
schema: 2
id: TKT-01M35Y5PN7D80KCJR6JF0DTH8Q
title: Read the Terva review settings from organization variables
type: chore
status: review
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
created_at: 2026-09-23T01:30:37Z
updated_at: 2026-09-23T01:36:20Z
created_by:
  id: agent:claude/terva-migration-a
  name: ""
updated_by:
  id: agent:claude/terva-migration-a
  name: ""
extensions: {}
---

## Description

terva-sh/terva-action-code-review PR 41, merged at 7090fc19699fda481d9138f0dcf73f80ee8cab06, removed the action's defaults for provider, model and thinking. A run missing any of them now fails with missing_provider, missing_model or missing_thinking. The organization decides them once, as the terva-sh organization Actions variables TERVA_REVIEW_PROVIDER, TERVA_REVIEW_BASE_URL, TERVA_REVIEW_MODEL and TERVA_REVIEW_THINKING, which already exist.

This repository's terva-review workflow pins the action at an older commit, installs Terva 0.137.0, and hardcodes openai-compatible, the CPA API URL, gpt-5.6-sol and low thinking. Move it to the org variables, and move the pin and Terva together, because 7090fc1 needs Terva 0.138.2. The user asked for every consumer to be migrated, a review requested on the PR, and its findings addressed. Merging is not part of this ticket's authorization.

## Acceptance criteria

- [x] The workflow pins the action at 7090fc1, installs Terva 0.138.2 by checksum, and reads provider, URL, model and thinking from the org variables
- [x] docs/pr-reviews.md names the new pin and Terva version and points at the org variables instead of listing defaults
- [x] A review dispatched from the PR branch reports gpt-6-sol at medium thinking
- [x] Every finding from that review carries a disposition here

## Implementation plan

Follow the reference consumer, examples/forgejo-review.yml, and docs/installation.md (Organization settings, Adopting a new pin) in terva-action-code-review at 7090fc1.

1. .forgejo/workflows/terva-review.yml: action ref to 7090fc19699fda481d9138f0dcf73f80ee8cab06; Terva to v0.138.2 with sha256 679ae4018ada55c1f2c298656552fe460bfcdcd79edee12a0ed0e145776eb804 (verified by downloading the asset); provider, base-url, model and thinking read from vars.TERVA_REVIEW_*. Nothing else in the workflow changes.
2. docs/pr-reviews.md: name the new pin and Terva version, and replace the sentence listing the old defaults with one pointing at the org variables. It names the variables, not their values, so a model change on the org does not leave this page stale.
3. Dispatch a review from the PR branch, so the new workflow runs, and confirm the run reports gpt-6-sol at medium thinking, which proves the variables reached the action.
4. Record the PR, run, reviewed head/base and each finding's disposition here.

Rejected: keeping the values in the workflow as a fallback. The action no longer has defaults on purpose, so that a model nobody in the organization chose never reviews code; a per-repo copy is exactly the drift the variables remove.

## Notes

**agent:claude/terva-migration-a** at 2026-09-23T01:36:20Z

PR: https://git.local.sothr.com/terva-sh/git-ticket/pulls/218. Reviewed head d4e28db4264351f231bc6d364d357903e297b67c, base 3ea908e7309be1f14016637ed46650a6fe432536.

Review request pr-218-migration:agent.claude.terva-migration-a, dispatched from the PR branch so the new workflow ran. Actions run https://git.local.sothr.com/terva-sh/git-ticket/actions/runs/548, Terva run 2a78b93c-387a-44d6-b10a-3a159acead10, clean summary https://git.local.sothr.com/terva-sh/git-ticket/pulls/218#issuecomment-10187. It reported model gpt-6-sol, thinking medium, provider openai-compatible, Terva 0.138.2, model check verified. None of those values is in this repository any more, so they reached the action from the terva-sh organization variables.

Outcome: clean, no findings at any severity, so there is nothing to dispose of. terva-review/code is success on the head. ci / Lint and Test passed on the same head.

The ticket commit after the reviewed head is bookkeeping only and was not re-reviewed. Merge is not authorized by this work; close the ticket when the PR merges.
