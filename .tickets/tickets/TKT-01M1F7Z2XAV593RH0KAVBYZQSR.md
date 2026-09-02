---
schema: 1
id: TKT-01M1F7Z2XAV593RH0KAVBYZQSR
title: Let a caller renew a claim rather than replace it
type: task
status: done
status_reason: null
priority: low
labels:
  - claims
assignees: []
milestone: null
parent: null
dependencies: []
references:
  - ref: proposal:git-ticket
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-01T19:43:32Z
updated_at: 2026-09-02T16:01:17Z
created_by:
  id: human:sothr
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Deferred question 1 of plan section 15. Claiming a ticket you already hold currently rewrites the claim. Renewing would extend expires_at and leave the rest alone. Nobody has asked for it yet.

## Summary

Answered in plan 6.4. Re-claiming a ticket you already hold now renews it rather than replacing it. claimed_at survives, and so does an expiry that nothing else supplied. Running the question is what found the defect under it, because a re-claim with no --expires-in cleared expires_at outright and widened a bounded claim into an unbounded one. A --renew flag was ruled out by the file format, since the claim stores two instants and never the duration between them.
