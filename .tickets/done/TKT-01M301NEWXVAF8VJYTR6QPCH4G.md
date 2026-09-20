---
schema: 2
id: TKT-01M301NEWXVAF8VJYTR6QPCH4G
title: Widen a pen's rule from requiredLabels to a match record
type: task
status: done
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
claim: null
archive: null
created_at: 2026-09-20T18:36:12Z
updated_at: 2026-09-20T19:27:27Z
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

- [x] A schema 4 board carries match with labels, status, type and parent, every field optional, and a schema 3 board opens with requiredLabels read as match.labels
- [x] layout.Route matches on every field, and Candidate says which fields a ticket failed and with what
- [x] git ticket canvas explain names the failed fields, show and pens print the whole rule, and pen add takes --status, --type and --parent
- [x] A write renders schema 4, and check --fix rewrites a schema 3 board to it as layout_not_canonical
- [x] Plan 10.10 and 12.10 describe the record and the envelope fields

## Implementation plan

Source-inspected plan, in commit order.

1. docs/plan.md first, since the plan is the design of record. 12.10 gains the match record: the four fields, what each means, that labels conjoins while status, type and parent disjoin, that schema 3 reads requiredLabels as match.labels and a write renders 4. 10.10 rewrites both JSON examples (pen carries match; candidate carries match, missingLabels, failed) and the outcome list becomes winner, no-match, later-rule. 11 changes the label_unknown field name to pens.ID.match.labels. 12.4 records the break. 12.1's pen add line gains --status, --type, --parent. README's canvas lines stop implying labels are the whole rule.

2. layout. pens.go: Pen.RequiredLabels becomes Match Match with yaml/json keys labels, status, type, parent. YAML decode of a pen accepts exactly one of match or requiredLabels, because UnmarshalYAML cannot see the file's schema; Parse then enforces which one the schema allows, the way it already enforces that routing needs schema 3. The JSON decoder takes match only, since the canvas server's whole-Routing replacement is the only JSON writer. validateRouting validates every field, parent through internal/idgrammar.Valid the way a frame member is, and refuses a pen whose match is empty in all four. canonicalRouting deduplicates each field. renderRouting emits match: {...} with empty fields omitted. layout.go: Schema = 4.

3. layout/resolve.go. RuleTicket gains Status, Type and Parent. The package-level Match function cannot survive beside a type of that name, so it becomes the method Match.Failures, returning the missing labels and the failed field names. Candidate carries Match, MissingLabels and Failed; Outcome loses missing-labels and gains no-match. check.go's LabelUnknown field becomes pens.ID.match.labels.

4. cli. canvas.go prints the whole rule in words on show, pens and explain, and names each failed field with what the ticket had and what the rule wanted. canvasPenJSON carries match. summarize and explain pass the ticket's status, type and parent into RuleTicket. canvaswrite.go: pen add gains repeatable --status, --type and --parent, the parents resolved through the store so a prefix works, and needs at least one of the four rather than at least one --label.

5. Fixtures and tests. layout-label-unknown and layout-ticket-missing re-render at schema 4 through check --fix; layout-not-canonical stays schema 2 and layout-invalid stays unparseable. No new store fixture: testdata/README.md couples the corpus to the codes of section 11 and this change adds no code. Tests: the resolve table gains a case per field, a disjunction, a conjunction across fields, and a schema 3 board still routing by its labels; cli/canvas_test.go's board becomes schema 4 with a pen matching on status, and one schema 3 test stays to prove the legacy read.

The gate is just ci, read by exit code.

## Notes

**agent:claude/t3code-a6d0ff31** at 2026-09-20T18:55:13Z

Labels conjoin; status, type and parent disjoin. The design said every field's list is a disjunction and gave status as the example, but two other parts of the same design only work if labels conjoins: missingLabels is defined as the labels the ticket lacks, and the show example prints 'labels frontend, bug' where status prints 'ready or blocked'. Backwards compatibility settles it, since requiredLabels meant all of them and a schema 3 board has to route unchanged. The reason it is not an inconsistency is in plan 12.10: a ticket carries many labels at once, so a list of them can only mean all of them, and a ticket holds one status, one type and one parent, so a list there can only mean any of them.

**agent:claude/t3code-a6d0ff31** at 2026-09-20T18:55:13Z

layout.Match could not be both the type and the function. The design names the wire record Match and also says Match(pen, ticket) becomes a per-field check, and Go has one package namespace for both. The type keeps the name, because it is what the file and both envelopes spell, and the comparison became the method Match.Failures(RuleTicket), returning the missing labels and the failed field names. A method on Pen was not available either: Pen has a field named Match, and a field and a method cannot share a name. 12.4 records the removal of the function beside the other breaks.

**agent:claude/t3code-a6d0ff31** at 2026-09-20T18:55:13Z

A match publishes all four fields as arrays in JSON, even the ones the rule does not test, through Match.MarshalJSON. The rendered YAML omits an empty field, as the design asks, so the two surfaces differ on purpose: section 10 and the corpus rules both say an absent collection is [] and never omitted or null, and a consumer reading match.labels should get an empty array rather than undefined. The board file has the opposite pressure, where an omitted field keeps a labels-only pen reading as it did at schema 3.

**agent:claude/t3code-a6d0ff31** at 2026-09-20T18:55:13Z

No new store fixture, which the ticket asked to be decided and said. testdata/README.md couples the corpus to section 11 by code: every code the plan defines needs a fixture and no sidecar may name a code the plan does not define, which TestCorpusCoversEveryPlanCode enforces. This change adds no code, so no condition is uncovered. layout-label-unknown and layout-ticket-missing were re-rendered at schema 4 with check --fix and each still gives exactly its expected.json; layout-label-unknown's field moved to pens.fe.match.labels. layout-not-canonical stays a schema 2 file and is now non-canonical for one more reason than it was, which is what it exists to show. Note for anyone who repeats this: running check --fix over the whole corpus rewrites layout-not-canonical and destroys the fixture, so restore it afterwards.

**agent:claude/t3code-a6d0ff31** at 2026-09-20T18:55:13Z

How a pen decodes, since it is the part most likely to be misread later. UnmarshalYAML cannot see the file's schema, so the pen decoder accepts exactly one of match or requiredLabels and Parse then refuses the wrong one for that schema, which extends the existing rule that routing fields require schema 3. The JSON decoder takes match only, since the one JSON writer is the canvas server replacing a whole Routing and it is built against this package. decodeYAMLRecord and decodeJSONRecord now take a recordFields of required, optional and oneOf, which is also what let the match record reject an unknown field: a nested yaml.Node decode does not inherit KnownFields from the document decoder.

**agent:claude/t3code-a6d0ff31** at 2026-09-20T18:58:05Z

Terva's first round found nothing at 8481d88f4ce5, reviewed against the stacked base b19bcd561104, and Forgejo CI is green. PR 216 is ready for the maintainer's merge after PR 215; both release as v0.23.0, and this ticket closes when that release is named and the canvas has bumped to it.

## Summary

Merged in PR 216 as part of fd32d73 and released as v0.23.0 on 2026-09-20. Schema 4: a pen's match record with labels, status, type and parent, schema 3 read as match.labels, writes at schema 4; layout.Route and the CLI read every field; pen add takes --status, --type and --parent. The canvas half follows in that repository's TKT-01M2Y91C17YTE0W0P3RBHTF50Y against this version.
