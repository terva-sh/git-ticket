---
schema: 2
id: TKT-01M2NT7QS84SB5P7DR2GXB9DZ0
title: Give an empty actor roster an actor without an editor
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - area/cli
  - area/integration
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
updated_at: 2026-09-16T19:35:56Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:claude/t3code
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

## Acceptance criteria

- [x] A store with an empty roster can be given an actor without hand-editing config.yml
- [x] The command runs against that store, since it writes config.yml rather than a ticket
- [x] Setting the default is part of it, so the repair does not end with the warning telling you to edit config.yml
- [x] The bare form reads the roster and says plainly when it is empty
- [x] Whether actor remove exists is decided and the reason recorded

## Notes

**agent:claude/t3code** at 2026-09-16T19:35:56Z

Built as `git ticket actor [add ID [--name N] [--default]]`, settled with the
user against `config --set` and against overloading `init`. `series` is the
precedent: a vocabulary the store declares, read and written by its own verb,
writing `config.yml` rather than a ticket.

**One correction, and it matters to terva more than to this ticket.** The
description says a store with an empty roster "refuses every write". It does
not. `resolveActor` refuses only when the caller names nobody *and* the config
declares nobody, so `--actor human:someone` writes fine against an empty roster.
Measured against a store `init` left empty: `create --title x --actor
human:someone` succeeded, and the same command without `--actor` was refused.

The ask still stands unchanged, because the roster could only be filled by hand
and that is the criterion terva parked on. But the framing does not: this was
never a store in which "no command works", and terva may want to look again at
the TUI refusal its ticket mentions. If that surface refuses an empty-roster
store outright, it is refusing more than git-ticket does, and passing `--actor`
would have worked the whole time.

**`--default` is not a convenience and the ticket did not ask for it.** Adding
the first actor without it leaves the store resolving writes to whoever heads
the roster and warning on every one, and that warning's own advice is to set
`defaults.actor`. A repair that ends by telling you to edit `config.yml` is not
the repair this ticket asked for, so setting the default is part of the command.
A test holds both halves: with `--default` a write is silent, without it the
warning still fires.

**There is no `actor remove`, and that is a decision rather than an omission.**
It is not the mirror of `series remove`. A series lives in the ID of every
ticket carrying it, so removal can count them and refuse. An actor is recorded
in `created_by`, `updated_by`, every note and comment, and every claim, and
those are history: a ticket saying who wrote a note in March stays true after
that person leaves. Removal cannot rewrite them, and leaving them means a
display name silently empties on the next write to any of those tickets, which
is exactly the behaviour TKT-01M2NT7VNBAM5KSAR0QKBT2JQD documented an hour ago.
Recorded in the plan and in `ticket/actor.go` rather than left as a gap somebody
fills by symmetry.

**On the reporting lesson the ticket ends with.** It is right, and this is the
second instance in one day: the `ApplyImport` partial result had also become
load-bearing downstream without upstream hearing. Both were found by reading
terva rather than by terva filing. Worth noting that `list --cross-branch` found
these three tickets on `main` before they were merged here, which is the tool
solving a smaller version of the same problem.

## Summary

git ticket actor lists the roster and actor add ID [--name N] [--default] fills it, writing config.yml rather than a ticket, which is what lets it run against the empty-roster store it exists to repair. --default is included because otherwise the repair ends with a warning telling you to edit config.yml. No actor remove: an actor is history rather than vocabulary, recorded in created_by, notes and claims, so removal cannot rewrite what it finds. The ticket's premise that an empty roster refuses every write is corrected in a note: it refuses only writes that name no actor.
