package ticket

import (
	"context"
	"path/filepath"
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
