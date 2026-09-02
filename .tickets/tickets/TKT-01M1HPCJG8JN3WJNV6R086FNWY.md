---
schema: 1
id: TKT-01M1HPCJG8JN3WJNV6R086FNWY
title: Make the instructions block refreshable in place
type: task
status: done
status_reason: null
priority: normal
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
references:
  - ref: review:backlog-md
    path: docs/review-backlog-md.md
claim: null
archive: null
created_at: 2026-09-02T18:34:03Z
updated_at: 2026-09-02T19:34:14Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

init --instructions writes the agent workflow block to AGENTS.md at the repository root and refuses when that file already exists, naming git ticket instructions so the user can paste it themselves. So a store initialized last month is stuck with last month prose, and shipping an instructions change means asking every user to re-paste it by hand.

Backlog.md solves this with agents --update-instructions, which rewrites its block in place and preserves everything around it. See section D of docs/review-backlog-md.md.

We should copy the mechanism and not the scope. They write CLAUDE.md, AGENTS.md, GEMINI.md, and .github/copilot-instructions.md, which is their vendor list rather than ours. AGENTS.md is the file this project already writes and the one terva reads.

Scope: wrap the block emitted by cli/instructions.md in stable begin and end markers, and add a command that replaces what sits between them and leaves the rest of the file untouched. A file with no markers is appended to rather than refused, which is the case init --instructions handles badly today.

Open: whether the marker is an HTML comment. Section F of the review notes that Backlog.md parses its own checklists out of AC:BEGIN and DOD:END comments, which breaks when somebody deletes a comment they had no reason to keep. The risk is far smaller here, because losing the markers costs a refresh rather than a checklist, but the shape is the same and is worth a moment.

## Implementation plan

1. Put begin and end markers in cli/instructions.md itself, so stdout, the
   JSON text, and the file all carry them.
2. Add spliceInstructions: replace between markers, append when there are
   none, refuse anything else.
3. Add instructions --write, and make init --instructions use the same write
   so an existing AGENTS.md is appended to rather than refused.
4. Record the marker contract in plan 12.1 and 10.5.

## Summary

Added markers and instructions --write.

The open question is settled as an HTML comment. It renders as nothing, so it
does not clutter a file a person reads and edits. Section F of the review
worries that Backlog.md parses its checklists out of markers, so losing one
loses data. These bound a block the tool regenerates, so losing them costs a
refresh and nothing else. The shapes are the same and the stakes are not.

The markers live in instructions.md rather than being added by the writer, so
every way the block leaves the binary carries them. That is what makes a block
somebody pasted by hand refreshable later, which is the whole point.

Answering it raised the question it was hiding: what to do with a file that
carries one marker and not its partner. Refuse. There is no honest reading of
where the block ends, and the wrong guess deletes prose somebody wrote. A
refusal costs a person one edit. The same goes for a reversed pair and for a
second copy of either.

init --instructions no longer refuses an existing AGENTS.md, which is the case
the ticket called badly handled. It appends, and the file is refreshable from
then on. It still checks before creating the store, so the one remaining
refusal leaves nothing half-built. A file already current is not rewritten, so
a no-op stays out of a diff.
