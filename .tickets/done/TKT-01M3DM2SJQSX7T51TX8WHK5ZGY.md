---
schema: 2
id: TKT-01M3DM2SJQSX7T51TX8WHK5ZGY
title: Show resolved reference targets across readers
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - area/cli
assignees: []
milestone: null
parent: null
origin: null
dependencies:
  - TKT-01M3DM2SGK9X1S1FN7RKPF02A1
blocks_on: none
references:
  - ref: ticket:TKT-01M3CV8R9J3QXQSN07B9VAGV7V
    path: null
claim: null
archive: null
created_at: 2026-09-26T01:08:11Z
updated_at: 2026-09-26T03:09:03Z
created_by:
  id: agent:codex/reference-design
  name: ""
updated_by:
  id: agent:codex/landing
  name: ""
extensions: {}
---

## Description

Expose resolved reference targets without changing stored ref or path bytes, per plan 5.5. Show, UI, and JSON should display an HTTPS destination or a locally bound foreign ticket where one is available, and remain useful when the foreign checkout is absent. refs keeps exact namespace and identifier lookup; add a discoverable resolution path without changing its current matching semantics. Exercise a PR reference, a foreign ticket, and an undeclared legacy reference end to end.

## Acceptance criteria

- [x] Show, UI, and JSON expose resolved target without changing stored ref or path
- [x] Refs lookup semantics remain unchanged and a resolution query is discoverable
- [x] PR, foreign ticket, absent checkout, and undeclared legacy examples are exercised

## Implementation plan

Decorate existing references at read time with the registry resolver, preserving stored ref and path bytes. Show human output and existing ticket JSON gain optional targets. Add refs --resolve for matching references without changing refs lookup. The TUI detail renders targets and offers an explicit reference picker to open a safe local target or portable URL through a host action. Exercise declared URL, foreign-ticket, absent checkout and undeclared legacy cases.

## Notes

**agent:codex/reference-readers** at 2026-09-26T02:51:44Z

Implemented read-time target decoration in show, JSON ticket reference objects and refs --resolve; stored ref and path bytes are unchanged. ResolveRefs uses the existing refs matcher, so namespace case handling and exact identifier matching stay aligned. The TUI detail shows refs and both portable and local targets; r opens a picker and Enter launches only the selected target, choosing an existing local path before HTTPS. An explicit picker was chosen over terminal-only hyperlinks because local files and terminals without hyperlink support still need an open path. The platform opener is isolated to one CLI helper, uses argv without a shell, and joins plan 7.4’s narrow non-Git exec exceptions. The detail footer replaces the Ctrl+C hint with r refs; Ctrl+C remains on the help page to stay inside 60 columns. Full just ci passed. A built-binary run proved human show, refs --resolve, JSON targets and check --strict; tests cover PR URL, foreign ticket with and without checkout, undeclared legacy ref, local-path precedence, and the TUI picker.

**agent:codex/reference-readers** at 2026-09-26T02:55:45Z

PR #228 is open against feat/reference-registry (PR #226) at implementation head 88d8b52b124f71b937d7b24a2ec9f60006ceec55. Local full just ci and the built-binary reader run passed before opening. The ticket is in review; requesting targeted review and PR CI. The PRs remain unmerged pending explicit authorization.

**agent:codex/reference-readers** at 2026-09-26T02:57:50Z

PR #228 review 764 (head ce1105c, base 3567f31; Actions run #613, id 11698) raised a medium finding: the TUI opener returned success on process start and released the child without waiting. Accepted. The opener now waits and propagates the desktop command's exit error, so the TUI reports success only when that command succeeds. A Linux regression test installs a failing fake xdg-open and checks error propagation. Full just ci passed. Requesting fresh targeted review.

**agent:codex/reference-readers** at 2026-09-26T02:59:31Z

PR #228 review 765 (head 91ac609, base 3567f31; Actions run #617, id 11702) confirmed review 764's opener finding resolved and raised a medium finding: refs --resolve ignored --ids. Accepted. The resolved human listing now uses the same storeAbbreviations helper as ordinary refs listings; JSON still carries full IDs. A regression test checks default abbreviated and explicit full output. Full just ci passed. Requesting fresh targeted review.

**agent:codex/reference-readers** at 2026-09-26T03:01:34Z

PR #228 targeted review is clean on head 3015d4f99b31ea4a31aea6e62907a74a9e978dc2, base 3567f3197260cb0dc9e413b0e990e56e67e7e8ad (request resolve-ids-review; Actions run #619, id 11708; clean summary comment 13943). Review 765's --ids finding is resolved. Full local just ci passed on the reviewed code. Recording this ticket-only result and carrying the review status to the bookkeeping head; review success is not merge authorization.

## Summary

Resolved targets in show, JSON, refs, and the TUI landed through PR #228. The opener and ID display review findings were fixed; full CI and targeted review passed.
