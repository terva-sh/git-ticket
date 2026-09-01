package ticket

import (
	"context"
	"regexp"
	"sort"
	"strings"
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
	if len(f.Milestone) > 0 {
		milestone := ""
		if t.Milestone != nil {
			milestone = *t.Milestone
		}
		if !matchesOne(f.Milestone, milestone) {
			return false
		}
	}
	return true
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

// Ready returns the tickets that can be started now: status ready, no live
// claim, and every dependency satisfied per plan 6.3.
//
// Only direct dependencies are considered, so this cannot loop on a dependency
// cycle. A cycle makes its members permanently unready, which is what check
// reports as dependency_cycle.
func (s *Store) Ready(ctx context.Context) ([]*Ticket, error) {
	all, err := s.tickets(ctx)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]*Ticket, len(all))
	for _, t := range all {
		byID[t.ID] = t
	}

	now := s.now()
	out := make([]*Ticket, 0, len(all))
	for _, t := range all {
		if t.Status != StatusReady {
			continue
		}
		// An expired claim does not hold a ticket. It grants no exclusivity to
		// anyone, so the ticket is available again.
		if t.Claim != nil && !t.Claim.Expired(now) {
			continue
		}
		if !dependenciesSatisfied(t, byID) {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}

// dependenciesSatisfied reports whether every dependency is done, or archived
// out of done. A dependency that does not exist is not satisfied, because
// nothing can ever satisfy it.
func dependenciesSatisfied(t *Ticket, byID map[string]*Ticket) bool {
	for _, dep := range t.Dependencies {
		other, ok := byID[dep]
		if !ok || !other.SatisfiesDependency() {
			return false
		}
	}
	return true
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
