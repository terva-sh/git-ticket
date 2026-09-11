---
schema: 2
id: TKT-01M298KEGY8KV17MCFN9V1A4WG
title: Move the export and import interchange into the ticket library
type: chore
status: in-progress
status_reason: null
priority: high
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
  - ref: plan:12.2
    path: docs/plan.md
  - ref: plan:12.8
    path: docs/plan.md
claim:
  actor: agent:terva/mieli
  branch: refactor/interchange-library
  worktree: null
  commit: null
  claimed_at: 2026-09-11T23:22:22Z
  expires_at: null
archive: null
created_at: 2026-09-11T22:14:54Z
updated_at: 2026-09-11T23:34:51Z
created_by:
  id: agent:terva/mieli
  name: Mieli
updated_by:
  id: agent:terva/mieli
  name: Mieli
extensions: {}
---

## Description

`export` and `import` arrived with their whole domain in `cli/`. The layout rule
this repository states is that `ticket/` is the library and `cli/` holds the
choices about flags, output and exit status, and the interchange inverted it.

### The measurement

`cli/export.go` and `cli/import.go` are 931 lines carrying zero exported
symbols. The entire `cli` package exports three things: `Env`, `UIParams`, and
`Run(args []string, env Env) int`.

So an embedding host reaches this feature only by building argv and reading
streams:

```go
cli.Run([]string{"export", id, "--out", dir}, env)
```

For export that is survivable, because a `mutation-result` envelope names the
files it wrote. For import it is not. There is no `--json` form at all, so a
host gets an exit code and English prose, and the map from each incoming ID to
the one this store minted, which is the only thing it cannot reconstruct, is in
neither.

### What is on the wrong side

Domain that belongs in `ticket`:

- `blobSHA`, `addedFileDiff`, `mboxMessage`, the wire format itself
- `parseNewFileHunks`, its reader
- `importOrder`, a topological sort over ticket edges
- `keepKnownLabels`, a thin wrap of `cfg.KnownLabel`
- `reconcile` and `reconciled`, which are the interchange rule and the most
  valuable thing the contribution brought

Presentation that rightly stays in `cli`: `runExport` and `runImport`, the
printing in `importPreview` and `importAdopt`, `warnDanglingEdges`,
`exportIdentity` because it reaches git through `cli`'s own `readGit`, and the
cosmetic `plusBar` and `exportDirIsFree`.

### The shape

A plan-and-apply pair, which is the library-shaped version of the
preview-then-adopt split the CLI already has:

```go
func (s *Store) Export(ctx context.Context, o ExportOptions) (*Export, error)
func (s *Store) PlanImport(ctx context.Context, o ImportOptions) (*ImportPlan, error)
func (s *Store) ApplyImport(ctx context.Context, p *ImportPlan) (*ImportResult, error)
```

`Export` returns the artifact as bytes rather than writing a directory. The CLI
decides that it lands on disk; another host may put it on a wire, and a return
value is testable without a filesystem. `PlanImport` returns what `reconcile`
decided for each incoming ticket, and the CLI renders that as its preview
instead of computing the same answer a second time.

### Typed changes, not English

`reconciled.Changes` is a `[]string` of English sentences, and that is debt from
the commit that fixed the data loss rather than from the contribution. A host
cannot render "label \"upstream\": not carried, this store does not declare it"
in its own voice; it wants a kind and a value. Moving the code is the moment to
make them typed, so this is not a pure relocation and should not be reviewed as
one.

### Cost and risk

Additive to `ticket`, nothing in `cli` breaks, so a minor under 12.4 with no
schema move.

The verification is cheap and strong. An export is deterministic given a fixed
clock and identity, so generate the artifacts before and after and compare them
byte for byte. That is the before-image technique PR #135 used to prove a
refactor moved no output.

## Acceptance criteria

- [ ] ticket exports Export, PlanImport and ApplyImport, and the interchange domain lives in the ticket package
- [ ] Export returns the artifact as bytes; writing a directory is the CLI's decision
- [ ] The reconciliation is computed once and rendered twice: PlanImport answers, and the CLI preview prints that answer rather than deriving its own
- [ ] Changes are typed values carrying a kind and a value, not preformatted English
- [ ] cli/export.go and cli/import.go hold flags, printing and exit status only
- [ ] An export generated before and after the move is byte-identical under a fixed clock and identity
- [ ] Plan 12.2 and 12.8 say the library owns the interchange and the CLI renders it
- [ ] A handoff document for terva describes the new API, with its code compiled against the published module

## Implementation plan

1. Pin the before-image first. Generate an export from this repository's own
   store with a fixed clock and identity and keep the bytes. Without it there is
   nothing to compare against later and the criterion cannot be met.

2. Move the wire format: `blobSHA`, `addedFileDiff`, `mboxMessage`,
   `parseNewFileHunks`. These have no dependency on `cmdContext` and go first,
   which shrinks the rest.

3. Define the typed `Change`. Enumerate the kinds the current strings carry:
   a dropped label, a dropped milestone, a dropped reference path, a dropped due
   date, a non-default blocks_on, checklists carried unticked, and the work
   record. The CLI keeps the current wording by formatting these, so the human
   output does not move while the API gains a shape.

4. Move `reconcile` to `PlanImport`, and `importAdopt`'s write loop to
   `ApplyImport`. The preview becomes a renderer over the plan.

5. Move `importOrder` and `keepKnownLabels`, which are already pure.

6. Compare the artifacts. Then the plan text, then the handoff.

Do not fold `export --since` or `create --file` into this. Both depend on it and
both are separately reviewable; a move that also adds features cannot be proven
byte-identical.

## Notes

**agent:terva/mieli** at 2026-09-11T23:22:18Z

draft to ready: The user directed this work to start now that the ticket body repair has merged.

**agent:terva/mieli** at 2026-09-11T23:26:08Z

Step 1 of the plan is done, on `refactor/interchange-library` at `5656c0b`.

`cli/testdata/export-before-image/` holds the pinned artifact, a cover letter
and a one-patch series carrying two tickets from the `clean` fixture.
`TestExportMatchesThePinnedBeforeImage` compares the bytes.

Three inputs had to be held still, and only two were already handled. `runCLI`
pins the clock to `referenceInstant`. The fixture supplies IDs and timestamps a
person wrote, instead of ULIDs minted during the run. The third would have
silently broken this on another machine: `exportIdentity` asks git for
`user.name` and `user.email`, so the `From:` header carries whoever ran the
suite unless the repository sets its own. `newFixtureStore` sets it to
`Pinned Sender <pinned@example.com>`.

No `-update` flag, deliberately. The bytes are evidence from before the move,
and a flag that rewrites them turns a failed comparison into a formality.

Falsification was run rather than assumed: changing one byte of the pinned
cover letter turns the test red, restoring it turns it green.

One cosmetic defect is pinned as-is. In the patch body the status column does
not align, because `blocked` is seven characters and `review` is six and both
are followed by two spaces. Fixing it during the move would make the artifact
differ for a reason unrelated to the move. It is worth a separate ticket after
this lands.

The criterion stays unticked. Half of it is evidence that cannot exist yet: the
after-image comparison needs the move to have happened.

**agent:terva/mieli** at 2026-09-11T23:34:45Z

Step 2 is done, on `refactor/interchange-library` at `16728ae`, pushed to
`origin`. Steps 3 through 6 are untouched.

`ticket/interchange.go` holds the wire format: `BlobSHA`, `AddedFileHunk`,
`MboxMessage`, `MboxFromLine`, and `ParseAddedFiles` over an `AddedFile{Path,
Body}`. `cli/export.go` and `cli/import.go` call it and shed 151 lines between
them.

`TestExportMatchesThePinnedBeforeImage` passes unchanged, which is the claim
this step had to support.

Two decisions a reviewer should see rather than infer.

One signature did not move verbatim. `addedFileDiff` took a `*strings.Builder`
and returned a count, and `AddedFileHunk` returns the hunk and the count
instead. A Builder parameter is fine as an internal convenience and poor as
public API. The before-image is what made changing it safe, which is the
argument for pinning it first.

The seam sits at the hunk rather than at the whole patch, because the plan on
this ticket puts the cosmetic `plusBar` in `cli`. So `exportTicketFiles` still
renders the diffstat and calls the library for each hunk. If step 4 moves the
stat, `plusBar` goes with it and that ruling should be revisited in the open
rather than drifted past.

### A defect this step found and did not fix

`AddedFileHunk` writes git's `\ No newline at end of file` marker and hashes
the bytes as they are. `ParseAddedFiles` skips the marker and appends a newline
to every line it reads, so it rebuilds a file one byte longer, hashes that, and
returns "does not match its blob name; the patch has been altered or
truncated". Export writes an artifact its own import refuses, and the error
accuses the sender of altering something nobody touched.

It predates this move: both halves are the contribution's code, copied
verbatim. It is latent because `Render` always ends a ticket file with a
newline, so nothing the store writes takes this path. A hand-authored file
would, which matters once `create --file` (TKT-01M298MJ0) exists.

`TestAFileWithNoTrailingNewlineDoesNotRoundTrip` pins the behaviour and says in
its own failure message what to do when it starts passing. The fix needs a
decision between honouring the marker on the way in and refusing such a file on
the way out, and changing the wire format inside a move that must be
byte-identical is the wrong moment. It wants its own ticket, filed after this
branch lands so the store write does not ride into this PR.

### Where the next session starts

Step 3, the typed `Change`. The kinds are enumerated in the implementation plan
above. `reconcile` and `reconciled` in `cli/import.go` are still where the
contribution left them, and `reconciled.Changes` is still `[]string` of English.
