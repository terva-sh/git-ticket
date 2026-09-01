---
schema: 1
id: TKT-01M1FM51CNHG3A7ETBA99HRKHE
title: Enforce the read-only Git promise with a test
type: chore
status: done
status_reason: null
priority: normal
labels:
  - policy
assignees: []
milestone: null
parent: null
dependencies: []
references: []
claim: null
archive: null
created_at: 2026-09-01T23:16:30Z
updated_at: 2026-09-01T23:16:38Z
created_by:
  id: human:sothr
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

## Notes

**agent:terva/mieli** at 2026-09-01T23:16:30Z

AGENTS.md and acceptance criterion 'no command performs a remote operation as a side effect' both promised the library runs no writing Git command. Nothing enforced it: runGit carried a doc comment and cli/readGit did not even have that. Adding git fetch would have passed CI green.

**agent:terva/mieli** at 2026-09-01T23:16:30Z

Weight went up at v0.2.0. A host embedding the cli package ships our Git calls inside its own binary, running in the user's repository, so our promise became theirs.

## Summary

Plan 7.4 now enumerates the two Git commands this code runs, rev-parse and symbolic-ref, with what reads each. TestGitCommandsAreReadOnly in ticket/gitpolicy_test.go parses the module's non-test Go and asserts three things: no exec.Command names a binary other than git, every one sits in runGit or readGit, and every helper call names a command from the plan table. The allowlist is read from the plan, so the two cannot drift, following TestCorpusCoversEveryPlanCode. Verified red on seven mutations in a throwaway copy before being trusted.
