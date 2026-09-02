---
schema: 1
id: TKT-01M1HJKBYGENBTJ7F9S71BN3Q1
title: Decide whether section 15 stays a hand-maintained index
type: spike
status: done
status_reason: null
priority: normal
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
references:
  - ref: plan:deferred-questions
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-02T17:27:51Z
updated_at: 2026-09-02T17:56:29Z
created_by:
  id: agent:terva/dev-loop
  name: ""
updated_by:
  id: agent:terva/dev-loop
  name: ""
extensions: {}
---

## Description

Section 15 is a hand-maintained list of questions that are already tickets. Every entry names the ULID of the ticket holding the detail, so the section is an index of a subset of the store, kept in step by attention.

That costs something measurable. Every new question inserts at the same anchor point, the end of the parked entries, so two agents filing questions in parallel conflict there every time. Naming by subject and ULID fixed the identity half of this and did not touch the insertion half. Filing one question this session took two rebases, a renumber, a reference repoint, a message rewrite and a PR body rewrite, and none of that work was about the question.

The shape of the answer is not obvious, which is why this is a spike rather than a task.

Generation is the ambitious version: mark the question tickets, let git ticket list produce the parked list, and commit the result behind a determinism gate the way a generated artifact needs. It ends the conflict class, and it moves part of the design of record into the store, which 1 and the reading order both say lives in docs/plan.md.

A pointer is the small version: section 15 keeps its prose and stops claiming to be the complete list, with the store as the index. Conflicts vanish because nothing is appended, and a reader offline loses the ability to see the open questions in the document that explains them.

Three things any answer has to handle:

- The settled half is the valuable half. Entries like the compatibility policy and the module path carry their answers and the reasoning behind them, and no query reproduces that. Generation covers the parked entries only, so the section stays prose either way and the benefit is smaller than it first looks.
- The questions are not machine-distinguishable today. Most are type spike, but Backlog.md import and renewing a claim are type task. A question label would need adding to the config.yml allowlist first, which is currently empty, or every labelled ticket earns a label_unknown warning.
- A generated block still conflicts if it is committed, because two agents regenerate the same lines. It is mechanically resolvable by regenerating rather than by reading, which is better, but it is not zero.

The trigger is a third collision in section 15, or a reader who cannot answer what is open because the section and the store disagree. Two collisions have happened. The store and the section agree today.

## Notes

**agent:terva/dev-loop** at 2026-09-02T17:51:31Z

The trigger fired on the merge of this question's own section 15 entry. What counts as a store (TKT-01M1HJTC) and this entry were written in parallel, appended at the same anchor, and conflicted, taking PR #17 to mergeable: False. That is the third collision here and the first of its kind: the two before it were two agents choosing the same number, which naming by subject and ULID settled. This one is two agents landing text in the same place, which naming never addressed.

**agent:terva/dev-loop** at 2026-09-02T17:56:29Z

Settled as stays hand-maintained. The measurement that decided it: four of the eight parked entries were longer than the ticket descriptions they name, custom statuses 756 characters in the plan against 192 in the ticket. The section is authorial, not an index, so a pointer would have cost a prose migration nobody had counted.

## Summary

Section 15 stays hand-maintained and the parked entries stay in docs/plan.md. Three reasons, in the plan under the same subject. The section is authorial rather than an index, four of eight entries being longer than their tickets. Splitting parked from settled would split one narrative across two homes and make settling a question a content migration. The collisions are append conflicts that stop the merge and resolve by keeping both, so they cost delay and never a wrong answer, where generation would cost a determinism gate and a label taxonomy the format needs for nothing else. list --label question is a real query, so this was a choice and not a limit.
