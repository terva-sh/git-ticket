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
updated_at: 2026-09-11T23:57:13Z
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

**agent:terva/mieli** at 2026-09-11T23:43:26Z

Step 3 is done, at `cec1dca`. Steps 4 through 6 remain.

`ticket.Change` carries `Kind`, `Value` and `Count`. The eight kinds are a
dropped label, milestone, due date, `blocks_on` and reference path, each
checklist carried unticked, and the work record. `ChangeKinds()` lists them.

`changeLine` in `cli/import.go` is where English lives from here on, and the
wording did not move. The existing import tests are what prove that: they
assert the exact sentences in both the preview and the adopt report, and they
passed untouched.

Two design points that are decisions rather than mechanics.

The checklists got two kinds rather than one kind with a field naming which
checklist. A consumer would have had to switch on that field anyway, so one
kind per sentence keeps the renderer a plain switch.

`changeLine` has a fallback for a kind it does not know, and
`TestEveryChangeKindHasALine` keeps that fallback unreachable in this build.
The pair is deliberate. An older binary meeting a newer library should say the
kind badly rather than drop it, since a silent drop is the exact failure this
vocabulary exists to prevent, while a kind added here and not given wording
should fail the suite rather than reach a reader.

`TestChangeLineHasNoTense` pins the property the `reconciled` doc comment
claims: one wording serves the preview, which has filed nothing, and the
report, which has. It is a string match over prose, so it is the test most
likely to need adjusting when somebody rewords a line. That is the intended
cost. "carried" is deliberately not in its word list, because it is doing
adjective work in "2 carried, every box unchecked".

### Where step 4 starts

`reconcile` still lives in `cli/import.go` and still takes `cfg`, `root`,
`incomingTicket`, `fromStore` and `actor`. It now returns typed changes, so
moving it to `PlanImport` no longer drags a wording decision across the
package boundary, which was the point of doing step 3 first.

**agent:terva/mieli** at 2026-09-11T23:53:12Z

Step 4 is done at `1bc2eef`, and step 5 came with it. Steps 6 remains: the
plan text and the terva handoff.

Step 5 could not be left behind. `PlanImport` cannot order an export without
`importOrder` and cannot reconcile without `keepKnownLabels`, so splitting them
would have meant a commit that does not build.

`ticket.PlanImport` decides and writes nothing. `ticket.ApplyImport` carries
out exactly what the plan says. `reconcile`, `importOrder`, `keepKnownLabels`,
`repoHasPath`, `originRecord` and `importDroppedEdges` all moved.
`cli/import.go` lost 411 lines and is now flags, wording and exit status.

Nothing a person sees moved. The import tests assert the exact preview and
adopt wording and passed untouched, and so did the export before-image.

### Decisions in this step

`PlanImport` takes the patch as bytes rather than a directory. That is what
lets an import arrive off a wire, and it makes the library testable with no
filesystem. The CLI keeps `readExportPatch` because the error naming
`0001-tickets.patch` is about pointing the command at the wrong directory,
which is a CLI concern.

`ImportResult` carries `FromID` beside `ID`. The map from an arriving ID to the
minted one is the single thing a caller cannot reconstruct, and its absence was
the concrete argument in this ticket's description.

The preview renders `plan.Tickets` and derives nothing. That is what closes the
drift this ticket names: preview and adopt had already diverged over labels
once.

### Two behaviour changes, both on paths that were already wrong

`--if-revision` no longer reaches the writes an import makes. It was being
passed to mutations on tickets created moments earlier, so it could only ever
fail. The library takes no revision precondition here.

A failed import no longer prints the changes for tickets it had already filed,
because the report now runs after the write rather than inside it. The error
still names the ticket that failed. This is a loss of information on the error
path and it is worth a second opinion.

### On the before-image

It has now held across three steps. Worth saying plainly: it only ever proved
the export artifact. The import side is guarded by the existing CLI tests
asserting exact wording, which is weaker, and the new library tests, which are
about the API rather than the output. If step 6 or a later change wants the
same confidence for import, the preview output is what to pin.

**agent:terva/mieli** at 2026-09-11T23:57:13Z

Step 6 is done at `617a646`, and with it every step of the implementation plan.
The ticket is not done, and the gap is mine.

The plan's six steps and the eight acceptance criteria do not describe the same
work. The steps move the wire format and then the whole import side. Nothing in
them builds `Store.Export`, and three criteria ask for it:

- "ticket exports Export, PlanImport and ApplyImport". There is no
  `ticket.Export`. `PlanImport` and `ApplyImport` exist.
- "Export returns the artifact as bytes; writing a directory is the CLI's
  decision". Not built.
- "cli/export.go and cli/import.go hold flags, printing and exit status only".
  `cli/import.go` does, at 233 lines. `cli/export.go` is 320 and still holds
  `exportSubject`, `exportCommitBody`, `exportTicketFiles` and `exportCover`,
  which compose the artifact and are not presentation.

The five that do hold: the reconciliation is computed once and rendered twice,
changes are typed, the export is byte-identical across the move, plan 12.2 and
12.8 say the library owns the interchange, and the handoff exists with compiled
code.

I should have caught this when I claimed the ticket and read both lists
together. Writing the plan's step 2 as "move the wire format" rather than "build
`Store.Export`" is what hid it, and every step after that was import work, so
nothing brought me back to the export half.

The criteria are the ask and they stand as written. What is left is one more
step of the same shape as step 4, on the other side:

`Store.Export(ctx, ExportOptions) (*Export, error)` returning the cover letter
and the patch as bytes. `exportTicketFiles`, `exportSubject`, `exportCommitBody`
and `exportCover` move, and `plusBar` goes with the diffstat, which retires the
ruling in this ticket's description that put it in `cli`. What stays is
`runExport`, `exportDirIsFree`, `warnDanglingEdges`, and `exportIdentity`, the
last because it reaches git through `cli`'s own `readGit`.

The before-image is already in place for exactly this, and it is the reason that
step is cheap: it proves the bytes did not move, which is the only hard part.

The eighth criterion is the separate case of evidence that cannot exist yet. The
handoff's example compiles, but against this tree through a `replace`, because
no release carries `PlanImport`. Compiling it without the `replace` belongs to
the verification run of the release that ships this.
