package view_test

import (
	"strings"
	"testing"
	"time"

	"github.com/terva-sh/git-ticket/ticket"
	"github.com/terva-sh/git-ticket/tui"
	"github.com/terva-sh/git-ticket/tui/tuitest"
	"github.com/terva-sh/git-ticket/tui/view"
)

func TestRunDrawsAndQuitsOnQ(t *testing.T) {
	term := tuitest.NewFakeTerm(100, 12)
	lister := func() ([]*ticket.Ticket, error) {
		return []*ticket.Ticket{
			{ID: "TKT-01ARZ3NDEKTSV4RRFFQ69G5FAV", Status: "ready", Priority: "high", Title: "Visible in the run loop"},
		}, nil
	}

	done := make(chan error, 1)
	go func() { done <- view.Run(term, lister, view.Actions{}) }()

	// The first frame is drawn before any key is read, so the title
	// appears in the output stream without any scripted input.
	deadline := time.After(5 * time.Second)
	for !strings.Contains(term.Output(), "Visible in the run loop") {
		select {
		case err := <-done:
			t.Fatalf("Run returned early: %v", err)
		case <-deadline:
			t.Fatalf("first frame never painted; output: %q", term.Output())
		case <-time.After(5 * time.Millisecond):
		}
	}

	term.Type("j") // consumed by the list, must not quit
	term.Type("q")
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("Run did not quit on q")
	}

	out := term.Output()
	if !strings.Contains(out, tui.SeqAltScreenOn) || !strings.Contains(out, tui.SeqAltScreenOff) {
		t.Fatalf("Run did not enter and leave the alternate screen")
	}
}

func TestRunEndsWhenTheInputEnds(t *testing.T) {
	term := tuitest.NewFakeTerm(80, 10)
	done := make(chan error, 1)
	go func() {
		done <- view.Run(term, func() ([]*ticket.Ticket, error) { return nil, nil }, view.Actions{})
	}()
	term.CloseInput()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run on EOF: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("Run did not end on input EOF")
	}
}
