---
schema: 2
id: TKT-01M2NT7VNBAM5KSAR0QKBT2JQD
title: Say that --actor resolves its display name from config.yml
type: task
status: draft
status_reason: null
priority: low
due_on: null
labels: []
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references:
  - ref: code:actors
    path: ticket/config.go
claim: null
archive: null
created_at: 2026-09-16T19:14:02Z
updated_at: 2026-09-16T19:14:20Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

`--actor` takes an id. The write it produces also fills a display NAME, resolved
from `config.yml`, and nothing in the help or the docs says so.

The behaviour is good. It is the discoverability that costs.

### The four cases, probed at v0.18.1

Run in a throwaway store against the released binary, because nothing documented
the answer:

| `config.yml` says | `--actor` writes |
|---|---|
| the id, declared **with** a name | that name |
| the id is not declared at all | `name: ""` |
| not declared, and the file already held a name for that id | `name: ""` — the existing name is NOT preserved |
| the id, declared with `name: ""` | `name: ""` |

So the roster is the source of truth for display names, and a write never
carries one forward from the ticket file.

### What it cost downstream

terva spent a session on this. Its `scripts/pr.sh` writes a `ships-in:<n>`
reference at pull-request time, and an agent-opened PR was recording
`updated_by.name: ""` where earlier writes through terva's own tools had
recorded `Mieli`. The natural reading was that `--actor` drops the name, and a
ticket was filed against `pr.sh` saying so, carrying this as an open question:

> Worth checking before deciding: whether git-ticket preserves an existing name
> when a write names the same actor by id alone. If it does, this is a
> git-ticket question rather than a `pr.sh` one, and the probe is two writes in
> a throwaway store. This session did not run that probe.

Running it showed the cause was neither: terva's store declared
`agent:terva/mieli` with an empty name. One line of `config.yml` fixed it and
`pr.sh` needed no change for that half.

That is a good outcome, reached by building a store and probing four cases,
where a documented sentence would have reached it immediately.

### The ask

A line in the `--actor` help, or wherever the roster is described, saying that
the display name comes from `config.yml` and that an actor absent from the
roster is written with an empty name.

The third row is the one most worth stating, because it is the one a reader is
likeliest to guess wrong: an existing name in the file is not preserved. Someone
who assumes otherwise will read a blanked name as data loss rather than as a
roster that never had the name.

### Adjacent, and not asked for

A store declaring an actor with `name: ""` is the state that produced the
confusion. Whether `check` should say something about it is a separate question
and this is not a request for it.
