package ticket

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"testing"
	"time"
)

func openFixtureStore(t *testing.T, name string) *Store {
	t.Helper()
	s, err := OpenWith(filepath.Join(corpusDir, "stores", name, "store"),
		OpenOptions{Now: fixedClock(), NoRoot: true})
	if err != nil {
		t.Fatalf("open %s: %v", name, err)
	}
	return s
}

func ids(ts []*Ticket) []string {
	out := make([]string, 0, len(ts))
	for _, t := range ts {
		out = append(out, t.ID)
	}
	return out
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func TestListFilters(t *testing.T) {
	s := openFixtureStore(t, "clean")
	ctx := context.Background()

	all, err := s.List(ctx, Filter{})
	if err != nil {
		t.Fatal(err)
	}
	// The clean store holds six live tickets and one archived one, and an
	// unfiltered list is about live work.
	if len(all) != 6 {
		t.Errorf("unfiltered list returned %d, want the 6 live tickets", len(all))
	}
	for _, tk := range all {
		if tk.Archived() {
			t.Errorf("%s is archived and should not be in an unfiltered list", tk.ID)
		}
	}

	withArchived, err := s.List(ctx, Filter{IncludeArchived: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(withArchived) != 7 {
		t.Errorf("list with archived returned %d, want 7", len(withArchived))
	}

	// Naming the archived status is another way of asking for them.
	archived, err := s.List(ctx, Filter{Status: []string{StatusArchived}})
	if err != nil {
		t.Fatal(err)
	}
	if len(archived) != 1 {
		t.Errorf("status archived returned %d, want 1", len(archived))
	}

	// Within a field the values are alternatives.
	twoStatuses, err := s.List(ctx, Filter{Status: []string{StatusReady, StatusDone}})
	if err != nil {
		t.Fatal(err)
	}
	if len(twoStatuses) != 2 {
		t.Errorf("ready or done returned %d, want 2", len(twoStatuses))
	}

	// Across fields they all have to hold.
	none, err := s.List(ctx, Filter{Status: []string{StatusDraft}, Priority: []string{"urgent"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(none) != 0 {
		t.Errorf("draft and urgent returned %v, want nothing", ids(none))
	}

	labelled, err := s.List(ctx, Filter{Labels: []string{"auth"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(labelled) == 0 {
		t.Error("no ticket carries the auth label, but the fixture has some")
	}
	for _, tk := range labelled {
		if !contains(tk.Labels, "auth") {
			t.Errorf("%s does not carry auth", tk.ID)
		}
	}
}

// The hierarchy fixture, which is the only store in the corpus whose parent
// links are all valid. parent-missing and parent-cycle are deliberately broken
// and cannot show what a correct hierarchy reads like.
const (
	hierEpic       = "TKT-01K401AAB00000000000000001"
	hierChildReady = "TKT-01K401AAB00000000000000002"
	hierChildDone  = "TKT-01K401AAB00000000000000003"
	hierGrandchild = "TKT-01K401AAB00000000000000004"
)

// TestListFiltersOnParent covers plan section 8: the parent filter matches
// direct children, and an empty string matches the tickets that have no parent.
//
// Terva reads this through Filter rather than through the CLI, per the Phase 3
// handoff, so the library owns the behaviour and gets its own test.
func TestListFiltersOnParent(t *testing.T) {
	s := openFixtureStore(t, "hierarchy")
	ctx := context.Background()

	list := func(f Filter) []string {
		t.Helper()
		got, err := s.List(ctx, f)
		if err != nil {
			t.Fatal(err)
		}
		return ids(got)
	}

	// Direct children only. The grandchild hangs off hierChildReady, so a
	// filter that walked the hierarchy would return three here instead of two.
	// That is the assertion separating a direct match from a descendant walk.
	kids := list(Filter{Parent: []string{hierEpic}})
	if len(kids) != 2 || !contains(kids, hierChildReady) || !contains(kids, hierChildDone) {
		t.Errorf("children of the epic = %v, want the two direct ones", kids)
	}
	if contains(kids, hierGrandchild) {
		t.Error("the grandchild is not a direct child of the epic")
	}

	// One level down, the grandchild is the direct child.
	if got := list(Filter{Parent: []string{hierChildReady}}); len(got) != 1 || got[0] != hierGrandchild {
		t.Errorf("children of the first child = %v, want the grandchild", got)
	}

	// An empty string asks for the tickets with no parent, which is what a
	// board needs for its top level and what the CLI spells --parent none.
	roots := list(Filter{Parent: []string{""}})
	if len(roots) != 1 || roots[0] != hierEpic {
		t.Errorf("parentless tickets = %v, want only the epic", roots)
	}

	// Within one filter the values are alternatives, so this is the epic's
	// children plus the first child's.
	both := list(Filter{Parent: []string{hierEpic, hierChildReady}})
	if len(both) != 3 {
		t.Errorf("two parents = %v, want all three descendants", both)
	}

	// Across filters they all have to hold.
	done := list(Filter{Parent: []string{hierEpic}, Status: []string{StatusDone}})
	if len(done) != 1 || done[0] != hierChildDone {
		t.Errorf("done children of the epic = %v, want one", done)
	}

	// The library matches the stored value and resolves nothing. Turning a
	// prefix into an ID is the CLI's job, per the resolveID comment there, so
	// an ID this store does not hold is simply no match rather than an error.
	if got := list(Filter{Parent: []string{"TKT-01K401AAB0000000000000009Z"}}); len(got) != 0 {
		t.Errorf("an unknown parent matched %v, want nothing", got)
	}
}

func TestSearch(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	hit := mustCreate(t, s, "Rotate the signing key")
	mustApply(t, s, hit.ID, AppendNote{Text: "The overlap window outlasts the longest refresh token."})
	miss := mustCreate(t, s, "Something else entirely")

	// Case-insensitive substring by default, over the title.
	got, err := s.Search(ctx, Query{Text: "SIGNING"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != hit.ID {
		t.Errorf("search returned %v, want just %s", ids(got), hit.ID)
	}

	// And over the body.
	got, err = s.Search(ctx, Query{Text: "refresh token"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != hit.ID {
		t.Errorf("body search returned %v, want just %s", ids(got), hit.ID)
	}

	// Regex mode.
	got, err = s.Search(ctx, Query{Text: "^Something", Regex: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != miss.ID {
		t.Errorf("regex search returned %v, want just %s", ids(got), miss.ID)
	}

	// A pattern that does not compile says so rather than matching nothing.
	if _, err := s.Search(ctx, Query{Text: "([", Regex: true}); CodeOf(err) != CodeInvalidField {
		t.Errorf("bad pattern = %v, want %s", err, CodeInvalidField)
	}

	// The type is not searchable, or every ticket would answer to "task".
	got, err = s.Search(ctx, Query{Text: "task"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("searching the type returned %v, want nothing", ids(got))
	}
}

func TestReadyRules(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	// A draft is not ready, whatever else is true of it.
	draft := mustCreate(t, s, "Still a sketch")
	// A plain ready ticket is.
	open := mustCreate(t, s, "Ready and unclaimed")
	mustApply(t, s, open.ID, SetStatus{Status: StatusReady})
	// A claimed one is not, because somebody holds it.
	claimed := mustCreate(t, s, "Ready but held")
	mustApply(t, s, claimed.ID, SetStatus{Status: StatusReady})
	mustApply(t, s, claimed.ID, ClaimTicket{ExpiresIn: time.Hour})

	got, err := s.Ready(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != open.ID {
		t.Fatalf("ready = %v, want just %s", ids(got), open.ID)
	}
	if contains(ids(got), draft.ID) {
		t.Error("a draft is not ready")
	}

	// An expired claim does not hold anything, so the ticket comes back.
	s.now = func() time.Time { return referenceInstant.Add(2 * time.Hour) }
	got, err = s.Ready(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(ids(got), claimed.ID) {
		t.Errorf("ready = %v, want the ticket with the expired claim back", ids(got))
	}
	s.now = fixedClock()

	// An unsatisfied dependency holds a ticket back.
	blocker := mustCreate(t, s, "Has to land first")
	waiter := mustCreate(t, s, "Waits on the blocker")
	mustApply(t, s, waiter.ID, SetStatus{Status: StatusReady})
	mustApply(t, s, waiter.ID, AddDependency{ID: blocker.ID})
	if got, _ := s.Ready(ctx); contains(ids(got), waiter.ID) {
		t.Error("a ticket waiting on a draft is not ready")
	}

	// Done satisfies it.
	mustApply(t, s, blocker.ID, SetStatus{Status: StatusReady})
	mustApply(t, s, blocker.ID, SetStatus{Status: StatusInProgress})
	mustApply(t, s, blocker.ID, SetStatus{Status: StatusDone})
	if got, _ := s.Ready(ctx); !contains(ids(got), waiter.ID) {
		t.Error("a ticket whose dependency is done should be ready")
	}

	// And so does archived out of done, per 6.3.
	mustApply(t, s, blocker.ID, ArchiveTicket{Reason: "filed"})
	if got, _ := s.Ready(ctx); !contains(ids(got), waiter.ID) {
		t.Error("archived out of done still satisfies a dependency")
	}
}

// TestReadinessExplainsItself covers the field a query could never carry: not
// just whether a ticket is startable but what stands in the way.
//
// Blocked is about dependencies alone. A draft and a held ticket are both
// unready with nothing in their way but their own state, and reporting them as
// blocked would send a reader looking for a dependency that is not there.
func TestReadinessExplainsItself(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	draft := mustCreate(t, s, "Still a sketch")

	open := mustCreate(t, s, "Ready and unclaimed")
	mustApply(t, s, open.ID, SetStatus{Status: StatusReady})

	held := mustCreate(t, s, "Ready but held")
	mustApply(t, s, held.ID, SetStatus{Status: StatusReady})
	mustApply(t, s, held.ID, ClaimTicket{ExpiresIn: time.Hour})

	blocker := mustCreate(t, s, "Has to land first")
	waiter := mustCreate(t, s, "Waits on the blocker")
	mustApply(t, s, waiter.ID, SetStatus{Status: StatusReady})
	mustApply(t, s, waiter.ID, AddDependency{ID: blocker.ID})

	r, err := s.Readiness(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if got := r[open.ID]; !got.Ready || got.Blocked {
		t.Errorf("open = %+v, want ready and unblocked", got)
	}
	if got := r[draft.ID]; got.Ready || got.Blocked {
		t.Errorf("draft = %+v, want unready but not blocked", got)
	}
	if got := r[held.ID]; got.Ready || got.Blocked {
		t.Errorf("held = %+v, want unready but not blocked", got)
	}

	got := r[waiter.ID]
	if got.Ready || !got.Blocked {
		t.Errorf("waiter = %+v, want blocked", got)
	}
	if len(got.Blocking) != 1 || got.Blocking[0] != blocker.ID {
		t.Errorf("blocking = %v, want [%s]", got.Blocking, blocker.ID)
	}
	if len(got.Missing) != 0 {
		t.Errorf("missing = %v, want empty: the dependency exists", got.Missing)
	}

	// Every ticket the verdict calls ready is exactly what Ready returns. The
	// two read the same derivation, and this is what holds them to it.
	list, err := s.Ready(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var agreed []string
	for id, v := range r {
		if v.Ready {
			agreed = append(agreed, id)
		}
	}
	sort.Strings(agreed)
	fromQuery := ids(list)
	sort.Strings(fromQuery)
	if !slices.Equal(agreed, fromQuery) {
		t.Errorf("the field and the query disagree\nfield: %v\nquery: %v", agreed, fromQuery)
	}
}

// TestUnresolvableDependenciesFailClosed is the rule worth copying from
// Backlog.md verbatim: a dependency that resolves to nothing, or to more than
// one ticket, blocks rather than counting as satisfied.
//
// Both are states check already reports, as dependency_missing and
// duplicate_id. Until somebody repairs them, nothing can be said to have met
// the dependency, and guessing would hand an agent work whose prerequisite
// nobody can point at.
func TestUnresolvableDependenciesFailClosed(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	// A dependency nothing claims.
	orphan := mustCreate(t, s, "Waits on a ticket that is not there")
	mustApply(t, s, orphan.ID, SetStatus{Status: StatusReady})
	const ghost = "TKT-01K3ZZZZZZZZZZZZZZZZZZZZZZ"
	// AddDependency refuses an ID with nothing behind it, which is the right
	// answer for a mutation. Reaching the state this asserts means writing the
	// file the way a bad merge would leave it.
	cur, err := s.Get(ctx, orphan.ID)
	if err != nil {
		t.Fatal(err)
	}
	cur.Dependencies = []string{ghost}
	if err := os.WriteFile(cur.Path, Render(cur), 0o644); err != nil {
		t.Fatal(err)
	}

	// A dependency two files claim. Take it all the way to done first, so the
	// waiter below is genuinely satisfied until the duplicate appears.
	twin := mustCreate(t, s, "Claimed by two files")
	mustApply(t, s, twin.ID, SetStatus{Status: StatusReady})
	mustApply(t, s, twin.ID, SetStatus{Status: StatusInProgress})
	done := mustApply(t, s, twin.ID, SetStatus{Status: StatusDone})

	waiter := mustCreate(t, s, "Waits on the twin")
	mustApply(t, s, waiter.ID, SetStatus{Status: StatusReady})
	mustApply(t, s, waiter.ID, AddDependency{ID: twin.ID})

	// Satisfied, for now.
	before, err := s.Readiness(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !before[waiter.ID].Ready {
		t.Fatalf("waiter should be ready while its dependency resolves: %+v", before[waiter.ID])
	}

	// Now a second file claims the same ID. Filenames are the ID, so a file
	// under another name is how a bad merge produces this. Every create is
	// already done, because the store refuses to write once an ID is ambiguous.
	dup := filepath.Join(s.TicketsDir(), "duplicate-of-the-twin.md")
	if err := os.WriteFile(dup, Render(done.Ticket), 0o644); err != nil {
		t.Fatal(err)
	}

	r, err := s.Readiness(ctx)
	if err != nil {
		t.Fatal(err)
	}

	got := r[orphan.ID]
	if got.Ready || !got.Blocked {
		t.Errorf("orphan = %+v, want blocked", got)
	}
	if len(got.Missing) != 1 || got.Missing[0] != ghost {
		t.Errorf("missing = %v, want [%s]", got.Missing, ghost)
	}

	// The twin is done twice over, so a rule that only asked "is it done"
	// would call this satisfied. Two files claiming one ID means no single
	// ticket claims it, so it is missing rather than met.
	got = r[waiter.ID]
	if got.Ready || !got.Blocked {
		t.Errorf("waiter on an ambiguous ID = %+v, want blocked", got)
	}
	if len(got.Missing) != 1 || got.Missing[0] != twin.ID {
		t.Errorf("missing = %v, want the ambiguous [%s]", got.Missing, twin.ID)
	}
	if len(got.Blocking) != 0 {
		t.Errorf("blocking = %v, want empty: the ID resolves to nothing single", got.Blocking)
	}

	// And the query agrees, because it reads the same derivation.
	list, err := s.Ready(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{orphan.ID, waiter.ID} {
		if contains(ids(list), id) {
			t.Errorf("%s is in ready with an unresolvable dependency", id)
		}
	}
}

// TestReadyOnTheArchivedDependencyFixture uses the corpus case built for this:
// one dependency archived out of done, one archived out of ready. The second
// satisfies nothing, so the ticket is not ready.
func TestReadyOnTheArchivedDependencyFixture(t *testing.T) {
	s := openFixtureStore(t, "archived-dependency")
	got, err := s.Ready(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("ready = %v, want nothing: one dependency was archived without reaching done", ids(got))
	}
}

// TestQueriesTerminateOnACycle is the one with teeth. A dependency cycle is a
// state the store can really be in, and neither Ready nor a transitive walk may
// run forever in it.
func TestQueriesTerminateOnACycle(t *testing.T) {
	s := openFixtureStore(t, "dependency-cycle")
	ctx := context.Background()

	done := make(chan struct{})
	go func() {
		defer close(done)
		if _, err := s.Ready(ctx); err != nil {
			t.Errorf("ready: %v", err)
		}
		all, err := s.List(ctx, Filter{})
		if err != nil {
			t.Errorf("list: %v", err)
			return
		}
		for _, tk := range all {
			if _, err := s.Deps(ctx, tk.ID, DepsOptions{Transitive: true}); err != nil {
				t.Errorf("deps: %v", err)
			}
			if _, err := s.Deps(ctx, tk.ID, DepsOptions{Transitive: true, Dependents: true}); err != nil {
				t.Errorf("dependents: %v", err)
			}
		}
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("a query did not terminate on a dependency cycle")
	}
}

func TestDeps(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	a := mustCreate(t, s, "A, the one that waits")
	b := mustCreate(t, s, "B, in the middle")
	c := mustCreate(t, s, "C, the root of it")
	mustApply(t, s, a.ID, AddDependency{ID: b.ID})
	mustApply(t, s, b.ID, AddDependency{ID: c.ID})

	direct, err := s.Deps(ctx, a.ID, DepsOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(direct) != 1 || direct[0].ID != b.ID {
		t.Errorf("direct deps = %v, want just B", ids(direct))
	}

	all, err := s.Deps(ctx, a.ID, DepsOptions{Transitive: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || !contains(ids(all), b.ID) || !contains(ids(all), c.ID) {
		t.Errorf("transitive deps = %v, want B and C", ids(all))
	}

	dependents, err := s.Deps(ctx, c.ID, DepsOptions{Transitive: true, Dependents: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(dependents) != 2 {
		t.Errorf("transitive dependents of C = %v, want A and B", ids(dependents))
	}
}

func TestFiles(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	tk := mustCreate(t, s, "Touches the signing code")
	path := "internal/auth/signing.go"
	mustApply(t, s, tk.ID, AddReference{Ref: "file:" + path, Path: &path})
	mustCreate(t, s, "Touches nothing")

	got, err := s.Files(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != tk.ID {
		t.Errorf("files = %v, want just %s", ids(got), tk.ID)
	}

	none, err := s.Files(ctx, "internal/auth/nothing.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(none) != 0 {
		t.Errorf("files = %v, want nothing", ids(none))
	}
}
