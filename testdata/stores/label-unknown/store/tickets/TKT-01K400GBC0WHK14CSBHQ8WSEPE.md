---
schema: 1
id: TKT-01K400GBC0WHK14CSBHQ8WSEPE
title: Labelled with a word the store does not know
type: task
status: ready
status_reason: null
priority: normal
labels:
  - auth
  - telemetry
assignees: []
milestone: null
parent: null
dependencies: []
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

`auth` is in the `config.yml` allowlist and `telemetry` is not, so exactly one
of the two labels produces a finding.

4.1 makes the allowlist advisory on purpose. A label is how someone marks work
before the vocabulary catches up, and erroring would mean a new label had to be
committed to config before it could be used once. So `check` warns, `--strict`
turns it into an error for a repository that wants the tighter rule, and nothing
is ever rewritten.

The finding names the ticket rather than the individual label, because `field`
identifies the frontmatter field and `labels` is the field.
