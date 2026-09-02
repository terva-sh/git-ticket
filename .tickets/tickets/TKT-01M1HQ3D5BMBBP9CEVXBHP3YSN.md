---
schema: 1
id: TKT-01M1HQ3D5BMBBP9CEVXBHP3YSN
title: Decide whether the renderer canonicalizes body section text
type: spike
status: draft
status_reason: null
priority: low
labels:
  - question
  - format
assignees: []
milestone: null
parent: null
dependencies: []
references: []
claim: null
archive: null
created_at: 2026-09-02T18:46:31Z
updated_at: 2026-09-02T18:46:31Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Found while adding the implementation plan mutation. See TKT-01M1HPAFH.

Parse normalizes and render does not. parse.go runs every section through trimBlankLines, so no parsed ticket carries blank lines around a section body. render.go section() writes the text verbatim. A Ticket constructed in Go with padded section text therefore renders bytes that parse back to something that renders differently, which is the round trip plan 5.3 requires.

Store.Create was the one reachable path: create --description passed the string straight into Body{Description: ...}. That is fixed by trimming in Create, and TestCreateTrimsBodySectionsItSeeds pins it. Every Set* mutation on a body section already trimmed, so the fix made Create agree with them rather than inventing a rule.

What is undecided is where the guarantee belongs. Trimming at each writer is what the code does now, and it holds only as long as every future writer remembers. A new mutation that sets a section and forgets to trim reintroduces it silently, because no test covers a writer that does not exist yet.

The alternative is for section() to trim on the way out, which makes render total rather than trusting its input. Plan 5.3 says the renderer must be a pure function of the parsed ticket and that two writers producing the same logical ticket must produce identical bytes. A ticket whose description is padded and one whose description is not are arguably the same logical ticket, and under that reading render is where the normalization belongs and the per-writer trims become redundant.

The cost is small but real. Rendering would stop being a faithful echo of the struct, so a caller that deliberately set leading whitespace would find it gone, and the round trip test would no longer be able to tell a writer that trims from one that does not.

No trigger fires today, because the reachable path is closed. This is worth settling before a second body section grows a mutation, which is when the per-writer discipline gets its first real test.
