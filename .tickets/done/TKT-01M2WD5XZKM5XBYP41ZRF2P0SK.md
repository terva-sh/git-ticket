---
schema: 2
id: TKT-01M2WD5XZKM5XBYP41ZRF2P0SK
title: Add the layout package for the canvas board file
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - area/format
  - area/integration
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-19T08:40:28Z
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

`.tickets/canvas/*.yml` is the board file git-ticket-canvas writes inside a directory this tool owns: card positions, frames, and routing rules, one line per record, sorted by ID. Its format lives in that repository's `internal/layout`, so `git ticket check` validates every other file under `.tickets/` and cannot see this one, and no `git ticket` command can read or write a board at all.

Move the package here as `layout`, a top-level package beside `ticket`, and have the canvas import it. No behaviour changes: the same bytes are read and written, the same errors are returned. The package brings its own atomic writer and mutex and does not join the ticket store's lock, because that would be a behaviour change and belongs to whichever ticket first needs it.

This is the first step of the sequence recorded on the canvas side under TKT-01M2ND1RH33T89QZ7JBA0YC1AZ (Move the layout schema into git-ticket) and its epic TKT-01M2ND0S8N5Y8V0HQFRCBKMXE3 (Let an agent organize a board), which follow docs/board-organization-design-v1.md in that repository. `check` learning the layout file and `git ticket canvas` commands come after, in their own tickets.

A new exported package is a minor version: tag v0.20.0 once merged, so the canvas can drop its copy.

## Acceptance criteria

- [x] layout is an importable package at github.com/terva-sh/git-ticket/layout with the tests it came with passing under just ci
- [x] The package renders the same bytes as internal/layout at git-ticket-canvas 9d1f6ed2122300747dc5ad10075830f5927f9bb6 for the same board
- [x] docs/plan.md section 12 names the layout file, who defines it, and that check does not validate it yet

## Implementation plan

Copy the six files of internal/layout from git-ticket-canvas at 9d1f6ed2122300747dc5ad10075830f5927f9bb6 into layout/ unchanged except for a package-comment line naming that origin; the package already imports github.com/terva-sh/git-ticket/ticket and gopkg.in/yaml.v3 and nothing from the canvas module, so no import changes. Add a 12.10 entry to docs/plan.md under Interfaces. Run just ci. Open a PR on Forgejo. Tagging v0.20.0 and pushing it to GitHub is left to the maintainer; the canvas side (TKT-01M2ND1RH33T89QZ7JBA0YC1AZ there) bumps to that tag and deletes its copy.

## Notes

**agent:claude/t3code-a6d0ff31** at 2026-09-19T08:41:45Z

Evidence: diff -r of the six files against git-ticket-canvas internal/layout at 0021a8d (unchanged since 9d1f6ed) shows only the seven added package-comment lines, so the renderer is the same code and produces the same bytes by construction. just ci passed: gofmt clean, go vet clean, go test -race across cli, layout, ticket, tui, tui/view, and check --fix --dry-run --strict found no problems. Not run: the canvas against this package, which is the canvas ticket's verification once v0.20.0 exists.

**agent:claude/t3code-a6d0ff31** at 2026-09-19T16:03:54Z

Terva review 32 on PR 208 (head ef11378209e9d680643565f82c8213fc682fc1b1): Boards returns filenames that Load refuses. Real and pre-existing in the canvas copy; deferred rather than fixed here because this ticket's second criterion is byte-identical behaviour with the source. Filed as TKT-01M2X6HV in this store.

## Summary

Landed in PR 208, merged as c70a6b9 and released as v0.20.0 on both forges. The package is the canvas's internal/layout at 9d1f6ed plus a package-comment origin note; plan.md 12.10 records it. The canvas imports it from v0.20.0 in its PR 14 with byte-identical board files proven there. Review 32's finding, Boards listing files Load refuses, is deferred to TKT-01M2X6HVX8N3H93JPWS43ZWCVF.
