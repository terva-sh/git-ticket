---
schema: 1
id: TKT-01M1FCGPG9Z1TMEX28AKAAFXB1
title: "Slice 3: add the five write tools"
type: task
status: archived
status_reason: null
priority: normal
labels:
  - integration
assignees: []
milestone: null
parent: TKT-01M1FCHH3QN04AYHZAP1M8DNQK
dependencies:
  - TKT-01M1FCGPFB787F6MQ37QYJY4PA
  - TKT-01M1FCJ8XDS7ECVWQ2W7VS8X0G
references: []
claim: null
archive:
  archived_at: 2026-09-01T22:03:57Z
  from_status: ready
  reason: "Terva owns this work and terva has no store here, so tracking it in git-ticket's ledger would go stale the moment terva starts. The content moved to docs/handoff-terva-phase-3.md, which is the artifact a terva agent works from. Archived rather than cancelled because none of it was rejected: it is being done, elsewhere. Unarchive if the split changes."
created_at: 2026-09-01T21:03:03Z
updated_at: 2026-09-01T22:03:57Z
created_by:
  id: human:sothr
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

ticket_create, ticket_update, ticket_transition, ticket_claim, ticket_comment. All workspace-mutation.

Picking local-data here would be a real bug. IsReadOnlyAuthority auto-allows local-read and local-data, and local-data means the tool own host-managed directory under TERVA_HOME. A ticket store lives in the user repository and lands in their next commit, so it is gated like write and edit.

Revision preconditions are required in the tool schema even though ApplyOptions.IfRevision is optional in the library. The library doc comment already records the reason: a person typing one command should not be forced into a read-then-write dance, but agents read before they write anyway and multi-agent is where the races happen. A stale precondition returns stale_revision and the tool surfaces a model-readable refusal naming the current revision, so the agent re-reads rather than retrying blindly.

Mutations carry Actor{ID, Name}, filled from the terva session identity when available, in the shape this store already uses: human:name and agent:terva/persona. Identity stays optional in the format.

Needs permission-policy tests covering plan mode and headless refusal.

## Notes

**agent:terva/mieli** at 2026-09-01T22:03:57Z

archived from ready: Terva owns this work and terva has no store here, so tracking it in git-ticket's ledger would go stale the moment terva starts. The content moved to docs/handoff-terva-phase-3.md, which is the artifact a terva agent works from. Archived rather than cancelled because none of it was rejected: it is being done, elsewhere. Unarchive if the split changes.
