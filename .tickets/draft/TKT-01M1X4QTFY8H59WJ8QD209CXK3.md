---
schema: 2
id: TKT-01M1X4QTFY8H59WJ8QD209CXK3
title: Have init write an eol=lf .gitattributes line for Windows stores
type: bug
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - format
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-07T05:16:30Z
updated_at: 2026-09-07T05:16:30Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

On Windows, git converts text files to CRLF on checkout by default. Plan 5.3 requires a ticket file to carry LF line endings, and `parse` enforces it: a file whose first line is `---\r` is rejected with `parse_error: file does not start with a --- frontmatter fence`. So a user who clones a repository containing a `.tickets/` store on Windows gets a store where every ticket fails to read, and `git ticket list` answers with nothing rather than with an error that explains itself.

This is not hypothetical. It is what took the Windows CI lane red on its first run, GitHub Actions run 34085691784: about 30 fixtures converted, and `corpus_test.go:51` named the cause as "CRLF line endings, 5.3 requires LF". This repository fixed its own checkout with `* text=auto eol=lf` in `.gitattributes`, which protects this repository and no other.

### The decision

`git ticket init` writes an `eol=lf` line into `.gitattributes` beside the merge-driver line, so a store is protected at creation. That is the user's ruling, taken on 2026-09-07 alongside the choice to ship v0.14.2 with the lock fix alone.

`init` already writes that file. `ensureMergeAttribute` in `cli/mergedriver.go` appends `.tickets/**/*.md merge=gitticket` when it is absent and leaves everything else in the file untouched, so a second line follows the same pattern and the same append-never-rewrite rule.

The line should be scoped to the store rather than global, because the file belongs to the repository and not to this tool. `.tickets/**/*.md text eol=lf` matches what `mergeAttrPattern` already builds.

### What this does not solve

`init` cannot help a store that already exists, and it cannot help a clone made before the line was written. The user considered making `parse` tolerate CRLF as well and chose not to, so a converted store stays unreadable until somebody adds the attribute and re-checks-out. If that turns out to bite in practice, reopening the tolerance question is the response, and this ticket is where the reasoning lives.

### Plan

The plan changes with the implementation and not before, per AGENTS.md: the format is authoritative and a code change that needs it moves it in the same commit. Section 7.5 covers the merge driver's `.gitattributes` entry and is the natural home for the second line.

## Acceptance criteria

- [ ] git ticket init writes a store-scoped eol=lf line into .gitattributes beside the merge-driver line, appending rather than rewriting
- [ ] A store created by init and cloned on Windows parses, proven on the Windows CI lane rather than inferred
- [ ] docs/plan.md section 7.5 describes the second line, changed in the same commit as the code
