---
schema: 1
id: TKT-01M1FAFS0ZG1DJPWSM9PR8ZD6Q
title: Scrub internal hostnames before pushing to the public mirror
type: chore
status: ready
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
updated_at: 2026-09-01T20:27:36Z
created_by:
  id: human:sothr
  name: ""
updated_by:
  id: human:sothr
  name: ""
extensions: {}
---

## Description

The public mirror exists at github.com/terva-sh/git-ticket, but this tree is not ready to be pushed there verbatim. Five tracked files name the internal infrastructure: AGENTS.md (4 hits), .forgejo/workflows/ci.yml (2), docs/plan.md (1), and two ticket bodies under .tickets/ (3). They name git.local.sothr.com and container.local.sothr.com. terva already treats this as a release gate rather than a habit: scripts/release.sh check-scrub builds the public tree and greps it for internal hostnames and remotes, and .forgejo is the first entry in its release EXCLUDES so the workflow never reaches the public tree at all. Decide the approach for this repository before the first mirror push, not after. An overlay like terva's is one option. Writing the docs so they never name the host is another, and is cheaper here because this repository has far less to hide: the CI conventions could say 'the internal Forgejo' and 'the local registry mirror' and lose nothing a reader needs.
