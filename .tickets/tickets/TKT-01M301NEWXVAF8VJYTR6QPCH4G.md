---
schema: 2
id: TKT-01M301NEWXVAF8VJYTR6QPCH4G
title: Widen a pen's rule from requiredLabels to a match record
type: task
status: in-progress
status_reason: null
priority: normal
due_on: null
labels:
  - area/format
  - area/cli
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
claim:
  actor: agent:claude/t3code-a6d0ff31
  branch: t3code/match-record
  worktree: /home/sothr/.t3/worktrees/git-ticket/t3code-a6d0ff31-match
  commit: 23ee2ac82fc86cc6654d07415300479211d2d757
  claimed_at: 2026-09-20T18:36:12Z
  expires_at: null
archive: null
created_at: 2026-09-20T18:36:12Z
updated_at: 2026-09-20T18:42:14Z
created_by:
  id: agent:claude/t3code-a6d0ff31
  name: ""
updated_by:
  id: agent:claude/t3code-a6d0ff31
  name: ""
extensions: {}
---

## Description

Schema 4 of the canvas board file, per plan 12.10 and the canvas repository's TKT-01M2Y91C17YTE0W0P3RBHTF50Y, which holds the design: a pen's requiredLabels becomes a match record with labels, status, type and parent, every field optional, an absent field matching everything, present fields conjoined, and a list within a field a disjunction. Schema 3 keeps opening and requiredLabels reads as match.labels; a write renders schema 4. layout.Route reads every field, git ticket canvas explain reports which fields a candidate failed, canvas show prints the rule, and pen add takes --status, --type and --parent. The canvas normaliser and resolver follow in that repository once this is released. A minor under 12.4: a new schema the reader also understands.

## Acceptance criteria

- [ ] A schema 4 board carries match with labels, status, type and parent, every field optional, and a schema 3 board opens with requiredLabels read as match.labels
- [ ] layout.Route matches on every field, and Candidate says which fields a ticket failed and with what
- [ ] git ticket canvas explain names the failed fields, show and pens print the whole rule, and pen add takes --status, --type and --parent
- [ ] A write renders schema 4, and check --fix rewrites a schema 3 board to it as layout_not_canonical
- [ ] Plan 10.10 and 12.10 describe the record and the envelope fields

## Implementation plan

Source-inspected plan, in commit order.

1. docs/plan.md first, since the plan is the design of record. 12.10 gains the match record: the four fields, what each means, that labels conjoins while status, type and parent disjoin, that schema 3 reads requiredLabels as match.labels and a write renders 4. 10.10 rewrites both JSON examples (pen carries match; candidate carries match, missingLabels, failed) and the outcome list becomes winner, no-match, later-rule. 11 changes the label_unknown field name to pens.ID.match.labels. 12.4 records the break. 12.1's pen add line gains --status, --type, --parent. README's canvas lines stop implying labels are the whole rule.

2. layout. pens.go: Pen.RequiredLabels becomes Match Match with yaml/json keys labels, status, type, parent. YAML decode of a pen accepts exactly one of match or requiredLabels, because UnmarshalYAML cannot see the file's schema; Parse then enforces which one the schema allows, the way it already enforces that routing needs schema 3. The JSON decoder takes match only, since the canvas server's whole-Routing replacement is the only JSON writer. validateRouting validates every field, parent through internal/idgrammar.Valid the way a frame member is, and refuses a pen whose match is empty in all four. canonicalRouting deduplicates each field. renderRouting emits match: {...} with empty fields omitted. layout.go: Schema = 4.

3. layout/resolve.go. RuleTicket gains Status, Type and Parent. The package-level Match function cannot survive beside a type of that name, so it becomes the method Match.Failures, returning the missing labels and the failed field names. Candidate carries Match, MissingLabels and Failed; Outcome loses missing-labels and gains no-match. check.go's LabelUnknown field becomes pens.ID.match.labels.

4. cli. canvas.go prints the whole rule in words on show, pens and explain, and names each failed field with what the ticket had and what the rule wanted. canvasPenJSON carries match. summarize and explain pass the ticket's status, type and parent into RuleTicket. canvaswrite.go: pen add gains repeatable --status, --type and --parent, the parents resolved through the store so a prefix works, and needs at least one of the four rather than at least one --label.

5. Fixtures and tests. layout-label-unknown and layout-ticket-missing re-render at schema 4 through check --fix; layout-not-canonical stays schema 2 and layout-invalid stays unparseable. No new store fixture: testdata/README.md couples the corpus to the codes of section 11 and this change adds no code. Tests: the resolve table gains a case per field, a disjunction, a conjunction across fields, and a schema 3 board still routing by its labels; cli/canvas_test.go's board becomes schema 4 with a pen matching on status, and one schema 3 test stays to prove the legacy read.

The gate is just ci, read by exit code.
