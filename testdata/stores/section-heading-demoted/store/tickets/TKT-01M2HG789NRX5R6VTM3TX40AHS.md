---
schema: 1
id: TKT-01M2HG789NRX5R6VTM3TX40AHS
title: A subheading names a section the format owns
type: bug
status: ready
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
created_at: 2026-08-31T12:00:00Z
updated_at: 2026-08-31T12:00:00Z
created_by:
  id: human:sothr
  name: Drew Short
updated_by:
  id: human:sothr
  name: Drew Short
extensions: {}
---

## Description

One finding, and three headings that could each have produced one. What
separates them is the name, never the level.

`### Acceptance criteria` below is the recorded warning. It renders as the
section it names and `show` prints its boxes, but `parse` puts the whole thing
in Description, so `ac --check 1` answers that the section has 0 items.

### Acceptance criteria

- [ ] this looks like a criterion and cannot be ticked
- [ ] nor can this one

### Trigger

A subheading naming nothing the format owns is ordinary and wanted. This store
carries dozens of them and they are why the check cannot key on the level: a
rule that flagged every `###` would be noise nobody could act on.

The fenced block below holds a third heading. It is text, not a heading, and a
fixture that omitted it would pass even if the check ignored fences entirely.

```markdown
### Notes

Inside a fence, so parseBody does not see it and neither does check.
```

## Acceptance criteria

- [ ] the real section, which is what the demoted one above was meant to be
