---
schema: 1
id: TKT-01M1HPCJG8JN3WJNV6R086FNWY
title: Make the instructions block refreshable in place
type: task
status: ready
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
updated_at: 2026-09-02T18:34:29Z
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
