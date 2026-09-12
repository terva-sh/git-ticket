---
schema: 2
id: TKT-01M29F9KSAEKDG85KD73X54S04
title: Make an export with no trailing newline survive its own import
type: bug
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - format
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references:
  - ref: plan:12.8
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-12T00:11:52Z
updated_at: 2026-09-12T00:42:30Z
created_by:
  id: agent:terva/mieli
  name: Mieli
updated_by:
  id: agent:terva/mieli
  name: Mieli
extensions: {}
---

## Description

`ticket.AddedFileHunk` writes a file that `ticket.ParseAddedFiles` refuses, and the error blames the sender for something nobody did.

`AddedFileHunk` follows git: a file whose last byte is not a newline gets the `\ No newline at end of file` marker, and the index line carries the blob name of the bytes exactly as they are.

`ParseAddedFiles` skips that marker and appends a newline to every line it reads. So it rebuilds a file one byte longer than the one that went out, hashes that, and reports:

    does not match its blob name (...); the patch has been altered or truncated

Nobody altered anything. Export produced an artifact its own import rejects, and the message sends the reader to look for tampering.

### Why it is latent

`Render` always ends a ticket file with a newline, so nothing the store writes takes this path. It was found by writing a round-trip test for the library API, not by a user.

It stops being latent as soon as a file that did not come from `Render` reaches `AddedFileHunk`. `create --file` (TKT-01M298MJ0DN713TQ01WRR472YA) is the ticket that would do it, since a hand-authored document need not end in a newline.

### The decision this needs

Two repairs are available and they are not equivalent.

Honour the marker on the way in: `ParseAddedFiles` tracks `\ No newline at end of file` and omits the final newline when rebuilding. This makes the round trip exact and keeps the artifact byte-compatible with git, which is the format's whole promise. It is the larger change, because the parser currently has no notion of a hunk's last line.

Refuse on the way out: `AddedFileHunk` errors on data that does not end in a newline. Cheaper, and it turns a confusing receive-time error into a clear send-time one, but it narrows what the library accepts and pushes the problem onto every caller.

`TestAFileWithNoTrailingNewlineDoesNotRoundTrip` in `ticket/interchange_test.go` pins the current behaviour and its failure message says what to do when it starts passing.

## Acceptance criteria

- [ ] A file whose last byte is not a newline round-trips through AddedFileHunk and ParseAddedFiles unchanged, or is refused with a message naming the trailing newline
- [ ] The choice between honouring the marker and refusing the input is recorded with its reasoning
- [ ] TestAFileWithNoTrailingNewlineDoesNotRoundTrip is replaced by a test asserting the chosen behaviour
- [ ] The export before-image still passes, so no artifact a store already produces has moved

## Notes

**agent:terva/mieli** at 2026-09-12T00:42:30Z

Groomed. Every claim re-read against the tree, and all of them hold.

`AddedFileHunk` writes `\ No newline at end of file` at `ticket/interchange.go:138`.
`ParseAddedFiles` rebuilds a body with `body.WriteString(l[1:] + "\n")` on every
`+` line and reaches the marker only to `continue` past it, so the rebuilt file
is one byte longer and fails its own blob check.
`TestAFileWithNoTrailingNewlineDoesNotRoundTrip` is at
`ticket/interchange_test.go:90`. Still latent, because `Render` always ends a
ticket file with a newline.

### What changed: the decision got more expensive

v0.16.0 published `AddedFileHunk` and `ParseAddedFiles` as exported API. When
this was filed, hours before the tag, both repairs in the description cost the
same. They no longer do.

Honouring the marker on the way in stays additive. It accepts input that is
refused today and changes nothing a current caller relies on.

Refusing on the way out is now a break. `AddedFileHunk` accepts data with no
trailing newline in a released binary, and narrowing that is a change to a
published surface under plan 12.4, so it buys a version bump on top of the code.
The description calls it "cheaper", and against a released library it is not.
That reasoning is superseded by this note.

This does not settle the choice. It moves the argument toward honouring the
marker, and the criterion asking for the decision with its reasoning still
stands.

### The trigger has not fired

`create --file` (TKT-01M298MJ0) is what would send a hand-authored file into
`AddedFileHunk`, and it remains an unpromoted draft. Its one dependency,
TKT-01M298KEG, went done with v0.16.0, so it is now unblocked rather than
started.

The order that follows: this is best settled inside the design of
TKT-01M298MJ0, because that ticket decides whether a document that never came
from `Render` reaches the wire format at all. Settling it first picks an answer
to a question nobody has asked yet.
