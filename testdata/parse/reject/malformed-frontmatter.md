---
schema: 1
id: TKT-01K3ZYV850HN9T7RV7553BV01C
title: Frontmatter a person broke by hand
type: task
status: ready
  priority: normal
labels: []
assignees: []
---

## Description

`priority` is indented one level under `status`, which makes YAML report a
mapping value where none is allowed. This is the shape a hand edit leaves
behind, not a shape any writer produces.

The reader cannot recover an ID from a document that did not parse, so the
recorded finding carries `"ticket": null`.
