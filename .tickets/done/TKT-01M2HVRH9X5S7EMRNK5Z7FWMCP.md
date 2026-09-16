---
schema: 2
id: TKT-01M2HVRH9X5S7EMRNK5Z7FWMCP
title: note --show and --list are absent from the help that documents note
type: bug
status: done
status_reason: null
priority: normal
due_on: null
labels: []
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references:
  - ref: code:notes
    path: cli/notes.go
claim:
  actor: agent:claude/t3code
  branch: t3code/review-new-tickets
  worktree: /home/sothr/.t3/worktrees/git-ticket/t3code-6907238f
  commit: f5cf792ca2870c5a86eccd520a3182d72a4ed1fd
  claimed_at: 2026-09-16T17:03:55Z
  expires_at: null
archive: null
created_at: 2026-09-15T06:23:39Z
updated_at: 2026-09-16T17:07:33Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:claude/t3code
  name: ""
extensions: {}
---

## Description

Found while terva raised its pin from v0.14.3 to v0.18.1 and surveyed what the
six releases changed. Verified against the published v0.18.1 binary, installed
from the module proxy and run at an absolute path.

`show` elides older notes and the stand-in line tells the reader exactly how to
get them back:

```
_2 earlier notes, 1-2 hidden. Read them with `git ticket note TKT-... --show 1-2`, or `--list` for an index._
```

Both flags work. Neither appears in `git ticket note --help`, which lists only
`--actor`, `--file`, `--if-revision`, `--json`, `--lock-timeout` and `--store`,
under the usage line `git ticket note ID TEXT`.

So the only documentation of the reading forms is a sentence that appears when
a ticket has more than one note. A reader who meets `note --help` first is told
`note` appends, and nothing else.

### Why the help cannot see them

Not an oversight in a line somewhere. `noteReadArgs` in `cli/notes.go` picks
`--list` and `--show` out of argv before `runTextEntry` is called, and the
comment there says why: `runTextEntry` serves `note`, `comment`, `plan` and
`summary` from one body and treats every non-flag word as text, so a `--list` it
did not expect would be appended as prose. The split keeps that function
unchanged and unshared with a mode the other three do not have.

That is a sound reason to parse them early. The consequence is that they are
never registered on the `FlagSet` the usage text is generated from, so no amount
of care in `runTextEntry` will surface them.

### Suggested shape, not a prescription

The cheapest fix that keeps the parsing decision intact is to write the two forms
into the command's own usage and flag text rather than into the shared one, so
`note --help` reads something like:

```
usage: git ticket note ID TEXT
       git ticket note ID --list
       git ticket note ID --show N | N-M | all
```

Registering them on the real `FlagSet` would also work and would cost the thing
the comment is protecting, so it is probably the wrong trade.

Worth deciding at the same time: whether `comment` gets the same reading forms.
`Entries` reads both sections, `show` compacts only notes today, and a user who
learns `note --list` will try `comment --list` next.

### Why terva cares

terva's `ticket_get` returns the whole `Notes` section, which is the same defect
`show` fixed in v0.18.0, and terva is about to fix it the same way. Whatever
terva shows in place of the withheld notes has to name a retrieval path. Pointing
at `note --show` is the obvious one and the natural thing to copy, and a flag a
user cannot find in `--help` is a poor thing to point a user at.

## Acceptance criteria

- [x] git ticket note --help documents the reading forms alongside the appending one.
- [x] The parsing stays as it is, or the change says why moving it is worth the case noteReadArgs was written to prevent.
- [x] A decision is recorded on whether comment gains the same --list and --show, since Entries already reads that section too.

## Definition of done

- [x] A test fails if the help text stops naming --show or --list.

## Notes

**agent:claude/t3code** at 2026-09-16T17:07:27Z

**comment does not gain --list and --show.** That is the third criterion, and
the reason is measured rather than argued from symmetry.

`show` compacts Notes only. `cli/commands.go` passes `t.Body.Notes` through
`compactNotes` and prints `t.Body.Comments` whole on the next line, and a
five-comment ticket prints all five with nothing elided. So the pressure that
made `note --show` necessary does not exist for comments: `note --show` was
built because `show` started withholding text, and a retrieval path for text
nothing withholds is a second way to read what is already on screen.

The prediction in the description, that a user who learns `note --list` will try
`comment --list`, is probably right. The answer to it is that the attempt fails
loudly and costs nothing, while shipping the pair would spread the early-parse
special case in `noteReadArgs` to a second command in exchange for symmetry
alone.

**The trigger to revisit is exact**: if `show` ever compacts Comments the way it
compacts Notes, `comment` needs the reading forms in the same change, because
that is the moment the argument above stops holding.

**The parsing stayed where it is, as the second criterion allows.** `noteReadArgs`
is untouched, and `TestNoteWritesTextThatLooksLikeAFlag` still proves the case it
was written to prevent.

One thing was fixed that this ticket did not name, found while checking that the
help page was true before shipping it. The page lists six flags, and the two
reading forms accepted none of them: `note ID --list --store PATH` failed with
"note --list and --show take one ticket ID", because `noteReadArgs` hands
everything it did not claim to `runNoteRead`, which counted the words and
refused anything but one. That contradicts the top-level usage, which promises a
global may come before or after the command, and it would have made the new help
page misleading in a way the old one was not: three usage forms under one flag
list that applied to only the first.

The fix parses what `noteReadArgs` left behind, downstream of it, so the split
that protects `runTextEntry` is not involved. `--json` is still refused with the
message it already had, and two IDs are still two IDs.

## Summary

note --help now opens on all three forms: the appending one from the dispatch table and the two reading ones from commandUsageForms, a lookup beside commandEpilogue and a lookup for the same reason. An epilogue says why --list and --show are not in the flag list, since they are read before the FlagSet exists, and that everything after -- is text. comment does not gain the pair: show compacts Notes only and prints Comments whole, so nothing is withheld there to retrieve. The reading forms also now accept the globals after the command, which they did not.
