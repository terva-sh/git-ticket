---
schema: 2
id: TKT-01M2NT7QS84SB5P7DR2GXB9DZ0
title: Give an empty actor roster an actor without an editor
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - integration
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references:
  - ref: ticket:TKT-01M1ZMDYPKS4MKCGDYWP5JVX2G
    path: null
claim: null
archive: null
created_at: 2026-09-16T19:13:58Z
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

A store whose `actors:` roster is empty refuses every write, and there is no
command that can add one. The only repair is opening `config.yml` in an editor.

terva has a ticket parked on this since 2026-09-08. Its last unmet acceptance
criterion reads, whole:

> A store that already exists with an empty roster can be given an actor without
> hand-editing `config.yml`

### Why it is awkward rather than merely missing

An empty roster is reachable by an ordinary route. `init` with no `--actor` and
no terminal creates the store, warns that the roster is empty, and returns
success, which is the right call for a script. The store it leaves behind then
refuses the next write, and the thing that would fix it is a write.

So the one state a user most needs a command for is the state in which no
command works.

terva's own surfaces hit this from two directions. `terva ticket ui` and the
`/ticket` TUI both refuse a store with an empty roster and name the lines to add,
because git-ticket rejects every write to such a store and an editor is the only
repair today. That refusal text exists only because this gap does.

### Checked at v0.18.1 before filing

38 verbs, up from the 35 counted when terva first parked this:

```
init create update show copy export import list ready search ui status claim
release link unlink deps files refs ac dod plan note comment summary archive
unarchive remove check migrate schema config series instructions completion
self-update install-merge-driver merge-driver
```

No `actor` verb.

`config` is new since then and is the obvious candidate. It is read-only: "print
what this store configured, including the allowlists", with no flag that writes.
So it does not close this.

### Not a prescription

An `actor` verb is the shape terva's ticket guessed at, but the decision is
upstream's. `config --set` reaching the roster would do it. So would `init
--actor` on an existing store treating an empty roster as the one thing it may
still fill, though that overloads a verb whose name says otherwise.

What the caller needs is only this: a supported way to move a store from "empty
roster, every write refused" to "one actor, usable", without a text editor.

### Reported late, and that is the real lesson

terva parked this on 2026-09-08 and did not file it here. Searching this store
for `roster` and for `actors:` on 2026-09-16 returns zero hits across every
status. A downstream block that upstream never hears about is indistinguishable
from work nobody wants, and it sat for eight days looking like the latter.
