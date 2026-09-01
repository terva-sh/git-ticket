---
schema: 1
id: TKT-01K3ZZ2JH000GHB4EE6SNRE6MD
title: Refresh fails when the clock jumps backward
type: bug
status: ready
status_reason: null
priority: high
labels:
  - auth
assignees: []
milestone: null
parent: null
dependencies: []
references: []
claim: null
archive: null
created_at: 2026-08-31T12:01:00Z
updated_at: 2026-08-31T12:01:00Z
created_by:
  id: human:sothr
  name: Drew Short
updated_by:
  id: human:sothr
  name: Drew Short
extensions: {}
---

## Description

Status `ready`, type `bug`. No claim and no dependencies, so `ready` returns it.

The `auth` label is in the `config.yml` allowlist, which is what keeps this
store free of `label_unknown`.
