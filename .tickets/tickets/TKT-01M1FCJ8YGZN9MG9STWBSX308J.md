---
schema: 1
id: TKT-01M1FCJ8YGZN9MG9STWBSX308J
title: Decide how much of the git ticket CLI terva ticket mirrors
type: spike
status: ready
status_reason: null
priority: high
labels:
  - integration
assignees: []
milestone: null
parent: TKT-01M1FCHH3QN04AYHZAP1M8DNQK
dependencies: []
references: []
claim: null
archive: null
created_at: 2026-09-01T21:03:55Z
updated_at: 2026-09-01T21:04:21Z
created_by:
  id: human:sothr
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

The CLI has 24 commands. Three options, none obviously right.

Mirror all 24. One vocabulary, and terva ticket is a drop-in for anyone who knows the CLI. Costs a large surface to keep in step, and every future git-ticket command becomes a terva change.

Mirror a subset. Smaller and easier to hold, but a user learns two vocabularies and has to know which one they are in.

Delegate to the git-ticket binary when it is on PATH. No duplication at all and it can never drift, but it adds a runtime dependency terva does not otherwise have, and it fails differently depending on what the user has installed.

Blocks slice 1, because the command surface is what slice 1 builds.
