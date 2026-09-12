---
schema: 2
id: TKT-01M29F9KSAEKDG85KD73X54S04
title: Make an export with no trailing newline survive its own import
type: bug
status: done
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
updated_at: 2026-09-12T01:39:28Z
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

- [x] A file whose last byte is not a newline round-trips through AddedFileHunk and ParseAddedFiles unchanged, or is refused with a message naming the trailing newline
- [x] The choice between honouring the marker and refusing the input is recorded with its reasoning
- [x] TestAFileWithNoTrailingNewlineDoesNotRoundTrip is replaced by a test asserting the chosen behaviour
- [x] The export before-image still passes, so no artifact a store already produces has moved

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

**agent:terva/mieli** at 2026-09-12T01:35:22Z

draft to ready: Promoted by the user. The repair choice is settled: honour the marker in ParseAddedFiles, because refusing in AddedFileHunk would narrow a surface v0.16.0 published.

**agent:terva/mieli** at 2026-09-12T01:38:57Z

Built on `fix/interchange-trailing-newline`. The repair is to honour the marker, settled by the user before the branch.

### The groom note's forward guidance is superseded

The groom note of 2026-09-12T00:42:30Z said to settle this inside the design of TKT-01M298MJ0 (Build git ticket create --file), because that ticket decides whether a document that never came from `Render` reaches the wire format. That ordering is now spent, and the answer it was waiting for is no.

`create --file` shipped in v0.17.0, and the trigger still has not fired. `exportTicketFiles` reads the ticket file back off disk with `os.ReadFile(t.Path)`, and that file was written by `Render`, so a document whose last byte is not a newline is parsed, re-rendered with one, and the missing byte never reaches `AddedFileHunk`. Measured rather than read: all 124 `.md` files in this store end in `0a`.

So the defect stayed latent through the CLI. What changed is the exposure, not the reachability: v0.16.0 published both functions, so any consumer can hand `AddedFileHunk` arbitrary bytes and get back a patch that `ParseAddedFiles` refuses.

### The decision and why

Honouring the marker on the way in, recorded in plan 12.8. Two reasons.

`AddedFileHunk` accepts data with no trailing newline in a released binary, so refusing it is a change to a published surface under 12.4. Teaching the parser to read the marker only accepts input that was refused before, which breaks nobody.

And the writer was never wrong. `TestAHunkWithNoTrailingNewlineAppliesWithRealGit` applies the hunk with real `git apply` and gets the exact bytes back, both with and without a trailing newline. The artifact has always been byte-compatible with git; only our reader disagreed with it. Refusing would have thrown away a valid patch to avoid fixing a parser.

This supersedes the description's claim that refusing is "cheaper", which the groom note had already begun to correct.

### The change

`ParseAddedFiles` tracks the marker in `noTrailingNewline` and trims the final newline once into `content`, which then feeds both the blob check and `AddedFile.Body`. Trimming once rather than twice is what keeps the name that is verified the name of the bytes that are handed back.

### Evidence

`TestAFileWithNoTrailingNewlineDoesNotRoundTrip` is gone, replaced by `TestAFileRoundTripsWhateverItsLastByte`, a two-row table that asserts the bytes back with both lengths in the failure message.

The second row is a control rather than a duplicate. A parser that trimmed the last byte unconditionally would pass the no-newline row and break every real export, since `Render` makes every file the store writes the trailing-newline row.

The test was proven able to fail, not merely observed passing. With `ticket/interchange.go` reverted to main's version the no-newline row fails and the control row passes, and reapplying the saved patch restored the file byte-identical by sha256.

`TestExportMatchesThePinnedBeforeImage` passes and `cli/testdata/export-before-image/` is untouched in `git status`, so no artifact a store already produces has moved. `go test ./...` is green.

## Summary

Done on `fix/interchange-trailing-newline`, all four criteria met.

`ParseAddedFiles` honours git's `\ No newline at end of file` marker instead of skipping it, so a file whose last byte is not a newline round-trips through `AddedFileHunk` and back unchanged. The decision and its reasoning are in plan 12.8.

The repair was the additive one by the user's choice. Refusing such a file at export would narrow `AddedFileHunk`, which v0.16.0 published, making it a break under 12.4. Honouring the marker only accepts input that was refused before.

The writer turned out never to have been wrong. Real `git apply` lands the hunk and produces the exact bytes, so the artifact was byte-compatible with git all along and only the reader disagreed with it.

The defect was never reachable through the CLI and still is not. `create --file` shipped in v0.17.0 and did not fire the trigger, because export reads the ticket file back off disk after `Render` has written a trailing newline onto it. The exposure is a caller of the published library, not a user of the binary.

`TestAFileRoundTripsWhateverItsLastByte` replaces the pinning test, with a trailing-newline control that catches an unconditional trim. `TestAHunkWithNoTrailingNewlineAppliesWithRealGit` is the git-compatibility half. The export before-image is untouched.
