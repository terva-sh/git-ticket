package view

import (
	"strings"
	"testing"

	"github.com/terva-sh/git-ticket/tui"
)

// TestFooterNamesTheWriteKeysAndFits holds the idle footer to the two
// promises that motivated it: the write flows are visible, and the
// line fits 80 columns so the Frame does not clip the hints off.
func TestFooterNamesTheWriteKeysAndFits(t *testing.T) {
	v := newTestList(fixed(
		mk("TKT-01ARZ3NDEKTSV4RRFFQ69G5FAV", "ready", "normal", "one"),
	))
	foot := tui.StripANSI(v.footerRow())
	for _, want := range []string{"n new", "e edit", "? help", "Enter open", "/ filter", "q quit"} {
		if !strings.Contains(foot, want) {
			t.Errorf("idle footer %q does not name %q", foot, want)
		}
	}
	if w := tui.VisibleWidth(foot); w > 80 {
		t.Errorf("idle footer is %d columns, wider than 80: %q", w, foot)
	}
}

func TestFilteredFooterOffersHelp(t *testing.T) {
	v := newTestList(fixed(
		mk("TKT-01ARZ3NDEKTSV4RRFFQ69G5FAV", "ready", "normal", "one"),
	))
	typeFilter(v, "status:ready")
	foot := tui.StripANSI(v.footerRow())
	if !strings.Contains(foot, "? help") {
		t.Errorf("filtered footer %q does not offer ? help", foot)
	}
}

// TestHelpOpensAndCloses walks the ? page's own ladder: ? opens it,
// Esc, q, and ? each close it back to the list without quitting, and
// Ctrl+C quits from inside it.
func TestHelpOpensAndCloses(t *testing.T) {
	for _, close := range []tui.Key{
		{Kind: tui.KeyEsc},
		{Kind: tui.KeyRune, Rune: 'q'},
		{Kind: tui.KeyRune, Rune: '?'},
	} {
		a := NewApp(fixed(
			mk("TKT-01ARZ3NDEKTSV4RRFFQ69G5FAV", "ready", "normal", "one"),
		), Actions{})
		if quit := a.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: '?'}); quit {
			t.Fatal("? quit the app instead of opening help")
		}
		page := strings.Join(renderApp(a, 80, 30), "\n")
		if !strings.Contains(page, "Keys") || !strings.Contains(page, "Filter") {
			t.Fatalf("after ?, the render is not the help page:\n%s", page)
		}
		if quit := a.HandleKey(close); quit {
			t.Fatalf("closing help with %+v quit the app", close)
		}
		back := strings.Join(renderApp(a, 80, 10), "\n")
		if !strings.Contains(back, "TITLE") {
			t.Fatalf("after closing help, the render is not the list:\n%s", back)
		}
	}

	a := NewApp(fixed(
		mk("TKT-01ARZ3NDEKTSV4RRFFQ69G5FAV", "ready", "normal", "one"),
	), Actions{})
	a.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: '?'})
	if quit := a.HandleKey(tui.Key{Kind: tui.KeyCtrlC}); !quit {
		t.Fatal("Ctrl+C inside help did not quit")
	}
}

// TestEveryListKeyIsHintedSomewhere is the contract of the ticket: a
// key the list state handles appears in the footer or on the ? page,
// or it might as well not exist. The inventory below mirrors the
// switches in App.HandleKey, ListView.HandleKey, and tui.List. Adding
// a binding means adding its row here and a hint there, in that order
// if you want the test to do its job.
func TestEveryListKeyIsHintedSomewhere(t *testing.T) {
	v := newTestList(fixed(
		mk("TKT-01ARZ3NDEKTSV4RRFFQ69G5FAV", "ready", "normal", "one"),
	))
	help := &HelpView{}
	hinted := tui.StripANSI(v.footerRow()) + "\n" +
		tui.StripANSI(strings.Join(help.Render(80, 40), "\n"))

	// Each entry is the hint token for one binding. Single letters are
	// matched as the start of a help table row or a footer phrase, so
	// an accidental letter inside prose does not satisfy them.
	inventory := map[string][]string{
		"up/down/j/k": {"j/k"},
		"g/G":         {"g/G"},
		"page keys":   {"PgUp/PgDn"},
		"Enter":       {"Enter open", "Enter, l"},
		"l":           {"Enter, l"},
		"n":           {"n new", "\n  n "},
		"e":           {"e edit", "\n  e "},
		"s":           {"\n  s "},
		"c":           {"\n  c "},
		"u":           {"\n  u "},
		"/":           {"/ filter", "\n  / "},
		"r":           {"\n  r "},
		"?":           {"? help", "\n  ? "},
		"Esc":         {"Esc"},
		"q":           {"q quit", "q, Ctrl+C"},
		"Ctrl+C":      {"Ctrl+C"},
	}
	for key, tokens := range inventory {
		found := false
		for _, tok := range tokens {
			if strings.Contains(hinted, tok) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("the %s binding is hinted nowhere: none of %q appear in the footer or help", key, tokens)
		}
	}
}
