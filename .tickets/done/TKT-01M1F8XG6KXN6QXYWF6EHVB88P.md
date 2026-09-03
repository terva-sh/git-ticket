---
schema: 1
id: TKT-01M1F8XG6KXN6QXYWF6EHVB88P
title: Decide whether the module path should follow where the code lives
type: spike
status: done
status_reason: null
priority: normal
labels:
  - release
assignees: []
milestone: null
parent: null
dependencies:
  - TKT-01M1F7Z31HR5GV06D6Y7WZWJK4
references:
  - ref: proposal:git-ticket
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-01T20:00:08Z
updated_at: 2026-09-02T17:09:39Z
created_by:
  id: human:sothr
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

A deferred question in plan section 15. go.mod declares github.com/terva-sh/git-ticket, and the repository is hosted on an internal Forgejo instance with no public mirror, so the declared path is not fetchable. A module path is a promise about where a consumer can find the code, and nothing enforces it: a replace directive or a local path works either way, which is why this can sit wrong for a long time without failing. It is also hard to change once anything depends on it. Either publish a mirror at the declared path, or rename the module to the host that serves it. Phase 3 is the first consumer, so decide before terva depends on either answer.

## Notes

**agent:terva/mieli** at 2026-09-02T14:57:58Z

Numbering note. This ticket calls itself deferred question 7, which it was under the old scheme where numbers closed up as questions were settled. Question 7 now means the parent hierarchy, TKT-01M1FCMN7QEWM584N192NBC7TD. Plan section 15 froze the numbers because of this collision, and each entry there now names its ticket ULID. Nothing here changes; the description is left as it was written.

**agent:terva/mieli** at 2026-09-02T17:09:39Z

Correction to the note above. Numbering was not frozen, it was dropped. Plan section 15 now names each question by a subject in bold and the ULID of its ticket, and this ticket's description no longer claims a number. The module path question is the entry beginning 'The module path'.

## Summary

Settled by publishing. go.mod already declared github.com/terva-sh/git-ticket and a public mirror now serves it, so the declared path is the real one and no import changes. Renaming to the internal host was wrong on its own terms: a private hostname does not belong in a public artifact, and it is the one string in a Go project that cannot be corrected cheaply once anything imports it. Recorded in plan 12.2 and section 15.
