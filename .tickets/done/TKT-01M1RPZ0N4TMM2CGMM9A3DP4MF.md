---
schema: 1
id: TKT-01M1RPZ0N4TMM2CGMM9A3DP4MF
title: Navigate between linked tickets in the TUI detail view
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
  branch: feat/tui-links
  worktree: /home/sothr/workspace/git.local.sothr.com/terva-sh/git-ticket
  commit: 3eb0ca54338b024ae7135ada334f0f591fc5db6d
  claimed_at: 2026-09-05T16:46:52Z
  expires_at: null
archive: null
created_at: 2026-09-05T11:58:48Z
updated_at: 2026-09-05T16:50:15Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Filed 2026-09-05 from a user ask. The goal in one line: see and
navigate between linked tickets without leaving the TUI.

Today the detail view is a dead end. An epic's body says nothing about
its children, because a child points up via parent: and nothing points
down; the only way to walk the family is to close the view, filter the
list, and guess. A child names its parent in frontmatter, but the view
renders that as text, not as somewhere you can go.

The shape: the detail view gains a linked-tickets section, assembled
from the edges the store already knows. For an epic, the children,
meaning every ticket whose parent is this one, each with its status,
type, and title, the columns the list view teaches. For any ticket, its
parent. Dependencies and dependents are the same kind of edge and the
deps command already computes both directions, so they belong in the
section too, though they can be a second slice if the first cut wants
to stay small. Each entry is openable: the selection moves into the
section, enter replaces the view with the selected ticket's detail, and
esc walks back. Navigation is a stack, so diving epic to child to
dependency and unwinding step by step behaves the way a reader expects,
and the list view is the floor of the stack.

Two library facts shape the implementation. Children are found by
scanning for parent. Nothing queryable indexes them: epics.md is a
generated view for a file browser, written by check --fix and never
read back by the tool, so the view assembles children from the
snapshot it loads. And that snapshot must be Filter{All: true} for this
view, because an epic's done children are exactly what a person checks
an epic for, and the open-work default would silently hide them.

If TKT-01M1R4B5K2 (Decide how prefixed ID tracks subdivide a store)
ships its origin: field, origin joins the section as one more edge, in
both directions. Nothing here waits on that: parent, children, and
dependencies are worth the feature on their own.

The conventions that bind it: every new binding appears in the detail
footer or the ? help page, and the vt10x harness drives the navigation
in tests, stack and all.

### Open questions

Whether dependencies and dependents land in the first cut or follow.

What the keys are. Enter-on-selection needs the selection to reach the
section first, which wants either a focus toggle or the viewport
cursor extending into section rows. The form and status picker both
solve focus already; steal from whichever fits.

## Acceptance criteria

- [x] an epic's detail lists its children with status, type, and title, and each can be opened
- [x] a ticket's detail can open its parent
- [x] navigation is a stack: esc returns to the ticket you came from, and the list view is the floor
- [x] the linked-tickets snapshot uses Filter{All: true}, so done children of an epic appear
- [x] every new binding is named in the detail footer or the ? help page

## Notes

**agent:terva/mieli** at 2026-09-05T16:50:15Z

Built and all five criteria are ticked. The two open questions were
settled with the user before the branch: all four edge kinds in the
first cut, and the modal picker on t rather than a focusable section,
stealing the status picker's shape as the draft suggested.

The mechanics. Actions gains Links, and storeLinks answers from a
Filter{All: true} snapshot, per the fourth criterion: parent by the
frontmatter field, children by scanning for it, needs from the
ticket's own dependencies, needed-by by scanning everyone else's. A
dependency naming a missing ticket is skipped rather than invented,
because dependency_missing is check's finding to report, not the
picker's. Each picker row reads role, status, type, title, the
columns the list view teaches.

The App's one-level detail field became a real stack, which is the
generalization its own comment said would wait for a feature that
earned it. Enter from the picker pushes, Esc pops one level at a
time, and the list is the floor. TestLinkPickerOpensAndDivesTheStack
walks the whole dive and unwind.

The t hint cost the detail footer its g/G entry: the 60-column budget
was full, so g/G moved to a new Detail section on the ? page, which
now also explains the stack. Every binding remains hinted somewhere,
which is the convention's actual requirement.

One bonus finding: the test fixture sets a parent on a created-done
ticket through SetParent, exercising the arrive-done path from
v0.9.0 in a context it was not built for, and it worked first try.

Origin stays future work, as filed: if TKT-01M1R4B5K2 ships its
origin: field, it joins the picker as one more role.

## Summary

Shipped. t in the TUI detail view opens a picker of the ticket's
linked tickets, all four edge kinds labeled from the viewed ticket's
side: parent, child, needs, needed by. Enter dives into the chosen
ticket's detail and Esc unwinds one level at a time, with the list as
the floor: the App's detail view became a real stack, the
generalization its own comment was waiting for. The snapshot is
Filter{All: true}, so an epic's done children appear. The g/G hint
moved to the ? page's new Detail section to make room for t in the
60-column footer. Origin joins as one more role if TKT-01M1R4B5K2
ships it.
