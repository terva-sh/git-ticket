package view

import (
	"strings"
	"testing"

	"github.com/terva-sh/git-ticket/ticket"
	"github.com/terva-sh/git-ticket/tui"
)

// typeInto sends s to the app one rune at a time.
func typeInto(a *App, s string) {
	for _, r := range s {
		a.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: r})
	}
}

func TestCreateFlowFilesATicket(t *testing.T) {
	var got struct{ title, desc string }
	created := false
	a := NewApp(fixed(tktA), Actions{
		Create: func(title, desc string) (string, error) {
			got.title, got.desc = title, desc
			created = true
			return "TKT-01NEWZZZZZZZZZZZZZZZZZZZZZ", nil
		},
	})
	a.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'n'})
	if body := strings.Join(renderApp(a, 100, 16), "\n"); !strings.Contains(body, "New ticket") {
		t.Fatalf("n did not open the create form:\n%s", body)
	}
	typeInto(a, "Fix the door")
	a.HandleKey(tui.Key{Kind: tui.KeyEnter}) // to the description
	typeInto(a, "It squeaks.")
	a.HandleKey(tui.Key{Kind: tui.KeyEnter}) // newline in prose
	typeInto(a, "Oil it.")
	a.HandleKey(tui.Key{Kind: tui.KeyCtrlS})

	if !created {
		t.Fatalf("Ctrl+S did not create")
	}
	if got.title != "Fix the door" || got.desc != "It squeaks.\nOil it." {
		t.Fatalf("Create got %+v", got)
	}
	body := strings.Join(renderApp(a, 100, 16), "\n")
	if !strings.Contains(body, "STATUS") || !strings.Contains(body, "created TKT-01NEWZZZ") {
		t.Fatalf("create did not land back on the list:\n%s", body)
	}
}

func TestCreateRequiresATitle(t *testing.T) {
	a := NewApp(fixed(tktA), Actions{
		Create: func(string, string) (string, error) {
			t.Fatal("Create ran without a title")
			return "", nil
		},
	})
	a.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'n'})
	a.HandleKey(tui.Key{Kind: tui.KeyCtrlS})
	if body := strings.Join(renderApp(a, 100, 16), "\n"); !strings.Contains(body, "a title is required") {
		t.Fatalf("no title refusal:\n%s", body)
	}
}

func TestEditFlowPrefillsAndSaves(t *testing.T) {
	tk := revved()
	tk.Body.Description = "The old prose."
	var got struct{ ref, rev, title, desc string }
	a := NewApp(fixed(tk), Actions{
		Edit: func(ref, rev, title, desc string) error {
			got.ref, got.rev, got.title, got.desc = ref, rev, title, desc
			return nil
		},
	})
	a.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'e'})
	body := strings.Join(renderApp(a, 100, 16), "\n")
	if !strings.Contains(body, "Edit") || !strings.Contains(body, tk.ID) {
		t.Fatalf("e did not open the edit form:\n%s", body)
	}
	if !strings.Contains(body, tk.Title) || !strings.Contains(body, "The old prose.") {
		t.Fatalf("edit form is not prefilled:\n%s", body)
	}
	typeInto(a, "!") // append to the title
	a.HandleKey(tui.Key{Kind: tui.KeyCtrlS})
	if got.ref != tk.ID || got.rev != "sha256:aaaa" {
		t.Fatalf("Edit got ref=%q rev=%q", got.ref, got.rev)
	}
	if got.title != tk.Title+"!" || got.desc != "The old prose." {
		t.Fatalf("Edit got title=%q desc=%q", got.title, got.desc)
	}
}

func TestEditConflictKeepsTextRearmsAndThenSaves(t *testing.T) {
	// The store-side ticket moves to revision bbbb behind the form's
	// back. The first save loses the race; the second, re-armed with
	// the reloaded revision, wins.
	tk := revved()
	moved := revved()
	moved.Revision = "sha256:bbbb"
	calls := 0
	var lastRev string
	a := NewApp(func() ([]*ticket.Ticket, error) {
		if calls == 0 {
			return []*ticket.Ticket{tk}, nil
		}
		return []*ticket.Ticket{moved}, nil
	}, Actions{
		Edit: func(ref, rev, title, desc string) error {
			calls++
			lastRev = rev
			if rev != "sha256:bbbb" {
				return &ticket.Error{Code: ticket.CodeStaleRevision, Message: "ticket changed since it was read"}
			}
			return nil
		},
	})
	a.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'e'})
	typeInto(a, "!")
	a.HandleKey(tui.Key{Kind: tui.KeyCtrlS})

	// The form survives, says so, and keeps the text.
	body := strings.Join(renderApp(a, 100, 16), "\n")
	if a.form == nil {
		t.Fatalf("conflict closed the form")
	}
	if !strings.Contains(body, "changed by another writer") {
		t.Fatalf("no conflict notice:\n%s", body)
	}
	if title, _ := a.form.Values(); !strings.HasSuffix(title, "!") {
		t.Fatalf("conflict lost the typed text: %q", title)
	}

	a.HandleKey(tui.Key{Kind: tui.KeyCtrlS})
	if lastRev != "sha256:bbbb" {
		t.Fatalf("second save was not re-armed: rev %q", lastRev)
	}
	if a.form != nil {
		t.Fatalf("successful save left the form open")
	}
	if foot := strings.Join(renderApp(a, 100, 16), "\n"); !strings.Contains(foot, "saved TKT-01ARZ3ND") {
		t.Fatalf("no saved message:\n%s", foot)
	}
}

func TestFormEscCancelsWithoutWriting(t *testing.T) {
	a := NewApp(fixed(revved()), Actions{
		Edit: func(string, string, string, string) error {
			t.Fatal("Esc wrote")
			return nil
		},
		Create: func(string, string) (string, error) {
			t.Fatal("Esc wrote")
			return "", nil
		},
	})
	a.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'e'})
	typeInto(a, "doomed text")
	a.HandleKey(tui.Key{Kind: tui.KeyEsc})
	if a.form != nil {
		t.Fatalf("Esc did not close the form")
	}
	if body := strings.Join(renderApp(a, 100, 16), "\n"); !strings.Contains(body, "STATUS") {
		t.Fatalf("Esc did not land on the list:\n%s", body)
	}
}

func TestFormTabSwitchesFieldsAndKeysAreText(t *testing.T) {
	a := NewApp(fixed(revved()), Actions{Create: func(string, string) (string, error) { return "", nil }})
	a.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'n'})
	// q, s, e, n are text inside the form, not shortcuts.
	typeInto(a, "qsen")
	if a.form == nil {
		t.Fatalf("a letter key acted as a shortcut inside the form")
	}
	a.HandleKey(tui.Key{Kind: tui.KeyTab})
	typeInto(a, "prose")
	title, desc := a.form.Values()
	if title != "qsen" || desc != "prose" {
		t.Fatalf("values = (%q, %q)", title, desc)
	}
	a.HandleKey(tui.Key{Kind: tui.KeyShiftTab})
	typeInto(a, "x")
	if title, _ := a.form.Values(); title != "qsenx" {
		t.Fatalf("shift+tab did not return to the title: %q", title)
	}
}

func TestFormUnwiredKeysSaySo(t *testing.T) {
	a := NewApp(fixed(revved()), Actions{})
	a.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'n'})
	if a.form != nil {
		t.Fatalf("create form opened with no Create wired")
	}
	a.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'e'})
	if a.form != nil {
		t.Fatalf("edit form opened with no Edit wired")
	}
	if body := strings.Join(renderApp(a, 100, 12), "\n"); !strings.Contains(body, "not wired") {
		t.Fatalf("no degradation message:\n%s", body)
	}
}
