package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/terva-sh/git-ticket/ticket"
)

type mappingFlags []string

func (f *mappingFlags) String() string { return strings.Join(*f, ",") }
func (f *mappingFlags) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func parseMappingFlags(values mappingFlags) ([]ticket.MappingSelection, error) {
	choices := make([]ticket.MappingSelection, 0, len(values))
	for _, value := range values {
		parts := strings.SplitN(value, "=", 2)
		if len(parts) != 2 || parts[0] == "" {
			return nil, fmt.Errorf("--map needs NAMESPACE=adopt|alias:LOCAL|decline[:LOCAL]")
		}
		choice := ticket.MappingSelection{Namespace: parts[0]}
		action := strings.SplitN(parts[1], ":", 2)
		choice.Action = action[0]
		if len(action) == 2 {
			choice.Local = action[1]
		}
		choices = append(choices, choice)
	}
	return choices, nil
}

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
	var adopt, adoptMappings, sameOwner bool
	var fromStore, ifMapRevision string
	var maps mappingFlags
	rest, err := ctx.parseFlags("import", args, func(fs *flag.FlagSet) {
		fs.BoolVar(&adopt, "adopt", false, "file every ticket afresh under this store's series; without it nothing is written")
		fs.BoolVar(&adoptMappings, "adopt-mappings", false, "write explicitly accepted portable reference mappings")
		fs.Var(&maps, "map", "choose NAMESPACE=adopt|alias:LOCAL|decline[:LOCAL]; repeat for each mapping")
		fs.StringVar(&ifMapRevision, "if-map-revision", "", "require the current reference registry revision")
		fs.BoolVar(&sameOwner, "same-owner", false, "both stores are yours, so carry the ticks, the status and the filing instant")
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
	var sidecar []byte
	sidecar, err = os.ReadFile(filepath.Join(dir, exportReferenceLookup))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if os.IsNotExist(err) {
		sidecar = nil // exports before references.json remain importable
	}
	choices, err := parseMappingFlags(maps)
	if err != nil {
		return err
	}
	mappingPlan, err := s.PlanReferenceMappings([]byte(patch), sidecar, choices)
	if err != nil {
		return err
	}
	if adopt && len(mappingPlan.Unresolved) > 0 {
		return fmt.Errorf("conflicting references need --map NAMESPACE=decline:LOCAL (or alias:LOCAL for offered mappings) before ticket adoption: %s", strings.Join(mappingPlan.Unresolved, ", "))
	}
	if adopt && !adoptMappings && mappingPlan.Changed {
		return fmt.Errorf("selected mappings need --adopt-mappings before ticket adoption")
	}

	// The zero actor for a preview, because it files nothing and asking for the
	// real one warns on stderr about a store that declares no default.
	actor := ticket.Actor{}
	if adopt {
		actor = ctx.actor(s)
	}

	plan, err := s.PlanImport(context.Background(), ticket.ImportOptions{
		Patch:             patch,
		ReferenceRewrites: mappingPlan.Rewrites,
		ReferenceRegistry: &mappingPlan.Registry,
		FromStore:         fromStore,
		SameOwner:         sameOwner,
		Actor:             actor,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", filepath.Join(dir, exportTicketPatch), err)
	}
	if len(plan.Tickets) == 0 {
		return fmt.Errorf("%s carries no tickets; it may be a patch series rather than an export", dir)
	}
	if plan.MapRevision != mappingPlan.MapRevision {
		return fmt.Errorf("reference registry changed while import was planned; preview again")
	}
	mappingsChanged := false
	if adoptMappings {
		result, err := s.ApplyReferenceMappings(context.Background(), mappingPlan, ifMapRevision)
		if err != nil {
			return err
		}
		if result.Changed {
			mappingsChanged = true
			fmt.Fprintf(ctx.env.Stderr, "adopted reference mappings in .tickets/references.yml (revision %s)\n", result.MapRevision)
		} else {
			fmt.Fprintf(ctx.env.Stderr, "reference mappings unchanged (revision %s)\n", result.MapRevision)
		}
		plan.MapRevision = result.MapRevision
	} else if ifMapRevision != "" && ifMapRevision != mappingPlan.MapRevision {
		return fmt.Errorf("reference registry revision does not match --if-map-revision")
	}

	// --same-owner is legal on its own. It changes what the reconciliation
	// decides, so a preview that ignored it would show a different answer from
	// the one --adopt carries out, and the preview's whole promise is that it
	// does not.
	if !adopt {
		return importPreview(ctx, s, dir, plan, mappingPlan, sameOwner, mappingsChanged)
	}
	return importAdopt(ctx, s, plan, sameOwner)
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
	case ticket.ChangeAcceptanceCriteriaCarried:
		return fmt.Sprintf("acceptance criteria: %d ticked at the origin, carried ticked", c.Count)
	case ticket.ChangeDefinitionOfDoneCarried:
		return fmt.Sprintf("definition of done: %d ticked at the origin, carried ticked", c.Count)
	case ticket.ChangeStatusCarried:
		return fmt.Sprintf("status %s: carried, this is the same owner's finished work", c.Value)
	case ticket.ChangeStatusNotCarried:
		return fmt.Sprintf("status %s: not carried, it lands in draft because only done and archived may arrive directly", c.Value)
	case ticket.ChangeOriginParentRecorded:
		return fmt.Sprintf("parent %s: kept as an origin-parent reference, the edge cannot name a ticket this store does not have", c.Value)
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
func importPreview(ctx *cmdContext, s *ticket.Store, dir string, plan *ticket.ImportPlan, mappings *ticket.ReferenceMappingPlan, sameOwner, mappingsWritten bool) error {
	cfg := s.Config()
	shared := 0
	for _, pt := range plan.Tickets {
		if series, _ := ticket.SplitID(pt.Incoming.ID); cfg.KnownSeries(series) {
			shared++
		}
	}

	fmt.Fprintf(ctx.out, "%s carries %s\n\n", dir, plural(len(plan.Tickets), "ticket"))
	if mappings.HasSidecar {
		fmt.Fprintln(ctx.out, "Reference lookup offered (destinations are claims by the sender):")
		for _, offer := range mappings.Offers {
			receiverLabel := "receiver"
			if mappingsWritten {
				receiverLabel = "receiver before mapping write"
			}
			fmt.Fprintf(ctx.out, "  %s  %s -> %s\n    %s: %s; choice: %s", offer.Namespace, offer.Example, offer.Destination, receiverLabel, offer.Existing, offer.Action)
			if offer.Local != "" {
				fmt.Fprintf(ctx.out, ":%s", offer.Local)
			}
			fmt.Fprintln(ctx.out)
			if mappingsWritten {
				switch offer.Action {
				case "adopt":
					if offer.Existing == "absent" {
						fmt.Fprintf(ctx.out, "    receiver now: mapping installed as %s\n", offer.Namespace)
					}
				case "alias":
					fmt.Fprintf(ctx.out, "    receiver now: mapping installed as %s\n", offer.Local)
				}
			}
			if offer.Existing == "identical" && offer.Action == "decline" {
				fmt.Fprintln(ctx.out, "    identical local mapping remains active; decline copies nothing")
			}
		}
		for _, name := range mappings.Undeclared {
			if local, renamed := mappings.Rewrites[name]; renamed {
				fmt.Fprintf(ctx.out, "  %s: used but undeclared; renamed to opaque %s\n", name, local)
			} else {
				fmt.Fprintf(ctx.out, "  %s: used but undeclared; remains opaque when no local declaration exists\n", name)
			}
		}
		for _, name := range mappings.Unresolved {
			fmt.Fprintf(ctx.out, "  %s: choose decline:LOCAL before --adopt (alias:LOCAL is also available for offered mappings)\n", name)
		}
		fmt.Fprintln(ctx.out)
	} else if len(mappings.Undeclared) > 0 {
		fmt.Fprintln(ctx.out, "No reference lookup was supplied by this older export; reference destinations are unknown.")
		for _, name := range mappings.Unresolved {
			fmt.Fprintf(ctx.out, "  %s: local declaration exists; choose --map %s=decline:LOCAL before --adopt\n", name, name)
		}
		fmt.Fprintln(ctx.out)
	}
	for _, pt := range plan.Tickets {
		fmt.Fprintf(ctx.out, "  %s  %s\n", pt.Incoming.ID, pt.Incoming.Title)
		for _, c := range pt.Changes {
			fmt.Fprintf(ctx.out, "    %s\n", changeLine(c))
		}
		for _, d := range pt.DroppedEdges {
			fmt.Fprintf(ctx.out, "    %s: not carried, this export does not include it\n", d)
		}
	}

	if mappingsWritten {
		fmt.Fprintf(ctx.out, "\nNo tickets written. --adopt files all %s afresh under this store's series.\n", plural(len(plan.Tickets), "ticket"))
	} else {
		fmt.Fprintf(ctx.out, "\nNothing written. --adopt files all %s afresh under this store's series.\n", plural(len(plan.Tickets), "ticket"))
	}
	// Named here rather than left for the adopt to reveal, because the commands
	// are the half of the move this tool will not perform, and a person deciding
	// whether to adopt should know the origin is still theirs to close.
	if sameOwner {
		fmt.Fprintf(ctx.out, "--same-owner also prints the commands that close each ticket at the origin,\nwhich only you can run: nothing here writes the sending store.\n")
	}
	// Advice, not a verdict. Whether this export is your own project's work is
	// something only you know: the series cannot say, because TKT is the default
	// every store has and two strangers share it by default.
	if shared > 0 {
		fmt.Fprintf(ctx.out, "\n%s already %s a series this store declares. If this export is your own\nproject's work, `git am %s` keeps the original IDs and is the better route.\nIf it came from elsewhere, --adopt is correct even though the series matches.\n",
			plural(shared, "ticket"), verb(shared, "uses", "use"), filepath.Join(dir, "*.patch"))
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
func importAdopt(ctx *cmdContext, s *ticket.Store, plan *ticket.ImportPlan, sameOwner bool) error {
	res, err := s.ApplyImport(context.Background(), plan)
	if err != nil {
		if res != nil {
			for _, filed := range res.Filed {
				fmt.Fprintf(ctx.env.Stderr, "filed before error: %s <- %s  %s\n", filed.ID, filed.FromID, filed.Title)
				fmt.Fprintf(ctx.env.Stderr, "  inspect: git ticket show %s\n", filed.ID)
			}
		}
		fmt.Fprintln(ctx.env.Stderr, "recover: git ticket check --strict; inspect the filed tickets before retrying import")
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
	// "as draft" is a promise the default keeps and --same-owner does not, since
	// a ticket that left done arrives done. The default's wording does not move.
	if sameOwner {
		fmt.Fprintf(ctx.env.Stderr, "%s filed. Review, then commit.\n", plural(len(res.Filed), "ticket"))
		importCloseOrigin(ctx, plan, res)
	} else {
		fmt.Fprintf(ctx.env.Stderr, "%s filed as draft. Review, then commit.\n", plural(len(res.Filed), "ticket"))
	}
	return nil
}

// importCloseOrigin prints the commands that close each ticket at the origin.
//
// Advice, and never the action. The sending store is another working tree, and
// 7.3 is explicit that a sync helper must never rewrite one; a command that
// reached across and edited a second repository would also be exactly the
// collision the no-sibling-writes policy exists to prevent. So the person who
// knows both stores runs these, in the store they are about.
//
// It is the same move the preview already makes when it offers `git am`: name
// the better route and let the reader take it.
func importCloseOrigin(ctx *cmdContext, plan *ticket.ImportPlan, res *ticket.ImportResult) {
	fmt.Fprintf(ctx.env.Stderr, "\nNothing here wrote the sending store. To close the origin, run these there:\n")
	for i, pt := range plan.Tickets {
		filed := res.Filed[i]
		fmt.Fprintf(ctx.env.Stderr, "  git ticket summary %s \"Moved to %s.\"\n", pt.Incoming.ID, filed.ID)
		// Only when there is a transition left to make. A ticket that arrived
		// done or archived is already closed, and printing a status command for
		// it would be advice that fails when taken.
		if st := pt.Incoming.Status; st != ticket.StatusDone && st != ticket.StatusArchived {
			fmt.Fprintf(ctx.env.Stderr, "  git ticket status %s done\n", pt.Incoming.ID)
		}
	}
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
