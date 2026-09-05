package view

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/terva-sh/git-ticket/ticket"
	"github.com/terva-sh/git-ticket/tui"
)

// linkedStore builds a real store with every edge kind around one epic:
// a child that is done (which the All filter must surface), an open
// child, a dependency, and a dependent. It returns the store and the
// IDs by role.
func linkedStore(t *testing.T) (s *ticket.Store, epic, doneChild, openChild, dep, dependent string) {
	t.Helper()
	dir := t.TempDir()
	actor := ticket.Actor{ID: "human:sothr"}
	s, err := ticket.Init(dir, ticket.InitOptions{
		Actor: actor,
		Now:   func() time.Time { return time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	mk := func(o ticket.CreateOptions) string {
		t.Helper()
		o.Actor = actor
		res, err := s.Create(context.Background(), o)
		if err != nil {
			t.Fatal(err)
		}
		return res.Ticket.ID
	}
	dep = mk(ticket.CreateOptions{Title: "The dependency", Status: ticket.StatusDone})
	epic = mk(ticket.CreateOptions{Title: "The epic", Type: "epic", Dependencies: []string{dep}})
	doneChild = mk(ticket.CreateOptions{Title: "The finished child", Status: ticket.StatusDone})
	openChild = mk(ticket.CreateOptions{Title: "The open child"})
	dependent = mk(ticket.CreateOptions{Title: "Waits on the epic", Dependencies: []string{epic}})

	// Parent links: a created-done ticket takes its parent through
	// update, same as any other; both children point at the epic.
	for _, child := range []string{doneChild, openChild} {
		if _, err := s.Apply(context.Background(), child, ticket.SetParent{Parent: &epic}, ticket.ApplyOptions{Actor: actor}); err != nil {
			t.Fatal(err)
		}
	}
	return s, epic, doneChild, openChild, dep, dependent
}

// TestStoreLinksFindsEveryEdge holds storeLinks to the TKT-01M1RPZ0
// decision: all four edge kinds, over an All snapshot, so the done
// child appears.
func TestStoreLinksFindsEveryEdge(t *testing.T) {
	s, epic, doneChild, openChild, dep, dependent := linkedStore(t)
	links, err := storeLinks(StoreParams{Store: s})(epic)
	if err != nil {
		t.Fatalf("links: %v", err)
	}

	got := map[string]string{} // id -> role
	for _, l := range links {
		got[l.Ticket.ID] = l.Role
	}
	want := map[string]string{
		doneChild: RoleChild,
		openChild: RoleChild,
		dep:       RoleNeeds,
		dependent: RoleNeededBy,
	}
	for id, role := range want {
		if got[id] != role {
			t.Errorf("%s has role %q, want %q", id, got[id], role)
		}
	}
	if len(links) != len(want) {
		t.Fatalf("links = %d entries, want %d", len(links), len(want))
	}
}

// TestStoreLinksFindsTheParent is the child's view of the same family.
func TestStoreLinksFindsTheParent(t *testing.T) {
	s, epic, doneChild, _, _, _ := linkedStore(t)
	links, err := storeLinks(StoreParams{Store: s})(doneChild)
	if err != nil {
		t.Fatalf("links: %v", err)
	}
	var foundParent bool
	for _, l := range links {
		if l.Role == RoleParent && l.Ticket.ID == epic {
			foundParent = true
		}
	}
	if !foundParent {
		t.Fatalf("the child's links lack its parent: %+v", links)
	}
}

// openLinkedApp opens an App whose detail shows the given ticket and
// presses t.
func openLinkedApp(t *testing.T, tk *ticket.Ticket, links []Linked, err error) *App {
	t.Helper()
	a := NewApp(fixed(tk), Actions{
		Links: func(string) ([]Linked, error) { return links, err },
	})
	a.list.Reload()
	a.HandleKey(tui.Key{Kind: tui.KeyEnter})
	a.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 't'})
	return a
}

// TestLinkPickerOpensAndDivesTheStack: t raises the labeled picker,
// Enter pushes the chosen ticket's detail, and Esc unwinds one level
// at a time down to the list.
func TestLinkPickerOpensAndDivesTheStack(t *testing.T) {
	parent := mk("TKT-01ARZ3NDEKTSV4RRFFQ69G5FAV", "ready", "normal", "The epic itself")
	child := mk("TKT-01BX5ZZKBKACTAV9WEVGEMMVRZ", "done", "normal", "The finished child")
	child.Type = "task"
	a := openLinkedApp(t, parent, []Linked{{Role: RoleChild, Ticket: child}}, nil)

	body := strings.Join(renderApp(a, 100, 12), "\n")
	if !strings.Contains(body, "linked tickets:") || !strings.Contains(body, "child") ||
		!strings.Contains(body, "The finished child") {
		t.Fatalf("picker body:\n%s", body)
	}

	// Enter dives into the child.
	a.HandleKey(tui.Key{Kind: tui.KeyEnter})
	if top := a.top(); top == nil || top.t.ID != child.ID {
		t.Fatalf("Enter did not push the child's detail")
	}
	if len(a.details) != 2 {
		t.Fatalf("stack depth = %d, want 2", len(a.details))
	}

	// Esc unwinds one level at a time: child, parent, list.
	a.HandleKey(tui.Key{Kind: tui.KeyEsc})
	if top := a.top(); top == nil || top.t.ID != parent.ID {
		t.Fatalf("first Esc did not return to the parent")
	}
	a.HandleKey(tui.Key{Kind: tui.KeyEsc})
	if a.top() != nil {
		t.Fatalf("second Esc did not reach the list floor")
	}
}

// TestLinkPickerEmptyAndUnwired: a ticket with no links says so inside
// the picker, and a host with no Links action says so in the footer.
func TestLinkPickerEmptyAndUnwired(t *testing.T) {
	lone := mk("TKT-01ARZ3NDEKTSV4RRFFQ69G5FAV", "ready", "normal", "No family")
	a := openLinkedApp(t, lone, nil, nil)
	body := strings.Join(renderApp(a, 100, 12), "\n")
	if !strings.Contains(body, "links to nothing") {
		t.Fatalf("empty picker body:\n%s", body)
	}

	b := NewApp(fixed(lone), Actions{})
	b.list.Reload()
	b.HandleKey(tui.Key{Kind: tui.KeyEnter})
	b.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 't'})
	rows := renderApp(b, 100, 12)
	if foot := rows[len(rows)-1]; !strings.Contains(foot, "not wired") {
		t.Fatalf("footer = %q, want the unwired message", foot)
	}
}
