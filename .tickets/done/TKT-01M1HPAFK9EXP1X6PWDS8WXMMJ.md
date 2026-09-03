---
schema: 1
id: TKT-01M1HPAFK9EXP1X6PWDS8WXMMJ
title: Let create seed the acceptance criteria and the definition of done
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
created_at: 2026-09-02T18:32:54Z
updated_at: 2026-09-02T19:01:14Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

create takes --title, --type, --priority, --description, --parent, --label, --assignee, and --depends-on. Filling in the checklists means a separate ac --add call per criterion, and dod --add per item.

For a person that is friction. For an agent it is worse: every extra call is a round trip and another chance to leave the ticket half-built when the session ends. Backlog.md seeds a whole task in one command and its agent guidelines lean on that. See section B3 of docs/review-backlog-md.md.

Scope: create --ac TEXT and create --dod TEXT, both repeatable, appending in the order given. This is additive under 12.4 and touches no format rule, because AddChecklistItem already exists and create would call it.

## Summary

Added repeatable create --ac and --dod, seeding both checkbox sections in the write that files the ticket.

CreateOptions carries AcceptanceCriteria and DefinitionOfDone, and both render through uncheckedItem, the same helper AddChecklistItem uses. TestCreateSeedsTheChecklists proves a seeded section is byte-identical to one built by repeated adds, so a store cannot end up carrying two spellings of the same list.

An empty criterion refuses the whole create rather than filing a ticket with half its checklist, which is the rule AddChecklistItem already applied to one item.
