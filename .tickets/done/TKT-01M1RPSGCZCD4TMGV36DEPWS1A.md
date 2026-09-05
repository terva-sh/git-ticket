---
schema: 1
id: TKT-01M1RPSGCZCD4TMGV36DEPWS1A
title: Show the type column in the TUI list view
type: task
status: done
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
claim:
  actor: agent:terva/mieli
  branch: feat/tui-type-column
  worktree: /home/sothr/workspace/git.local.sothr.com/terva-sh/git-ticket
  commit: ddd4a9ef910b5a15d377d1010a3606b4b367aefa
  claimed_at: 2026-09-05T11:55:50Z
  expires_at: null
archive: null
created_at: 2026-09-05T11:55:47Z
updated_at: 2026-09-05T11:56:41Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Filed 2026-09-05 from a user ask. The TUI list renders ID, STATUS,
PRIORITY, TITLE, so an epic and a task are indistinguishable until the
detail view opens. The store's own working set mixes epics, tasks, and
spikes, and which is which is exactly what a person scanning the list
is asking.

The fix is one more column. TYPE goes between STATUS and PRIORITY,
because that is the order the detail header already prints: `draft
spike low` is status, type, priority, and the list should not teach a
different order than the view one keystroke deeper. Width is
content-derived like the other columns, per the rule in widths(): a
fixed width is the abbreviation mistake the store's own CLI refuses to
make. The longest type today is five characters, so the cost per row
is seven columns of an 80-column line.

## Acceptance criteria

- [x] the list header and every row carry TYPE between STATUS and PRIORITY, width content-derived
- [x] the column order matches the detail header's status, type, priority
- [x] a rendering test covers the column

## Notes

**agent:terva/mieli** at 2026-09-05T11:56:41Z

Built and all three criteria are ticked.

The change is contained in tui/view/list.go: widths() gains a typ
width, minimum four for the header word, and header() and row() print
TYPE between STATUS and PRIORITY. The placement comment records the
reason: the detail header prints "draft spike low", status, type,
priority, and the list should not teach a different order than the
view one keystroke deeper.

TestListShowsTheTypeColumn in tui/view/list_test.go covers it: every
column name present, TYPE positioned between STATUS and PRIORITY by
index, and an epic row and a task row each showing their type. The
existing fixtures carry no Type, so the test builds its own two
tickets rather than widening the mk() signature every other test
shares.

No new key, so the hinting convention is not in play, and no width
worry: the longest type is five characters and the Frame clips each
row to the terminal anyway.

## Summary

Shipped. The TUI list renders a TYPE column between STATUS and
PRIORITY, matching the detail header's status, type, priority order,
with a content-derived width like every other column. An epic, a task,
and a spike are now distinguishable from the list without opening one.
TestListShowsTheTypeColumn holds the column set, the order, and the
row content.
