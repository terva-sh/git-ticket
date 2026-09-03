---
schema: 1
id: TKT-01M1FAFS0ZG1DJPWSM9PR8ZD6Q
title: Scrub internal hostnames before pushing to the public mirror
type: chore
status: done
status_reason: null
priority: high
labels:
  - release
assignees: []
milestone: null
parent: null
dependencies: []
references:
  - ref: proposal:git-ticket
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-01T20:27:36Z
updated_at: 2026-09-01T20:37:10Z
created_by:
  id: human:sothr
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

The public mirror exists at github.com/terva-sh/git-ticket, but this tree is not ready to be pushed there verbatim. Ten mentions of the internal forge and registry hostnames are spread across AGENTS.md, .forgejo/workflows/ci.yml, docs/plan.md, and three ticket bodies under .tickets/. terva already treats this as a release gate rather than a habit: scripts/release.sh check-scrub builds the public tree and greps it for internal hostnames and remotes, and .forgejo is the first entry in its release EXCLUDES so the workflow never reaches the public tree at all. Decide the approach for this repository before the first mirror push, not after. An overlay like terva's is one option. Writing the docs so they never name the host is another, and is cheaper here because this repository has far less to hide: the CI conventions can say 'the internal Forgejo' and 'the internal registry' and lose nothing a reader needs.

## Notes

**agent:terva/mieli** at 2026-09-01T20:37:10Z

Scrubbed by rewriting the prose, not by building a filtered tree. AGENTS.md, the two CI comments, and four lines across three ticket bodies now say 'the internal Forgejo' and 'the internal registry'. The status-API recipe in AGENTS.md derives the host from git remote get-url origin instead of writing it down, which is also more portable than what it replaced.

**agent:terva/mieli** at 2026-09-01T20:37:10Z

One hostname cannot be reworded: the container image in .forgejo/workflows/ci.yml is a working reference, not prose. Excluding that directory was considered and rejected. It would need a filtered tree and synthesized commits, which costs the mirror its shared history and tag objects, and terva's precedent does not transfer: release.sh is 1873 lines and its EXCLUDES hides .github, .claude, .terva, AGENTS.md, docs/plans, docs/ideas and docs/proposals. This repository has one line. The image ships.

## Summary

Done. The mirror is a plain git push with the same commits and tags, because the prose no longer names the internal hosts. git grep for the origin host outside .forgejo returns nothing. The single deliberate exposure is the golang:1.25-alpine image reference in the CI workflow, which is recorded in the file itself and in AGENTS.md rather than left to be rediscovered.
