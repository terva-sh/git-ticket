---
schema: 2
id: TKT-01M2X6HVX8N3H93JPWS43ZWCVF
title: Boards lists layout files that Load refuses
type: bug
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - area/format
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-19T16:03:54Z
updated_at: 2026-09-19T23:28:20Z
created_by:
  id: agent:claude/t3code-a6d0ff31
  name: ""
updated_by:
  id: agent:claude/t3code-a6d0ff31
  name: ""
extensions: {}
---

## Description

layout.Store.Boards returns every non-directory entry under .tickets/canvas ending in .yml with the suffix stripped, without applying boardNameOK, while Store.path applies it and Load refuses anything else. A file such as "with space.yml" is listed as a board and then fails to open with "invalid board name". The canvas's board picker reads Boards, so a stray file becomes a board a person can select and cannot open. Found by Terva review 32 on git-ticket PR 208 while the package was being moved in unchanged; the fix was kept out of that PR to preserve byte-identical behaviour with the canvas copy. Fix: filter names through boardNameOK in Boards, and add a test with a malformed filename beside a valid board. Decide whether to log or silently skip the file; the canvas lists an unopenable store with its reason rather than hiding it, and the same rule probably applies here.

## Implementation plan

Filter each directory entry's name through boardNameOK in Boards before listing it, so the list holds only names Load accepts. Skip rather than report: a file the package would refuse to write is not a board, and a diagnostic for a stray file belongs to check once it learns the layout file (TKT-01M2ND1RMSJ1KAEZM7HZX5DEHR in the canvas store), which is also where the canvas's rule of naming an unopenable thing with its reason lands. Changing the signature to carry reasons now would be the first break of a package published yesterday. One test: a directory holding default.yml, ok-2.yml, 'with space.yml', and a leftover .default.1.tmp lists exactly default and ok-2.

## Notes

**agent:claude/t3code-a6d0ff31** at 2026-09-19T23:28:20Z

Fix and test in layout/layout.go and layout_test.go. just ci passed. Skip chosen over report, per the plan; the reason is in the code comment and on the ticket.

## Summary

Boards now filters each name through boardNameOK, so it lists only boards Load accepts; a test holds two valid boards beside a spaced name, a dotted name, and a temp file and checks the list opens. Stray files are skipped silently; reporting them is check's once it reads the layout directory.
