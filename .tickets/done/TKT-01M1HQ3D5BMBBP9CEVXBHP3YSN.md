---
schema: 1
id: TKT-01M1HQ3D5BMBBP9CEVXBHP3YSN
title: Decide whether the renderer canonicalizes body section text
type: spike
status: done
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
updated_at: 2026-09-02T20:00:38Z
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

## Implementation plan

1. Add Body.normalize in parse.go, mirroring parseBody field for field.
2. Call it in writeTicket, the one funnel every write already passes through.
3. Remove the per-writer TrimSpace on section text, keeping the guards and
   keeping removeChecklistItem's own trim.
4. Record the rule in plan 5.3 and the settlement in section 15.

## Summary

Settled: the renderer does not canonicalize. The normalization moved to
writeTicket, and the normalizer changed from TrimSpace to trimBlankLines.

The ticket framed this as render-versus-each-writer. Both were wrong. Render
cannot be the place, because writeTicket hands back the same Ticket it
rendered, callers read that struct and the CLI serializes it, so a renderer
that normalized alone would leave the struct and the file disagreeing about
the ticket's own text. That is a worse bug than the one it fixes, because byte
instability shows up in a diff and this would not. Per-writer trimming holds
only as long as every future writer remembers, which was the ticket's own
complaint.

writeTicket is the third option and the right one: Apply and Create both
funnel through it, so one call settles every writer present and future, and
the struct stays honest because it is normalized before it is returned.

Settling it turned up a second bug the per-writer fix had introduced. Those
writers used TrimSpace, which is stronger than the round trip needs. A probe
over the real renderer showed only blank lines at the edges of a section
destabilize it: an indented first line, leading spaces, and trailing spaces are
all stable. So TrimSpace was silently reindenting any section opening with an
indented code block, for no round-trip reason. trimBlankLines is the exact
fixed point of parse, and a test now holds that over the whole roundtrip
corpus.

One trim did not move. removeChecklistItem still trims its own result, because
normalize only reaches the edges of a section and checklistOps batches a
removal before an add. Padding left mid-section by the removal would be
stranded where nothing trims it.
