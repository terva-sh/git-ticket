package ticket

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Importing an export, per plan 12.8.
//
// The pair is plan and apply, which is the library-shaped version of the
// preview-then-adopt split the CLI shows a person. PlanImport decides and writes
// nothing; ApplyImport carries out exactly what the plan says. A caller that
// renders a preview and a caller that files the tickets read the same answer,
// so the two can never drift apart, which they had already begun to do over
// labels while the reconciliation lived in two places.

// ImportOptions is what an import needs to know beyond the store itself.
type ImportOptions struct {
	// Patch is the export's ticket patch, as bytes. Taking bytes rather than a
	// directory is what lets an import come off a wire, and what makes this
	// testable without a filesystem.
	Patch string
	// FromStore names the sending store, recorded as provenance. It is empty
	// when the sender did not say.
	FromStore string
	// SameOwner says the two stores have one owner, so the sender's evidence is
	// this store's own and travels: the checklist ticks, the status as far as
	// 6.2.1 allows one to arrive, and the instant the work was filed.
	//
	// It says nothing about vocabulary. A label or milestone this store does not
	// declare is still dropped, and a due date still does not travel, because
	// one owner keeping two stores is not one owner keeping two allowlists. That
	// split is also what keeps check --strict green on arrival, which is the
	// constraint that shaped the label rule to begin with.
	//
	// The caller asserts it and the library does not infer it. Nothing in an
	// export says who owns the sending store, and guessing from the series is
	// the mistake import already made once: TKT is the default every store has,
	// so two strangers share it by default.
	SameOwner bool
	// Actor is recorded on everything ApplyImport writes. A plan built for a
	// preview can leave it zero, because a preview writes nothing.
	Actor Actor
}

// IncomingTicket is one ticket as it arrived.
type IncomingTicket struct {
	Ticket *Ticket
	// Path is where it sat in the sending store, which is the only thing that
	// says what its status was without trusting the frontmatter twice.
	Path string
}

// PlannedTicket is one incoming ticket and everything this store decided about
// it. It is the whole answer: a renderer needs nothing else, and ApplyImport
// consults nothing else.
type PlannedTicket struct {
	// Incoming is the ticket as it arrived, for a report that wants to name the
	// sender's ID and title.
	Incoming *Ticket
	// OriginPath is where it sat in the sending store.
	OriginPath string
	// Create is the ticket this store will file, with the sender's vocabulary
	// already reconciled.
	Create CreateOptions
	// Refs are the sender's references with an unresolvable path stripped,
	// followed by the provenance ones.
	Refs []AddReference
	// Record is the sending store's summary, notes and comments gathered into
	// one note, or empty when it carried none.
	Record string
	// Ticks are the checklist boxes to re-tick after the create, by section and
	// by one-based index. It is empty unless ImportOptions.SameOwner was set.
	//
	// They are applied after the fact rather than seeded, because
	// CreateOptions.AcceptanceCriteria takes text and files every box empty. A
	// second field there would be a published surface carrying a rule that
	// belongs to import alone.
	Ticks []ChecklistTick
	// Changes names what this store imposed.
	Changes []Change
	// DroppedEdges names the dependencies and the parent that point outside
	// this export, in the form "dependency ID" or "parent ID".
	DroppedEdges []string
	// keptParent is the parent ID kept as an origin-parent reference, or empty.
	//
	// It exists so that droppedEdges reads the decision planTicket made rather
	// than working the same condition out again. A parent kept as provenance is
	// not a dropped edge, and the first real run of --same-owner reported it as
	// both: "kept as an origin-parent reference" followed by "not carried".
	keptParent string
}

// ChecklistTick names one box to re-tick on an adopted ticket.
type ChecklistTick struct {
	Section ChecklistSection
	// Index counts from one in the order the items appear, which is the same
	// counting SetChecklistItem uses.
	Index int
}

// ImportPlan is what PlanImport decided, in the order ApplyImport will file it.
type ImportPlan struct {
	Tickets []PlannedTicket
	// Actor is recorded on everything ApplyImport writes.
	Actor Actor
}

// ImportedTicket pairs the ID a ticket arrived with and the one this store
// minted for it. That map is the one thing a caller cannot reconstruct.
type ImportedTicket struct {
	FromID string
	ID     string
	Title  string
}

// ImportResult is what ApplyImport filed.
type ImportResult struct {
	Filed []ImportedTicket
}

// PlanImport decides how an export becomes tickets of this store, and writes
// nothing.
//
// One rule runs through it: the statement of the work travels, and what the
// receiver never agreed to does not. Title, type, priority, description, plan
// and both checklists are the work. A label or milestone this store does not
// declare, a reference path that resolves to nothing here, and a deadline
// somebody else set are not. Each of those is named rather than dropped in
// silence, because a quiet loss is the failure that costs the reader their
// trust in everything else the command said.
//
// The checklists arrive with the sender's boxes ticked and land unticked. A
// tick is evidence about the sender's work, and it says nothing about whether
// this store has met the criterion. That is the same argument that files every
// imported ticket as a draft.
//
// ImportOptions.SameOwner is the one case where that argument does not hold,
// and it is a second case on the rule rather than an exception to it. When the
// two stores have one owner the sender's evidence is this store's own, so the
// ticks, the status and the filing instant travel. What the receiver agreed to
// is still the test: labels, milestones, due dates and blocks_on are reconciled
// exactly as before, because one owner keeping two stores is not one owner
// keeping two allowlists.
func (s *Store) PlanImport(ctx context.Context, o ImportOptions) (*ImportPlan, error) {
	files, err := ParseAddedFiles(o.Patch)
	if err != nil {
		return nil, err
	}
	incoming := make([]IncomingTicket, 0, len(files))
	for _, f := range files {
		t, err := Parse([]byte(f.Body))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f.Path, err)
		}
		incoming = append(incoming, IncomingTicket{Ticket: t, Path: f.Path})
	}

	ordered, err := importOrder(incoming)
	if err != nil {
		return nil, err
	}

	cfg, root := s.Config(), s.Root()
	plan := &ImportPlan{Actor: o.Actor, Tickets: make([]PlannedTicket, 0, len(ordered))}
	for _, in := range ordered {
		pt := planTicket(cfg, root, in, o, ordered)
		pt.DroppedEdges = droppedEdges(in, ordered, pt.keptParent)
		plan.Tickets = append(plan.Tickets, pt)
	}
	return plan, nil
}

// ApplyImport files every ticket in the plan, rewriting the edges among them to
// the IDs this store mints as it goes.
//
// It decides nothing. Everything it writes was settled by PlanImport, which is
// what makes a preview an honest promise rather than a second opinion.
//
// It is deliberately not atomic, per plan 7.3: making it transactional needs a
// scratch branch and a helper that rewrites a worktree, which that section
// forbids. So a failure partway leaves the tickets already filed in the store,
// and the result reports them alongside the error naming what stopped it.
//
// That pairing is the point. ImportedTicket carries FromID beside the minted
// ID, and after a partial failure that mapping is the one thing a caller cannot
// work out for itself: it does not know how many landed, so it cannot re-read
// the store and match on title without knowing where to stop. Without it the
// honest report is "some unknown number of tickets may exist in your store, go
// and look", and the caller cannot even name what to look for.
//
// A nil plan is the one error that returns nil, because nothing was filed and
// an empty result would claim a run that never happened.
func (s *Store) ApplyImport(ctx context.Context, p *ImportPlan) (*ImportResult, error) {
	if p == nil {
		return nil, fmt.Errorf("no import plan")
	}
	remap := make(map[string]string, len(p.Tickets))
	out := &ImportResult{Filed: make([]ImportedTicket, 0, len(p.Tickets))}

	for _, pt := range p.Tickets {
		create := pt.Create
		// The edges are rewritten here rather than in the plan, because the ID
		// this store mints for a ticket is not known until it is filed. The
		// order PlanImport chose is what makes the lookup succeed.
		create.Dependencies = append([]string(nil), pt.Create.Dependencies...)
		for _, d := range pt.Incoming.Dependencies {
			if to, ok := remap[d]; ok {
				create.Dependencies = append(create.Dependencies, to)
			}
		}
		if pt.Incoming.Parent != nil {
			if to, ok := remap[*pt.Incoming.Parent]; ok {
				parent := to
				create.Parent = &parent
			}
		}

		res, err := s.Create(ctx, create)
		if err != nil {
			return out, fmt.Errorf("filing %s (%s): %w", pt.Incoming.Title, pt.Incoming.ID, err)
		}
		remap[pt.Incoming.ID] = res.Ticket.ID

		for _, ref := range pt.Refs {
			if _, err := s.Apply(ctx, res.Ticket.ID, ref, ApplyOptions{Actor: p.Actor}); err != nil {
				return out, fmt.Errorf("carrying reference %s to %s: %w", ref.Ref, res.Ticket.ID, err)
			}
		}
		// The ticks go on after the create rather than through it, because
		// CreateOptions seeds a checklist from text and files every box empty.
		// The indices are the plan's, counted over the same items the create
		// just seeded in the same order, so they line up by construction.
		for _, tick := range pt.Ticks {
			m := SetChecklistItem{Section: tick.Section, Index: tick.Index, Checked: true}
			if _, err := s.Apply(ctx, res.Ticket.ID, m, ApplyOptions{Actor: p.Actor}); err != nil {
				return out, fmt.Errorf("carrying %s item %d to %s: %w", tick.Section, tick.Index, res.Ticket.ID, err)
			}
		}
		if pt.Record != "" {
			if _, err := s.Apply(ctx, res.Ticket.ID, AppendNote{Text: pt.Record}, ApplyOptions{Actor: p.Actor}); err != nil {
				return out, fmt.Errorf("carrying the work record to %s: %w", res.Ticket.ID, err)
			}
		}

		out.Filed = append(out.Filed, ImportedTicket{
			FromID: pt.Incoming.ID,
			ID:     res.Ticket.ID,
			Title:  pt.Incoming.Title,
		})
	}
	return out, nil
}

// planTicket decides how one incoming ticket becomes a ticket of this store.
func planTicket(cfg Config, root string, in IncomingTicket, o ImportOptions, all []IncomingTicket) PlannedTicket {
	t := in.Ticket
	p := PlannedTicket{Incoming: t, OriginPath: in.Path}

	labels, droppedLabels := keepKnownLabels(cfg, t.Labels)
	for _, l := range droppedLabels {
		p.Changes = append(p.Changes, Change{Kind: ChangeLabelDropped, Value: l})
	}

	p.Create = CreateOptions{
		Title:              t.Title,
		Type:               t.Type,
		Priority:           t.Priority,
		Labels:             labels,
		Assignees:          t.Assignees,
		Description:        t.Body.Description,
		ImplementationPlan: t.Body.ImplementationPlan,
		AcceptanceCriteria: ChecklistItems(t.Body.AcceptanceCriteria),
		DefinitionOfDone:   ChecklistItems(t.Body.DefinitionOfDone),
		Actor:              o.Actor,
	}
	if o.SameOwner {
		p.planEvidence(t)
	} else {
		if n := len(p.Create.AcceptanceCriteria); n > 0 {
			p.Changes = append(p.Changes, Change{Kind: ChangeAcceptanceCriteriaUnchecked, Count: n})
		}
		if n := len(p.Create.DefinitionOfDone); n > 0 {
			p.Changes = append(p.Changes, Change{Kind: ChangeDefinitionOfDoneUnchecked, Count: n})
		}
	}

	// A milestone is an allowlisted vocabulary exactly as a label is, so it is
	// reconciled the same way rather than by a second rule.
	if t.Milestone != nil && *t.Milestone != "" {
		if cfg.KnownMilestone(*t.Milestone) {
			m := *t.Milestone
			p.Create.Milestone = &m
		} else {
			p.Changes = append(p.Changes, Change{Kind: ChangeMilestoneDropped, Value: *t.Milestone})
		}
	}
	if t.DueOn != nil && *t.DueOn != "" {
		p.Changes = append(p.Changes, Change{Kind: ChangeDueOnDropped, Value: *t.DueOn})
	}
	// blocks_on gates a ticket on edges that mostly did not survive the remint,
	// so it is left at the default. Saying so costs one line and only appears
	// when the sender set it to something.
	if t.BlocksOn != "" && t.BlocksOn != BlocksOnNone {
		p.Changes = append(p.Changes, Change{Kind: ChangeBlocksOnDropped, Value: string(t.BlocksOn)})
	}

	for _, ref := range t.References {
		keep := AddReference{Ref: ref.Ref, Path: ref.Path}
		if keep.Path != nil && *keep.Path != "" && !repoHasPath(root, *keep.Path) {
			// The reference survives without its path. What the sender pointed
			// at is still worth knowing, and only the path is the thing that
			// resolves to nothing here.
			p.Changes = append(p.Changes, Change{Kind: ChangeReferencePathDropped, Value: keep.Ref})
			keep.Path = nil
		}
		p.Refs = append(p.Refs, keep)
	}

	// Provenance is a reference and never origin. `origin` is checked against
	// this store and an unresolvable one is an error, which is the guarantee it
	// exists to carry; a reference is deliberately not checked, which is exactly
	// what a foreign ID needs.
	p.Refs = append(p.Refs, AddReference{Ref: "origin-ticket:" + t.ID})
	if o.FromStore != "" {
		p.Refs = append(p.Refs, AddReference{Ref: "origin-store:" + o.FromStore})
	}
	// A parent left behind is the link the move loses, and it is not the same
	// loss as a dependency left behind. Under one owner the hierarchy is a real
	// thing the ticket belongs to, so the ID is kept as provenance where the
	// edge cannot go: parent names a ticket check resolves in this store, and a
	// foreign ID there is parent_missing, an error.
	//
	// Only under SameOwner. A stranger's parent ID resolves nowhere the receiver
	// can follow and states nothing they can use, which is the reverse of what a
	// reference is for.
	if o.SameOwner {
		if id := t.Parent; id != nil && *id != "" && !carries(all, *id) {
			p.Refs = append(p.Refs, AddReference{Ref: "origin-parent:" + *id})
			p.Changes = append(p.Changes, Change{Kind: ChangeOriginParentRecorded, Value: *id})
			p.keptParent = *id
		}
	}

	if p.Record = originRecord(t, o.FromStore); p.Record != "" {
		p.Changes = append(p.Changes, Change{Kind: ChangeWorkRecordCarried})
	}
	return p
}

// planEvidence carries the sender's evidence, for the SameOwner case.
//
// Three things move and each is evidence rather than vocabulary: which boxes the
// owner ticked, whether the owner finished the work, and when the owner filed
// it. Everything else planTicket decides is left exactly as it was.
func (p *PlannedTicket) planEvidence(t *Ticket) {
	for _, sec := range []struct {
		section ChecklistSection
		text    string
		kind    ChangeKind
	}{
		{AcceptanceCriteria, t.Body.AcceptanceCriteria, ChangeAcceptanceCriteriaCarried},
		{DefinitionOfDone, t.Body.DefinitionOfDone, ChangeDefinitionOfDoneCarried},
	} {
		ticked := 0
		// Checklist is the same view the JSON envelope and the TUI read, so a
		// box a person ticked by hand counts exactly as one this tool wrote.
		for i, item := range Checklist(sec.text) {
			if !item.Checked {
				continue
			}
			ticked++
			p.Ticks = append(p.Ticks, ChecklistTick{Section: sec.section, Index: i + 1})
		}
		if ticked > 0 {
			p.Changes = append(p.Changes, Change{Kind: sec.kind, Count: ticked})
		}
	}

	// Status travels only as far as 6.2.1 already lets one arrive. done and
	// archived are the finished states a backport is allowed to file directly;
	// every other status lands in draft, because promotion out of draft is a
	// human call and an import that filed a ticket straight into ready would
	// walk around the gate rather than through it.
	//
	// Saying so is the point. A ticket that left in-progress and arrives draft
	// is the report this whole flag came from, and the fix is not to widen
	// CreateOptions but to stop the loss being silent.
	switch t.Status {
	case StatusDone, StatusArchived:
		p.Create.Status = string(t.Status)
		p.Changes = append(p.Changes, Change{Kind: ChangeStatusCarried, Value: string(t.Status)})
	case StatusDraft, "":
		// Already where it is going, so there is nothing to say.
	default:
		p.Changes = append(p.Changes, Change{Kind: ChangeStatusNotCarried, Value: string(t.Status)})
	}

	// The filing instant, so a moved ticket keeps its real age. CreateOptions
	// takes the ULID's time part from it too, so the moved ticket sorts among
	// the work it was actually done beside rather than at the end of the store.
	// An instant after now is refused by Create, which cannot happen for a
	// ticket that already exists somewhere.
	if !t.CreatedAt.Time.IsZero() {
		p.Create.Created = t.CreatedAt.Time
	}
}

// carries reports whether this export includes the ticket with this ID.
func carries(all []IncomingTicket, id string) bool {
	for _, o := range all {
		if o.Ticket.ID == id {
			return true
		}
	}
	return false
}

// importOrder puts a ticket after everything it points at, so that a dependency
// or a parent inside the same export is already filed, and its new ID is known,
// by the time the ticket naming it is written.
//
// Edges to tickets outside the export are not ordering constraints; they are
// dropped at write time and reported, since a reminted store cannot resolve them.
func importOrder(in []IncomingTicket) ([]IncomingTicket, error) {
	byID := make(map[string]IncomingTicket, len(in))
	for _, t := range in {
		byID[t.Ticket.ID] = t
	}
	var ordered []IncomingTicket
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

// droppedEdges names the edges pointing outside this export.
//
// They cannot survive a remint: the ID they name belongs to another store, and
// a dependency on a ticket that does not exist here is dependency_missing, an
// error. Dropping them is the only thing that leaves a valid store, so the
// duty is to be loud about it rather than to avoid it.
//
// keptParent is the one exception, and it is a change of wording rather than of
// behaviour: the edge is gone either way, but when it has been kept as an
// origin-parent reference, ChangeOriginParentRecorded already says so, and more
// precisely. Reporting it here too would tell the reader it was not carried
// directly after telling them it was.
func droppedEdges(t IncomingTicket, in []IncomingTicket, keptParent string) []string {
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
	if p := t.Ticket.Parent; p != nil && *p != "" && !present[*p] && *p != keptParent {
		dropped = append(dropped, "parent "+*p)
	}
	return dropped
}

// keepKnownLabels splits a ticket's labels into the ones this store declares and
// the ones it does not.
//
// A label outside the allowlist is label_unknown, a warning, and a receiver
// whose CI runs check --strict fails on warnings. Carrying a sender's vocabulary
// into a store that never agreed to it is not a kindness, so the unknown ones
// are dropped and named. This is the reconciliation git am cannot do, and the
// clearest argument for import existing beside it.
func keepKnownLabels(cfg Config, labels []string) (keep, dropped []string) {
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

// originRecord gathers the sending store's work record into one note.
//
// This is the reasoning, the alternatives that were tried, and what the sender
// found out, which is the part of a ticket that is expensive to reproduce. It
// arrives as one note rather than as restored sections, because a note is
// honest about provenance: the actors and the instants in it are the sending
// store's, and nothing here can verify either.
//
// The headings are "###" rather than "##" because a section of a ticket is a
// line beginning "## " and this text is going inside one. The carried text
// cannot contain such a line itself: had it done so, parse would have made it a
// section of the sender's ticket and it would not be in Notes to be carried.
func originRecord(t *Ticket, fromStore string) string {
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
