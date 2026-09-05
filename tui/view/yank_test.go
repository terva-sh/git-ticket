package view

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/terva-sh/git-ticket/ticket"
	"github.com/terva-sh/git-ticket/tui"
)

// openDetail builds an App over one ticket, opens its detail view, and
// returns the App ready for the y binding.
func openDetail(t *testing.T, acts Actions) *App {
	t.Helper()
	a := NewApp(fixed(tktA), acts)
	a.list.Reload()
	a.HandleKey(tui.Key{Kind: tui.KeyEnter})
	if a.top() == nil {
		t.Fatal("the detail view did not open")
	}
	return a
}

// TestYankCopiesAndTheFooterConfirms: y goes through Actions.Copy, and
// the footer names the ticket, the byte count, and the path taken, so
// a person knows the paste will work before they switch windows.
func TestYankCopiesAndTheFooterConfirms(t *testing.T) {
	var asked string
	a := openDetail(t, Actions{
		Copy: func(ref string) (string, int, error) {
			asked = ref
			return "wl-copy", 42, nil
		},
	})
	a.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'y'})
	if asked != tktA.ID {
		t.Fatalf("Copy was asked for %q, want %q", asked, tktA.ID)
	}
	rows := renderApp(a, 80, 12)
	foot := rows[len(rows)-1]
	for _, want := range []string{"copied", "42 bytes", "wl-copy"} {
		if !strings.Contains(foot, want) {
			t.Fatalf("footer = %q, want it to contain %q", foot, want)
		}
	}

	// The next key clears the flash back to the hints.
	a.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'j'})
	rows = renderApp(a, 80, 12)
	if foot := rows[len(rows)-1]; !strings.Contains(foot, "y copy") {
		t.Fatalf("footer after a key = %q, want the hints back", foot)
	}
}

// TestYankUnwiredSaysSo: nil-ness is the feature detection, and the
// footer says so instead of doing nothing silently.
func TestYankUnwiredSaysSo(t *testing.T) {
	a := openDetail(t, Actions{})
	a.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'y'})
	rows := renderApp(a, 80, 12)
	if foot := rows[len(rows)-1]; !strings.Contains(foot, "not wired") {
		t.Fatalf("footer = %q, want the unwired message", foot)
	}
}

// TestYankFailureIsLoud: a failed copy reads differently from a copy
// that landed, because a paste that silently holds yesterday's
// clipboard is the worst outcome.
func TestYankFailureIsLoud(t *testing.T) {
	a := openDetail(t, Actions{
		Copy: func(string) (string, int, error) {
			return "", 0, errors.New("no clipboard tool on PATH")
		},
	})
	a.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'y'})
	rows := renderApp(a, 80, 12)
	foot := rows[len(rows)-1]
	if !strings.Contains(foot, "copy failed") || !strings.Contains(foot, "no clipboard tool") {
		t.Fatalf("footer = %q, want the loud failure", foot)
	}
}

// TestStoreCopyFeedsTheStoredBody holds the seam end to end against a
// real store: what reaches the clipboard writer is the file's own
// bytes below the frontmatter, not the view's styled render.
func TestStoreCopyFeedsTheStoredBody(t *testing.T) {
	dir := t.TempDir()
	s, err := ticket.Init(dir, ticket.InitOptions{
		Actor: ticket.Actor{ID: "human:sothr"},
		Now:   func() time.Time { return time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.Create(context.Background(), ticket.CreateOptions{
		Title:       "Yank me",
		Description: "Prose that must arrive verbatim.",
		Actor:       ticket.Actor{ID: "human:sothr"},
	})
	if err != nil {
		t.Fatal(err)
	}

	var got []byte
	copyFn := storeCopy(StoreParams{
		Store: s,
		Clipboard: func(body []byte) (string, error) {
			got = body
			return "pbcopy", nil
		},
	})
	via, n, err := copyFn(res.Ticket.ID)
	if err != nil {
		t.Fatalf("copy: %v", err)
	}
	if via != "pbcopy" || n != len(got) {
		t.Fatalf("via %q n %d, want pbcopy and %d", via, n, len(got))
	}

	data, err := os.ReadFile(res.Ticket.Path)
	if err != nil {
		t.Fatal(err)
	}
	idx := bytes.Index(data[4:], []byte("\n---\n"))
	want := data[4+idx+5:]
	if !bytes.Equal(got, want) {
		t.Fatalf("the clipboard got:\n%q\nwant the stored body:\n%q", got, want)
	}
	if !bytes.Contains(got, []byte("Prose that must arrive verbatim.")) {
		t.Fatalf("the body lacks the description: %q", got)
	}
}

// TestStoreCopyUnwiredIsNil: no Clipboard writer means no Copy action
// at all, which is what makes the footer say "not wired".
func TestStoreCopyUnwiredIsNil(t *testing.T) {
	if storeCopy(StoreParams{}) != nil {
		t.Fatal("storeCopy without a Clipboard writer must be nil")
	}
}
