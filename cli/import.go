package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

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

	incoming, err := readExportTickets(rest[0])
	if err != nil {
		return err
	}
	if len(incoming) == 0 {
		return fmt.Errorf("%s carries no tickets; it may be a patch series rather than an export", rest[0])
	}

	ordered, err := importOrder(incoming)
	if err != nil {
		return err
	}

	if !adopt {
		return importPreview(ctx, s, rest[0], ordered, fromStore)
	}
	return importAdopt(ctx, s, ordered, fromStore)
}

// incomingTicket is one ticket read out of an export.
type incomingTicket struct {
	Ticket *ticket.Ticket
	// Path is where it sat in the sending store, which is the only thing that
	// says what its status was without trusting the frontmatter twice.
	Path string
}

// readExportTickets reads the ticket files an export carries.
//
// It parses the export's own patch rather than any patch: the format is this
// project's, produced by the command next door, and a general patch parser is a
// large thing to own for no gain. A file that is not in that shape is refused by
// name rather than guessed at.
func readExportTickets(dir string) ([]incomingTicket, error) {
	patch := filepath.Join(dir, exportTicketPatch)
	data, err := os.ReadFile(patch)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no %s in %s; import takes a directory written by export", exportTicketPatch, dir)
		}
		return nil, err
	}
	files, err := ticket.ParseAddedFiles(string(data))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", patch, err)
	}
	var out []incomingTicket
	for _, f := range files {
		t, err := ticket.Parse([]byte(f.Body))
		if err != nil {
			return nil, fmt.Errorf("%s: %s: %w", patch, f.Path, err)
		}
		out = append(out, incomingTicket{Ticket: t, Path: f.Path})
	}
	return out, nil
}

// importOrder puts a ticket after everything it points at, so that a dependency
// or a parent inside the same export is already filed, and its new ID is known,
// by the time the ticket naming it is written.
//
// Edges to tickets outside the export are not ordering constraints; they are
// dropped at write time and reported, since a reminted store cannot resolve them.
func importOrder(in []incomingTicket) ([]incomingTicket, error) {
	byID := make(map[string]incomingTicket, len(in))
	for _, t := range in {
		byID[t.Ticket.ID] = t
	}
	var ordered []incomingTicket
	state := make(map[string]int) // 0 unseen, 1 visiting, 2 done
	var visit func(string) error
	visit = func(id string) error {
		switch state[id] {
		case 2:
			return nil
		case 1:
			return fmt.Errorf("the export contains a dependency cycle at %s", id)
		}
		state[id] = 1
		cur := byID[id]
		edges := append([]string{}, cur.Ticket.Dependencies...)
		if cur.Ticket.Parent != nil && *cur.Ticket.Parent != "" {
			edges = append(edges, *cur.Ticket.Parent)
		}
		sort.Strings(edges)
		for _, e := range edges {
			if _, ok := byID[e]; ok {
				if err := visit(e); err != nil {
					return err
				}
			}
		}
		state[id] = 2
		ordered = append(ordered, cur)
		return nil
	}
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if err := visit(id); err != nil {
			return nil, err
		}
	}
	return ordered, nil
}

// reconciled is what adopting one incoming ticket would do to it: the fields to
// file it with, the work record to carry, the references to attach, and every
// change this store imposes on the way in.
//
// Preview and adopt both read this rather than each working it out. They
// duplicated the label rule once already, and a preview that promises one thing
// while the write does another is worse than showing nothing at all.
type reconciled struct {
	Create ticket.CreateOptions
	// Record is the sending store's summary, notes and comments gathered into
	// one note, or empty when it carried none.
	Record string
	// Refs are the sender's references with an unresolvable path stripped,
	// followed by the provenance ones.
	Refs []ticket.AddReference
	// Changes names what this store imposed. Each reads as "subject: what
	// happens" and carries no tense, so one wording serves the preview and the
	// write. Two tenses would be two wordings to keep in step.
	Changes []string
}

// reconcile decides how one incoming ticket becomes a ticket of this store.
//
// One rule runs through it: the statement of the work travels, and what the
// receiver never agreed to does not. Title, type, priority, description, plan
// and both checklists are the work. A label or milestone this store does not
// declare, a reference path that resolves to nothing here, and a deadline
// somebody else set are not. Each of those is named rather than dropped in
// silence, because a quiet loss is the failure that costs the reader their trust
// in everything else the command said.
//
// The checklists arrive with the sender's boxes ticked and land unticked. A tick
// is evidence about the sender's work, and it says nothing about whether this
// store has met the criterion. That is the same argument that files every
// imported ticket as a draft.
func reconcile(cfg ticket.Config, root string, in incomingTicket, fromStore string, actor ticket.Actor) reconciled {
	t := in.Ticket
	var r reconciled

	labels, droppedLabels := keepKnownLabels(cfg, t.Labels)
	for _, l := range droppedLabels {
		r.Changes = append(r.Changes, fmt.Sprintf("label %q: not carried, this store does not declare it", l))
	}

	r.Create = ticket.CreateOptions{
		Title:              t.Title,
		Type:               t.Type,
		Priority:           t.Priority,
		Labels:             labels,
		Assignees:          t.Assignees,
		Description:        t.Body.Description,
		ImplementationPlan: t.Body.ImplementationPlan,
		AcceptanceCriteria: ticket.ChecklistItems(t.Body.AcceptanceCriteria),
		DefinitionOfDone:   ticket.ChecklistItems(t.Body.DefinitionOfDone),
		Actor:              actor,
	}
	if n := len(r.Create.AcceptanceCriteria); n > 0 {
		r.Changes = append(r.Changes, fmt.Sprintf("acceptance criteria: %d carried, every box unchecked", n))
	}
	if n := len(r.Create.DefinitionOfDone); n > 0 {
		r.Changes = append(r.Changes, fmt.Sprintf("definition of done: %d carried, every box unchecked", n))
	}

	// A milestone is an allowlisted vocabulary exactly as a label is, so it is
	// reconciled the same way rather than by a second rule.
	if t.Milestone != nil && *t.Milestone != "" {
		if cfg.KnownMilestone(*t.Milestone) {
			m := *t.Milestone
			r.Create.Milestone = &m
		} else {
			r.Changes = append(r.Changes, fmt.Sprintf("milestone %q: not carried, this store does not declare it", *t.Milestone))
		}
	}
	if t.DueOn != nil && *t.DueOn != "" {
		r.Changes = append(r.Changes, fmt.Sprintf("due date %s: not carried, this store has not agreed to it", *t.DueOn))
	}
	// blocks_on gates a ticket on edges that mostly did not survive the remint,
	// so it is left at the default. Saying so costs one line and only appears
	// when the sender set it to something.
	if t.BlocksOn != "" && t.BlocksOn != ticket.BlocksOnNone {
		r.Changes = append(r.Changes, fmt.Sprintf("blocks_on %s: not carried, the edges it gates on do not all travel", t.BlocksOn))
	}

	for _, ref := range t.References {
		keep := ticket.AddReference{Ref: ref.Ref, Path: ref.Path}
		if keep.Path != nil && *keep.Path != "" && !repoHasPath(root, *keep.Path) {
			// The reference survives without its path. What the sender pointed
			// at is still worth knowing, and only the path is the thing that
			// resolves to nothing here.
			r.Changes = append(r.Changes, fmt.Sprintf("reference %s: the path is not carried, no such file in this repository", keep.Ref))
			keep.Path = nil
		}
		r.Refs = append(r.Refs, keep)
	}

	// Provenance is a reference and never origin. `origin` is checked against
	// this store and an unresolvable one is an error, which is the guarantee it
	// exists to carry; a reference is deliberately not checked, which is exactly
	// what a foreign ID needs.
	r.Refs = append(r.Refs, ticket.AddReference{Ref: "origin-ticket:" + t.ID})
	if fromStore != "" {
		r.Refs = append(r.Refs, ticket.AddReference{Ref: "origin-store:" + fromStore})
	}

	if r.Record = originRecord(t, fromStore); r.Record != "" {
		r.Changes = append(r.Changes, "summary, notes and comments: carried as one note naming the origin")
	}
	return r
}

// originRecord gathers the sending store's work record into one note.
//
// This is the reasoning, the alternatives that were tried, and what the sender
// concluded. On a handoff it is often worth more than the description. Filing
// each entry as a note of this store would restamp it with whoever ran import
// and with the instant they ran it, which invents an attribution; dropping the
// lot loses the most valuable half of the ticket. So it travels whole, in one
// note that says where it came from and declines to vouch for it.
//
// The headings are ### rather than ##, because parse splits a section on a line
// beginning "## " and this text is going inside one. The carried text cannot
// contain such a line itself: had it done so, parse would have made it a section
// of the sender's ticket and it would not be in Notes to be carried.
func originRecord(t *ticket.Ticket, fromStore string) string {
	blocks := []struct{ title, body string }{
		{"Summary at the origin", t.Body.Summary},
		{"Notes at the origin", t.Body.Notes},
		{"Comments at the origin", t.Body.Comments},
	}
	var b strings.Builder
	for _, blk := range blocks {
		if strings.TrimSpace(blk.body) == "" {
			continue
		}
		fmt.Fprintf(&b, "\n### %s\n\n%s\n", blk.title, strings.TrimSpace(blk.body))
	}
	if b.Len() == 0 {
		return ""
	}
	where := "another store"
	if fromStore != "" {
		where = fromStore
	}
	return fmt.Sprintf("The work record this ticket arrived with, from %s as %s.\n"+
		"The actors and the instants below are the sending store's, and nothing here\n"+
		"can verify either of them.\n%s", where, t.ID, b.String())
}

// importPreview says what would happen and writes nothing.
//
// This is the default because an export is somebody else's content, and a
// command that ingests it without showing its hand first asks for trust it has
// not earned. --adopt is the sentence that grants it.
func importPreview(ctx *cmdContext, s *ticket.Store, dir string, in []incomingTicket, fromStore string) error {
	cfg := s.Config()
	shared := 0
	for _, t := range in {
		if series, _ := ticket.SplitID(t.Ticket.ID); cfg.KnownSeries(series) {
			shared++
		}
	}

	fmt.Fprintf(ctx.out, "%s carries %s\n\n", dir, plural(len(in), "ticket"))
	for _, t := range in {
		fmt.Fprintf(ctx.out, "  %s  %s\n", t.Ticket.ID, t.Ticket.Title)
		// The zero actor, because a preview files nothing and asking for the
		// real one warns on stderr about a store that declares no default.
		for _, c := range reconcile(cfg, s.Root(), t, fromStore, ticket.Actor{}).Changes {
			fmt.Fprintf(ctx.out, "    %s\n", c)
		}
		for _, d := range importDroppedEdges(t, in) {
			fmt.Fprintf(ctx.out, "    %s: not carried, this export does not include it\n", d)
		}
	}

	fmt.Fprintf(ctx.out, "\nNothing written. --adopt files all %s afresh under this store's series.\n", plural(len(in), "ticket"))
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

// importAdopt files each ticket afresh, rewriting the edges among them to the
// IDs this store just minted and reconciling the vocabulary it does not share.
func importAdopt(ctx *cmdContext, s *ticket.Store, in []incomingTicket, fromStore string) error {
	cfg := s.Config()
	actor := ctx.actor(s)
	remap := make(map[string]string, len(in))
	var filed []string
	for _, t := range in {
		r := reconcile(cfg, s.Root(), t, fromStore, actor)
		for _, d := range t.Ticket.Dependencies {
			if to, ok := remap[d]; ok {
				r.Create.Dependencies = append(r.Create.Dependencies, to)
			}
		}
		if t.Ticket.Parent != nil {
			if to, ok := remap[*t.Ticket.Parent]; ok {
				p := to
				r.Create.Parent = &p
			}
		}
		res, err := s.Create(context.Background(), r.Create)
		if err != nil {
			return fmt.Errorf("filing %s (%s): %w", t.Ticket.Title, t.Ticket.ID, err)
		}
		remap[t.Ticket.ID] = res.Ticket.ID
		filed = append(filed, res.Ticket.ID)

		for _, ref := range r.Refs {
			if _, err := ctx.applyTo(s, res.Ticket.ID, ref); err != nil {
				return fmt.Errorf("carrying reference %s to %s: %w", ref.Ref, res.Ticket.ID, err)
			}
		}
		if r.Record != "" {
			if _, err := ctx.applyTo(s, res.Ticket.ID, ticket.AppendNote{Text: r.Record}); err != nil {
				return fmt.Errorf("carrying the work record to %s: %w", res.Ticket.ID, err)
			}
		}
		for _, c := range r.Changes {
			fmt.Fprintf(ctx.env.Stderr, "  %s: %s\n", res.Ticket.ID, c)
		}
	}

	for i, t := range in {
		fmt.Fprintf(ctx.out, "%s  <- %s  %s\n", filed[i], t.Ticket.ID, t.Ticket.Title)
		for _, dropped := range importDroppedEdges(t, in) {
			fmt.Fprintf(ctx.env.Stderr, "  %s: %s not carried, this export does not include it\n", filed[i], dropped)
		}
	}
	fmt.Fprintf(ctx.env.Stderr, "%s filed as draft. Review, then commit.\n", plural(len(filed), "ticket"))
	return nil
}

// importDroppedEdges names the dependencies and parent an incoming ticket points
// at that the export does not carry.
//
// They cannot survive a remint: the ID they name belongs to another store, and
// a dependency on a ticket that does not exist here is dependency_missing, an
// error. Dropping them is the only thing that leaves a valid store, so the
// command's duty is to be loud about it rather than to avoid it.
func importDroppedEdges(t incomingTicket, in []incomingTicket) []string {
	present := make(map[string]bool, len(in))
	for _, o := range in {
		present[o.Ticket.ID] = true
	}
	var dropped []string
	for _, d := range t.Ticket.Dependencies {
		if !present[d] {
			dropped = append(dropped, "dependency "+d)
		}
	}
	if t.Ticket.Parent != nil && *t.Ticket.Parent != "" && !present[*t.Ticket.Parent] {
		dropped = append(dropped, "parent "+*t.Ticket.Parent)
	}
	return dropped
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

// keepKnownLabels splits a ticket's labels into the ones this store declares and
// the ones it does not.
//
// A label outside the allowlist is label_unknown, a warning, and a receiver
// whose CI runs check --strict fails on warnings. Carrying a sender's vocabulary
// into a store that never agreed to it is not a kindness, so the unknown ones
// are dropped and named. This is the reconciliation git am cannot do, and the
// clearest argument for import existing beside it.
func keepKnownLabels(cfg ticket.Config, labels []string) (keep, dropped []string) {
	for _, l := range labels {
		if cfg.KnownLabel(l) {
			keep = append(keep, l)
			continue
		}
		dropped = append(dropped, l)
	}
	return keep, dropped
}

// repoHasPath reports whether a reference's repository-relative path exists here.
func repoHasPath(root, rel string) bool {
	if root == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
	return err == nil
}
