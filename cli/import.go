package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/terva-sh/git-ticket/ticket"
)

// import is the half of the export pair that git cannot do alone.
//
// Where the two stores share a series, nothing here is needed: `git am
// *.patch` applies the export and the tickets are correct on arrival, which is
// what the export tests assert. This command exists for the case that fails —
// a ticket whose series the receiving store does not declare. `unknown_series`
// is an error rather than a warning, and the prefix is inside an immutable ID,
// so the only repair is to file the ticket again under a series this store
// knows. That is what importing means here: not applying a patch, but adopting
// its contents.
//
// It does not try to work out whether an export came from your own project.
// An earlier version decided that from the series, and it was wrong: `TKT` is
// the default every store has, so two strangers almost always share it and the
// guess said "same project" for the common case. The tool cannot know, so it
// asks. Without --adopt nothing is written and both routes are named; with it,
// every ticket is filed afresh under this store's own series.
//
// It runs no git at all, and moves no branch. Plan 7.3 is explicit
// that a sync helper "must never silently push, merge, switch branches, or
// rewrite a worktree", and an import that ran `git am` onto a scratch branch
// would do three of those four. Tickets are written through the same locked,
// atomic path every other write uses, and the result is staged for an ordinary
// `git commit` by the person who asked for it.
//
// What is left in this file is the command: flags, wording, and exit status.
// The reconciliation itself is ticket.PlanImport and ticket.ApplyImport, so a
// host that is not a terminal reaches it without building argv, and the preview
// renders the same answer the write carries out rather than deriving a second
// one.

// runImport adopts the tickets an export carries into this store.
func runImport(ctx *cmdContext, args []string) error {
	var adopt bool
	var fromStore string
	rest, err := ctx.parseFlags("import", args, func(fs *flag.FlagSet) {
		fs.BoolVar(&adopt, "adopt", false, "file every ticket afresh under this store's series; without it nothing is written")
		fs.StringVar(&fromStore, "from-store", "", "the store this export came from, recorded as provenance")
	})
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return usageErr("import takes one export directory")
	}
	s, err := ctx.openStore()
	if err != nil {
		return err
	}
	dir := rest[0]

	patch, err := readExportPatch(dir)
	if err != nil {
		return err
	}

	// The zero actor for a preview, because it files nothing and asking for the
	// real one warns on stderr about a store that declares no default.
	actor := ticket.Actor{}
	if adopt {
		actor = ctx.actor(s)
	}

	plan, err := s.PlanImport(context.Background(), ticket.ImportOptions{
		Patch:     patch,
		FromStore: fromStore,
		Actor:     actor,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", filepath.Join(dir, exportTicketPatch), err)
	}
	if len(plan.Tickets) == 0 {
		return fmt.Errorf("%s carries no tickets; it may be a patch series rather than an export", dir)
	}

	if !adopt {
		return importPreview(ctx, s, dir, plan)
	}
	return importAdopt(ctx, s, plan)
}

// readExportPatch reads the ticket patch an export carries.
//
// The error names the file the command expects, because the likeliest way to
// arrive here is pointing import at a directory of patches that is not an
// export at all.
func readExportPatch(dir string) (string, error) {
	patch := filepath.Join(dir, exportTicketPatch)
	data, err := os.ReadFile(patch)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("no %s in %s; import takes a directory written by export", exportTicketPatch, dir)
		}
		return "", err
	}
	return string(data), nil
}

// changeLine is the CLI's voice for one reconciliation change.
//
// The library reports what happened and this decides how to say it. Each line
// reads as "subject: what happens" and carries no tense, so the preview and the
// write share one wording. Two tenses would be two wordings to keep in step.
//
// An unknown kind still prints. A library that grows a kind this binary has not
// learned would otherwise drop it in silence, which is the exact failure the
// change vocabulary exists to prevent, and TestEveryChangeKindHasALine keeps
// that fallback unreachable in this build.
func changeLine(c ticket.Change) string {
	switch c.Kind {
	case ticket.ChangeLabelDropped:
		return fmt.Sprintf("label %q: not carried, this store does not declare it", c.Value)
	case ticket.ChangeMilestoneDropped:
		return fmt.Sprintf("milestone %q: not carried, this store does not declare it", c.Value)
	case ticket.ChangeDueOnDropped:
		return fmt.Sprintf("due date %s: not carried, this store has not agreed to it", c.Value)
	case ticket.ChangeBlocksOnDropped:
		return fmt.Sprintf("blocks_on %s: not carried, the edges it gates on do not all travel", c.Value)
	case ticket.ChangeReferencePathDropped:
		return fmt.Sprintf("reference %s: the path is not carried, no such file in this repository", c.Value)
	case ticket.ChangeAcceptanceCriteriaUnchecked:
		return fmt.Sprintf("acceptance criteria: %d carried, every box unchecked", c.Count)
	case ticket.ChangeDefinitionOfDoneUnchecked:
		return fmt.Sprintf("definition of done: %d carried, every box unchecked", c.Count)
	case ticket.ChangeWorkRecordCarried:
		return "summary, notes and comments: carried as one note naming the origin"
	}
	if c.Value != "" {
		return fmt.Sprintf("%s %s: reported by the library, which this build does not have wording for", c.Kind, c.Value)
	}
	return fmt.Sprintf("%s: reported by the library, which this build does not have wording for", c.Kind)
}

// importPreview says what would happen and writes nothing.
//
// This is the default because an export is somebody else's content, and a
// command that ingests it without showing its hand first asks for trust it has
// not earned. --adopt is the sentence that grants it.
//
// It renders the plan and decides nothing. That is the guarantee the preview is
// making: what it shows is what --adopt will carry out, because both read the
// same ImportPlan rather than each working the answer out.
func importPreview(ctx *cmdContext, s *ticket.Store, dir string, plan *ticket.ImportPlan) error {
	cfg := s.Config()
	shared := 0
	for _, pt := range plan.Tickets {
		if series, _ := ticket.SplitID(pt.Incoming.ID); cfg.KnownSeries(series) {
			shared++
		}
	}

	fmt.Fprintf(ctx.out, "%s carries %s\n\n", dir, plural(len(plan.Tickets), "ticket"))
	for _, pt := range plan.Tickets {
		fmt.Fprintf(ctx.out, "  %s  %s\n", pt.Incoming.ID, pt.Incoming.Title)
		for _, c := range pt.Changes {
			fmt.Fprintf(ctx.out, "    %s\n", changeLine(c))
		}
		for _, d := range pt.DroppedEdges {
			fmt.Fprintf(ctx.out, "    %s: not carried, this export does not include it\n", d)
		}
	}

	fmt.Fprintf(ctx.out, "\nNothing written. --adopt files all %s afresh under this store's series.\n", plural(len(plan.Tickets), "ticket"))
	// Advice, not a verdict. Whether this export is your own project's work is
	// something only you know: the series cannot say, because TKT is the default
	// every store has and two strangers share it by default.
	if shared > 0 {
		fmt.Fprintf(ctx.out, "\n%s already use a series this store declares. If this export is your own\nproject's work, `git am %s` keeps the original IDs and is the better route.\nIf it came from elsewhere, --adopt is correct even though the series matches.\n",
			plural(shared, "ticket"), filepath.Join(dir, "*.patch"))
	}
	if extra := importOtherPatches(dir); len(extra) > 0 {
		fmt.Fprintf(ctx.env.Stderr, "%s also carries %s of code, which import does not apply: git am %s\n",
			dir, plural(len(extra), "patch"), filepath.Join(dir, "*.patch"))
	}
	return nil
}

// importAdopt files the plan and reports what landed.
//
// The write is one library call, so the edge rewriting and the ordering are not
// this file's business. What is left here is the report, and its subject is the
// ID this store minted, which is the one thing the sender's copy cannot tell
// the reader.
func importAdopt(ctx *cmdContext, s *ticket.Store, plan *ticket.ImportPlan) error {
	res, err := s.ApplyImport(context.Background(), plan)
	if err != nil {
		return err
	}

	for i, pt := range plan.Tickets {
		for _, c := range pt.Changes {
			fmt.Fprintf(ctx.env.Stderr, "  %s: %s\n", res.Filed[i].ID, changeLine(c))
		}
	}

	for i, pt := range plan.Tickets {
		filed := res.Filed[i]
		fmt.Fprintf(ctx.out, "%s  <- %s  %s\n", filed.ID, pt.Incoming.ID, pt.Incoming.Title)
		for _, dropped := range pt.DroppedEdges {
			fmt.Fprintf(ctx.env.Stderr, "  %s: %s not carried, this export does not include it\n", filed.ID, dropped)
		}
	}
	fmt.Fprintf(ctx.env.Stderr, "%s filed as draft. Review, then commit.\n", plural(len(res.Filed), "ticket"))
	return nil
}

// importOtherPatches names the code patches an export carries beside its
// tickets, so the preview can say they are somebody else's job.
func importOtherPatches(dir string) []string {
	all, err := filepath.Glob(filepath.Join(dir, "*.patch"))
	if err != nil {
		return nil
	}
	var out []string
	for _, p := range all {
		if filepath.Base(p) != exportTicketPatch {
			out = append(out, p)
		}
	}
	return out
}
