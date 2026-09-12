# Handoff: the interchange is library API now

For terva, or any host embedding `github.com/terva-sh/git-ticket/ticket`.

This supersedes nothing. `docs/handoff-terva-phase-3.md` and the three
`reply-terva-*.md` documents still say what they said; this one adds an API that
did not exist when they were written.

## What changed

`git ticket export` and `git ticket import` shipped in v0.15.0 with their whole
domain inside the `cli` package, which exports three symbols: `Env`, `UIParams`,
and `Run`. So the only way to reach the interchange was to build argv and read
prose.

For export that was survivable, because the `mutation-result` envelope names the
files it wrote. For import it was not. `import` has no `--json` form, so a host
got an exit code and English, and the map from each arriving ticket ID to the ID
the receiving store minted was in neither.

The interchange now lives in `ticket`. The command is flags, wording and exit
status over it.

## The API

```go
func (s *Store) Export(ctx context.Context, o ExportOptions) (*Export, error)
func (s *Store) PlanImport(ctx context.Context, o ImportOptions) (*ImportPlan, error)
func (s *Store) ApplyImport(ctx context.Context, p *ImportPlan) (*ImportResult, error)
```

### Export

```go
type ExportOptions struct {
	IDs  []string  // resolved like any ref, so a unique prefix works
	From string    // the From: header, as "Name <mail>"
	Now  time.Time // the Date: header instant
}

type Export struct {
	Cover   []byte    // the cover letter; not part of the series, do not name it *.patch
	Patch   []byte    // the one patch, adding the ticket files
	Tickets []*Ticket // resolved, so you can report without reading them again
}
```

It returns bytes. Where they land is yours to decide, which is what lets you
put an export on a wire instead of on a disk.

`From` is a parameter rather than something the library works out. It comes from
git config, and the library runs no git at all. The CLI reads it and passes it
in, and you should do the same rather than expecting a default: an export with
no identity says `unknown <unknown@localhost>`.

The cover is numbered `0/1` and the patch is `1`. Numbers from 2 up are free for
code you compose in with `git format-patch --start-number 2`, and a plain `git
am *.patch` on the receiving side then applies the tickets and the code
together. The cover is `.txt` precisely so that glob skips it.

### Import

`PlanImport` decides and writes nothing. `ApplyImport` carries out exactly what
the plan says and decides nothing. That split is the library-shaped version of
the `import` / `import --adopt` split a person sees, and it exists so a preview
is a promise rather than a second opinion.

```go
type ImportOptions struct {
	Patch     string // the export's 0001-tickets.patch, as bytes
	FromStore string // recorded as provenance; may be empty
	Actor     Actor  // recorded on everything ApplyImport writes
}

type ImportPlan struct {
	Tickets []PlannedTicket
	Actor   Actor
}

type PlannedTicket struct {
	Incoming     *Ticket        // as it arrived, for naming the sender's ID and title
	OriginPath   string         // where it sat in the sending store
	Create       CreateOptions  // what this store will file
	Refs         []AddReference // the sender's references, plus provenance
	Record       string         // the sender's summary, notes and comments as one note
	Changes      []Change       // what this store imposed
	DroppedEdges []string       // "dependency ID" / "parent ID" pointing outside the export
}

type ImportResult struct {
	Filed []ImportedTicket
}

type ImportedTicket struct {
	FromID string // the ID it arrived with
	ID     string // the ID this store minted
	Title  string
}
```

`Filed` is index-aligned with `plan.Tickets`, and `FromID` is there because that
mapping is the one thing you cannot work out afterwards.

`Patch` is bytes rather than a directory on purpose. An import can arrive over a
wire, and a decision you can test without a filesystem is worth having.

## Changes are values, not sentences

```go
type Change struct {
	Kind  ChangeKind
	Value string // the label, milestone, date, blocks_on value, or reference
	Count int    // the item count, for the checklist kinds
}

func ChangeKinds() []ChangeKind
```

The kinds are `ChangeLabelDropped`, `ChangeMilestoneDropped`,
`ChangeDueOnDropped`, `ChangeBlocksOnDropped`, `ChangeReferencePathDropped`,
`ChangeAcceptanceCriteriaUnchecked`, `ChangeDefinitionOfDoneUnchecked`, and
`ChangeWorkRecordCarried`.

`ChangeKinds` exists so you can prove you render all of them. The CLI has a test
that walks it and fails when a kind has no wording, and that test is worth
copying: a kind nobody prints is a loss the reader never hears about, which is
the failure this vocabulary exists to prevent.

Write your own wording. The CLI's is tuned for a terminal and carries no tense,
so one string serves both the preview and the report.

## The interchange rule

Worth stating, because it decides what your users will ask you about.

The statement of the work travels: title, type, priority, description,
implementation plan, both checklists, and assignees. What the receiver never
agreed to does not: a label or milestone this store does not declare, a
reference path that resolves to no file here, and a due date somebody else set.
Every one of those is reported as a `Change` rather than dropped in silence.

Checklists arrive with the sender's boxes ticked and land unticked. A tick is
evidence about the sender's work and says nothing about whether this store has
met the criterion. Expect that question.

The sender's summary, notes and comments arrive as one note naming the origin,
rather than as restored sections, because a note is honest about provenance: the
actors and instants inside it are the sending store's and nothing here can
verify either.

## The wire format, if you need it directly

```go
func BlobSHA(data []byte) string
func AddedFileHunk(path string, data []byte) (hunk string, lines int)
func MboxMessage(from string, when time.Time, subject, body, diff string) string
func ParseAddedFiles(patch string) ([]AddedFile, error)

const MboxFromLine = "From 00000000...0000 Mon Sep 17 00:00:00 2001"

type AddedFile struct {
	Path string // slash-spelled, relative to the repository root
	Body string
}
```

None of this runs git. `BlobSHA` computes the object name git would give the
content, which is what keeps the index line honest without shelling out, and
what lets an export be built where there is no repository at all.

One defect to know about if you generate files yourself. `AddedFileHunk` writes
git's `\ No newline at end of file` marker and hashes the bytes as they are,
while `ParseAddedFiles` skips that marker and appends a newline to every line,
so a file with no trailing newline does not round-trip: it comes back reporting
"does not match its blob name; the patch has been altered or truncated" when
nobody altered anything. It is latent for tickets, because `Render` always ends
a ticket file with a newline. `TestAFileWithNoTrailingNewlineDoesNotRoundTrip`
pins it. Do not feed `AddedFileHunk` a file that does not end in a newline.

## Worked example

```go
package example

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/terva-sh/git-ticket/ticket"
)

// Send builds an export and hands back the two files to write or transmit.
func Send(ctx context.Context, repo string, ids []string, from string) (cover, patch []byte, err error) {
	s, err := ticket.Discover(repo)
	if err != nil {
		return nil, nil, err
	}
	art, err := s.Export(ctx, ticket.ExportOptions{
		IDs:  ids,
		From: from,
		Now:  time.Now(),
	})
	if err != nil {
		return nil, nil, err
	}
	return art.Cover, art.Patch, nil
}

// Adopt shows an export's reconciliation, then files it.
func Adopt(ctx context.Context, repo, patchPath string, actor ticket.Actor) error {
	s, err := ticket.Discover(repo)
	if err != nil {
		return err
	}
	patch, err := os.ReadFile(patchPath)
	if err != nil {
		return err
	}

	plan, err := s.PlanImport(ctx, ticket.ImportOptions{
		Patch:     string(patch),
		FromStore: "the-other-project",
		Actor:     actor,
	})
	if err != nil {
		return err
	}

	// Nothing has been written yet. Show the person what will happen.
	for _, pt := range plan.Tickets {
		fmt.Printf("%s  %s\n", pt.Incoming.ID, pt.Incoming.Title)
		for _, c := range pt.Changes {
			fmt.Printf("    %s\n", describe(c))
		}
		for _, edge := range pt.DroppedEdges {
			fmt.Printf("    %s: points outside this export\n", edge)
		}
	}

	res, err := s.ApplyImport(ctx, plan)
	if err != nil {
		// Not atomic. Tickets before the failure are already filed.
		return err
	}
	for _, f := range res.Filed {
		fmt.Printf("%s was %s  %s\n", f.ID, f.FromID, f.Title)
	}
	return nil
}

// describe is your voice, not the library's.
func describe(c ticket.Change) string {
	switch c.Kind {
	case ticket.ChangeLabelDropped:
		return fmt.Sprintf("label %q is not in this store's allowlist", c.Value)
	case ticket.ChangeMilestoneDropped:
		return fmt.Sprintf("milestone %q is not in this store's allowlist", c.Value)
	case ticket.ChangeDueOnDropped:
		return fmt.Sprintf("due date %s belongs to the sender", c.Value)
	case ticket.ChangeBlocksOnDropped:
		return fmt.Sprintf("blocks_on %s gates on edges that did not travel", c.Value)
	case ticket.ChangeReferencePathDropped:
		return fmt.Sprintf("reference %s keeps its ref and loses its path", c.Value)
	case ticket.ChangeAcceptanceCriteriaUnchecked:
		return fmt.Sprintf("%d acceptance criteria, every box unchecked", c.Count)
	case ticket.ChangeDefinitionOfDoneUnchecked:
		return fmt.Sprintf("%d definition-of-done items, every box unchecked", c.Count)
	case ticket.ChangeWorkRecordCarried:
		return "the sender's summary, notes and comments arrive as one note"
	}
	return string(c.Kind)
}
```

## Things that will bite

`ApplyImport` is not atomic, and the plan is explicit that this is a decision
rather than an oversight. If a create fails on the third of five tickets, the
first two are filed and the error names the third. Recovery is reading what
landed and removing it. Making it transactional needs a scratch branch, and
plan 7.3 forbids a helper that rewrites a worktree.

The error path also returns no partial result today, so a caller cannot report
what did land from the return value alone. If that matters to you, say so and it
can return the partial `ImportResult` alongside the error.

Every imported ticket is filed as a draft, whatever status it had at the sender.

`ApplyImport` takes no revision precondition. There is nothing to precondition:
every ticket it touches is one it created moments earlier.

## Status of this document

The worked example is not illustrative. It was extracted to its own module and
built against the published release, with no `replace` of any kind:

```text
module example.test/handoff
require github.com/terva-sh/git-ticket v0.16.0

$ go build ./... && go vet ./...   # both clean
```

So every field name and signature above is one the compiler accepted against
bytes the module proxy served, rather than against a working tree.

**The version floor is v0.16.0.** That is the first release carrying `Export`,
`PlanImport` and `ApplyImport`, so pin to it or later.

The published module was checked against the tag rather than taken on trust,
because `go list -m` answers from the local module cache and can report a hash
that was never published:

```text
$ curl -sS https://proxy.golang.org/github.com/terva-sh/git-ticket/@v/v0.16.0.info
{"Version":"v0.16.0", ... "Hash":"3820f842dc5ad72c41bdd4859de8dac16d503096", ...}

$ git rev-parse v0.16.0^{commit}
3820f842dc5ad72c41bdd4859de8dac16d503096
```

An earlier version of this section said the floor was unsettled and told you not
to pin. That was written before any release carried the API, and it is
superseded by the two paragraphs above.
