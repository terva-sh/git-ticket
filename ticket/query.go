package ticket

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Every query reads every file. At the scale this format targets, hundreds to a
// few thousand tickets, that is a few milliseconds and needs no index. An index
// is deferred, and if one is ever added it must be disposable and rebuildable
// from the files.

// Filter narrows a listing. Within one field the values are alternatives, and
// across fields they all have to hold: status ready or blocked, and priority
// high.
type Filter struct {
	Status    []string
	Type      []string
	Priority  []string
	Labels    []string
	Assignees []string
	Milestone []string
	// Parent selects the direct children of a ticket. An empty string matches
	// the tickets that have no parent at all, which is what a board needs for
	// its top level, and which the CLI spells --parent none.
	//
	// Direct children only, per plan section 8. A caller that wants a whole
	// tree already has every parent edge from one List call, and walking the
	// hierarchy here would mean precomputing a descendant set before this could
	// answer a question about one ticket.
	Parent []string
	// IncludeArchived adds archived tickets to the result. They are left out by
	// default, because a list of work is about work that is still live. Asking
	// for the archived status explicitly also brings them in.
	IncludeArchived bool
}

func (f Filter) wantsArchived() bool {
	if f.IncludeArchived {
		return true
	}
	for _, s := range f.Status {
		if s == StatusArchived {
			return true
		}
	}
	return false
}

func (f Filter) matches(t *Ticket) bool {
	if t.Archived() && !f.wantsArchived() {
		return false
	}
	if !matchesOne(f.Status, t.Status) ||
		!matchesOne(f.Type, t.Type) ||
		!matchesOne(f.Priority, t.Priority) {
		return false
	}
	if !matchesAny(f.Labels, t.Labels) || !matchesAny(f.Assignees, t.Assignees) {
		return false
	}
	if !matchesOne(f.Milestone, deref(t.Milestone)) ||
		!matchesOne(f.Parent, deref(t.Parent)) {
		return false
	}
	return true
}

// deref reads an optional field. A nil pointer and an empty string mean the
// same thing here, that the ticket does not have one, which is what lets a
// filter ask for the tickets with no milestone or no parent.
func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// matchesOne reports whether value is among the wanted ones. No wanted values
// means the field was not filtered on.
func matchesOne(wanted []string, value string) bool {
	if len(wanted) == 0 {
		return true
	}
	for _, w := range wanted {
		if w == value {
			return true
		}
	}
	return false
}

// matchesAny reports whether the ticket carries at least one of the wanted
// values.
func matchesAny(wanted, have []string) bool {
	if len(wanted) == 0 {
		return true
	}
	for _, w := range wanted {
		for _, h := range have {
			if w == h {
				return true
			}
		}
	}
	return false
}

// tickets reads every ticket that parses, sorted by ID. A file that does not
// parse is left out: a query is not the place to learn that a file is broken,
// and check is.
func (s *Store) tickets(ctx context.Context) ([]*Ticket, error) {
	files, err := s.load()
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]*Ticket, 0, len(files))
	for _, f := range files {
		if f.Ticket != nil {
			out = append(out, f.Ticket)
		}
	}
	// ULIDs sort by creation time, so this is chronological for free.
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// List returns the tickets matching the filter, oldest first.
func (s *Store) List(ctx context.Context, f Filter) ([]*Ticket, error) {
	all, err := s.tickets(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*Ticket, 0, len(all))
	for _, t := range all {
		if f.matches(t) {
			out = append(out, t)
		}
	}
	return out, nil
}

// Query is a search. The text matches case-insensitively as a substring unless
// Regex is set, in which case it is an RE2 pattern.
type Query struct {
	Text   string
	Regex  bool
	Filter Filter
}

// Search looks through the title, the body sections, and the references, per
// plan section 8.
func (s *Store) Search(ctx context.Context, q Query) ([]*Ticket, error) {
	if q.Text == "" {
		return nil, &Error{Code: CodeInvalidField, Message: "an empty search", Field: "query"}
	}
	var match func(string) bool
	if q.Regex {
		re, err := regexp.Compile(q.Text)
		if err != nil {
			return nil, &Error{Code: CodeInvalidField, Message: "bad pattern: " + err.Error(), Field: "query", Err: err}
		}
		match = re.MatchString
	} else {
		needle := strings.ToLower(q.Text)
		match = func(hay string) bool { return strings.Contains(strings.ToLower(hay), needle) }
	}

	candidates, err := s.List(ctx, q.Filter)
	if err != nil {
		return nil, err
	}
	out := make([]*Ticket, 0, len(candidates))
	for _, t := range candidates {
		if anyMatch(match, searchable(t)...) {
			out = append(out, t)
		}
	}
	return out, nil
}

// searchable is every field a search reads. The frontmatter beyond the title is
// deliberately not included: a search for "task" should not return every ticket
// of that type.
func searchable(t *Ticket) []string {
	fields := []string{
		t.Title,
		t.Body.Description,
		t.Body.AcceptanceCriteria,
		t.Body.DefinitionOfDone,
		t.Body.ImplementationPlan,
		t.Body.Notes,
		t.Body.Comments,
		t.Body.Summary,
	}
	for _, r := range t.References {
		fields = append(fields, r.Ref)
		if r.Path != nil {
			fields = append(fields, *r.Path)
		}
	}
	return fields
}

func anyMatch(match func(string) bool, fields ...string) bool {
	for _, f := range fields {
		if f != "" && match(f) {
			return true
		}
	}
	return false
}

// Readiness says whether a ticket can be picked up now, and what stands in the
// way when it cannot.
//
// It is derived from the whole store at read time and never stored, like
// revision and path in plan 7.1, so no ticket file carries it and nothing can
// go stale against the files.
type Readiness struct {
	// Ready is the verdict the Ready query filters on: the status is ready, no
	// live claim holds it, and every dependency is satisfied per plan 6.3.
	Ready bool

	// Blocked reports that at least one dependency or blocking child is
	// unsatisfied or cannot be resolved. A draft, or a ticket somebody else is
	// holding, is not ready and not blocked, because nothing is in its way
	// except its own state.
	//
	// It covered dependencies alone until blocks_on arrived. Widening it was
	// the point: a caller asks this field whether a ticket can be started, and
	// an epic waiting on its children cannot be.
	Blocked bool

	// Blocking names the dependencies that resolve to a ticket which is not
	// done. Missing names the dependency IDs that no single ticket in this
	// store claims, which covers both an ID nothing claims and one that two
	// files claim. Both are sorted.
	//
	// Both fail closed. An ID that resolves to nothing, or to more than one
	// file, never counts as satisfied, because a dependency nobody can point at
	// is not a dependency anybody met.
	Blocking []string
	Missing  []string

	// BlockingChildren names the direct children that are not done, for a
	// ticket whose blocks_on is children. It is empty for every other ticket,
	// and it is sorted.
	//
	// Children get their own field rather than joining Blocking because
	// Blocking is published in plan 10.2 and versioned under 12.4. A consumer
	// rendering "waiting on" from it would print a child ID labelled as a
	// dependency, with nothing to signal the difference. A new field is
	// additive, so a consumer that ignores it reads exactly what it read
	// before.
	//
	// An epic with no children at all is therefore not blocked. Blocking it
	// would name no blocker, which section 8 refuses for a draft on the same
	// grounds. check reports that state as blocks_on_no_children instead.
	BlockingChildren []string
}

// Readiness answers for every ticket in the store, keyed by ID.
//
// One call builds the index once, so a caller explaining a whole listing does
// not re-read the store per row.
func (s *Store) Readiness(ctx context.Context) (map[string]Readiness, error) {
	all, err := s.tickets(ctx)
	if err != nil {
		return nil, err
	}
	return readinessOf(all, s.now()), nil
}

// Ready returns the tickets that can be started now: status ready, no live
// claim, and every dependency satisfied per plan 6.3.
//
// Only direct dependencies are considered, so this cannot loop on a dependency
// cycle. A cycle makes its members permanently unready, which is what check
// reports as dependency_cycle.
//
// It filters on the same verdict Readiness computes rather than repeating the
// rule, so the query and the field cannot come to disagree about one ticket.
func (s *Store) Ready(ctx context.Context) ([]*Ticket, error) {
	all, err := s.tickets(ctx)
	if err != nil {
		return nil, err
	}
	ready := readinessOf(all, s.now())
	out := make([]*Ticket, 0, len(all))
	for _, t := range all {
		if ready[t.ID].Ready {
			out = append(out, t)
		}
	}
	return out, nil
}

func readinessOf(all []*Ticket, now time.Time) map[string]Readiness {
	// Counting rather than only indexing is what makes an ambiguous dependency
	// fail closed. Two files claiming one ID is the duplicate_id that check
	// reports as an error, and until somebody repairs it neither file can be
	// said to satisfy anything.
	byID := make(map[string]*Ticket, len(all))
	claimants := make(map[string]int, len(all))
	for _, t := range all {
		claimants[t.ID]++
		byID[t.ID] = t
	}

	// The child index is the reverse of parent, built once for the whole store
	// rather than rescanned per epic. It is derived at read time and never
	// stored, like the rest of Readiness.
	children := make(map[string][]string)
	for _, t := range all {
		if t.Parent != nil && *t.Parent != "" {
			children[*t.Parent] = append(children[*t.Parent], t.ID)
		}
	}

	out := make(map[string]Readiness, len(all))
	for _, t := range all {
		var r Readiness
		for _, dep := range t.Dependencies {
			other, ok := byID[dep]
			switch {
			case !ok || claimants[dep] > 1:
				r.Missing = append(r.Missing, dep)
			case !other.SatisfiesDependency():
				r.Blocking = append(r.Blocking, dep)
			}
		}
		if t.BlocksOn == BlocksOnChildren {
			seen := make(map[string]bool, len(children[t.ID]))
			for _, child := range children[t.ID] {
				if seen[child] {
					continue
				}
				seen[child] = true
				// Fail closed for the same reason a dependency does: an ID two
				// files claim is the duplicate_id check reports, and until
				// somebody repairs it neither file can be said to be done.
				other, ok := byID[child]
				if !ok || claimants[child] > 1 || !other.SatisfiesDependency() {
					r.BlockingChildren = append(r.BlockingChildren, child)
				}
			}
		}

		sort.Strings(r.Blocking)
		sort.Strings(r.Missing)
		sort.Strings(r.BlockingChildren)

		r.Blocked = len(r.Blocking)+len(r.Missing)+len(r.BlockingChildren) > 0
		// An expired claim does not hold a ticket. It grants no exclusivity to
		// anyone, so the ticket is available again.
		held := t.Claim != nil && !t.Claim.Expired(now)
		r.Ready = t.Status == StatusReady && !r.Blocked && !held

		out[t.ID] = r
	}
	return out
}

// DepsOptions selects which edges to walk.
type DepsOptions struct {
	// Transitive follows the graph rather than stopping at the direct edges.
	Transitive bool
	// Dependents walks the edges backwards: what waits on this ticket.
	Dependents bool
}

// Deps returns the tickets this one waits on, or the ones waiting on it.
//
// The walk keeps a visited set, so a dependency cycle terminates rather than
// running until the stack gives out. A cycle is a real state a store can be in,
// and check is where a user is told about it.
func (s *Store) Deps(ctx context.Context, ref string, o DepsOptions) ([]*Ticket, error) {
	all, err := s.tickets(ctx)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]*Ticket, len(all))
	ids := make([]string, 0, len(all))
	for _, t := range all {
		byID[t.ID] = t
		ids = append(ids, t.ID)
	}
	id, err := ResolveRef(ref, ids)
	if err != nil {
		return nil, err
	}

	edges := func(from string) []string {
		if !o.Dependents {
			if t, ok := byID[from]; ok {
				return t.Dependencies
			}
			return nil
		}
		var out []string
		for _, t := range all {
			for _, d := range t.Dependencies {
				if d == from {
					out = append(out, t.ID)
				}
			}
		}
		return out
	}

	seen := map[string]bool{id: true}
	queue := append([]string{}, edges(id)...)
	var found []string
	for len(queue) > 0 {
		next := queue[0]
		queue = queue[1:]
		if seen[next] {
			continue
		}
		seen[next] = true
		found = append(found, next)
		if o.Transitive {
			queue = append(queue, edges(next)...)
		}
	}

	sort.Strings(found)
	out := make([]*Ticket, 0, len(found))
	for _, f := range found {
		if t, ok := byID[f]; ok {
			out = append(out, t)
		}
	}
	return out, nil
}

// Files returns the tickets that record a reference to a path.
//
// This reads the references the agents wrote and is only as complete as they
// were. It is advisory and is not derived from Git history.
func (s *Store) Files(ctx context.Context, path string) ([]*Ticket, error) {
	if path == "" {
		return nil, &Error{Code: CodeInvalidField, Message: "an empty path", Field: "path"}
	}
	all, err := s.tickets(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*Ticket, 0)
	for _, t := range all {
		for _, r := range t.References {
			if r.Path != nil && *r.Path == path {
				out = append(out, t)
				break
			}
			if strings.TrimPrefix(r.Ref, "file:") == path && strings.HasPrefix(r.Ref, "file:") {
				out = append(out, t)
				break
			}
		}
	}
	return out, nil
}
