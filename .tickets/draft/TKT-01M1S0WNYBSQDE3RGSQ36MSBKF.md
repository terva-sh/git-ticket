---
schema: 1
id: TKT-01M1S0WNYBSQDE3RGSQ36MSBKF
title: Sort the TUI list by priority, status, and time
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-05T14:52:17Z
updated_at: 2026-09-05T14:52:17Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Filed 2026-09-05 from a user ask: a sort mode for `git ticket ui`, at
least priority, status, and creation or update time, so the list can
answer "what matters most" and "what moved last" without leaving the
terminal.

What exists, so the draft starts from the tree. `list --sort` takes
`id`, `due_on`, and `priority`, applied in the CLI after the store
answers, with `id` the default because a ULID sorts by creation time
and the store already answers in that order. The TUI has no sort at
all: StoreLister returns the store's ID order and the list shows it.
So creation time is already every view's default, and the ask reduces
to giving the TUI the orders the CLI has, plus two orders nobody has.

The two new orders go into the shared vocabulary first, so `list
--sort` gains them in the same change and the two surfaces keep
teaching one rule, the way the filter grammar mirrors ticket.Filter.
Update time is `updated_at`, newest first, because "what moved last"
is a recency question. A status order needs a defined ranking before
it can sort, and the lifecycle already defines one: draft, ready,
in-progress, blocked, review, done, archived, the order 6.1 lists.
Whether that ranking reads best ascending or with the working set
first is a decision for the build, not this filing.

### Open questions

The UI mechanism. A cycling key (`o` for order) is one keystroke and
discoverable in the footer; a filter-line token (`sort:priority`)
mirrors how the list already takes instructions and costs no new key.
Whichever wins, the active order has to be visible somewhere the eye
already rests, header or footer, because an invisible sort mode reads
as a broken list.

Whether `due_on` rides along in the TUI for free once the vocabulary
is shared. Probably yes, and the deadline-driven day wants it.

### Not this ticket

A hand-set order. TKT-01M1HPCJH (Decide whether the format stores a
hand-set order) draws the line: display sort is a flag with no format
change behind it, which is all of this ticket, and a persisted ordinal
is that draft's own question. Building this fires nothing there.

### Trigger

None. Startable feature work, parked as a draft until promoted.

## Acceptance criteria

- [ ] the TUI list can sort by priority, status, updated time, and creation time, with creation staying the default
- [ ] the new orders land in the shared vocabulary, so list --sort gains updated_at and status in the same change
- [ ] the active order is visible in the header or footer, and the mechanism's keys or tokens are hinted
- [ ] no format change: building this fires nothing on TKT-01M1HPCJH
