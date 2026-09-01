---
schema: 1
id: TKT-01M1FCJ8YGZN9MG9STWBSX308J
title: Decide how much of the git ticket CLI terva ticket mirrors
type: spike
status: done
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
updated_at: 2026-09-01T21:29:55Z
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

## Notes

**agent:terva/mieli** at 2026-09-01T21:29:55Z

All three options in the description were wrong, and the reason is a constraint none of them named: internal/cli could not be imported. Go's internal rule blocks any package outside github.com/terva-sh/git-ticket, so a host had no way to reach the command surface. That is what made mirroring look expensive. Both mirror options meant hand-writing flag parsing, rendering, and error mapping over the ticket package, which is exactly the second parser 12.1 exists to prevent, and it drifts from the day it is written.

**agent:terva/mieli** at 2026-09-01T21:29:55Z

Delegating to the git-ticket binary on PATH is dead on terva packaging rather than on taste. terva ships as a small static Go binary and installs through fpkg, so requiring a second binary breaks the install story and fails differently depending on what the user happens to have. A harness cannot depend on a tool it does not ship.

**agent:terva/mieli** at 2026-09-01T21:29:55Z

The fourth option costs almost nothing because the CLI layer was already shaped for it. Run(args []string, env Env) int is the only entry point and Env is the only configuration, and cmd/git-ticket/main.go is twenty lines of process plumbing around that one call. go doc confirms the package exports exactly two identifiers. Env was already fully injectable, with Dir, Getenv, Stdout, Stderr and Now, because the tests drive Run through it. Exporting published two names, not a surface.

## Summary

Mirror everything, by exporting the command surface rather than reimplementing it. internal/cli is now cli, so terva ticket is one call to cli.Run and gets all 24 commands, both output modes, and the exit statuses of 10.2 with zero drift: a command added here appears in terva on the next module bump with no terva change. Plan 12.2 records the package, its two-identifier API, and why it is exported. The split that matters: terva ticket uses cli.Run for rendered text and an exit status, and the ticket_* tools use the ticket package directly for structured values. Proved with a module outside the repository that drove ready, --json show, and a missing ticket through cli.Run, with the store discovered from an injected Dir.
