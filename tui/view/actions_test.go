package view

import (
	"errors"
	"strings"
	"testing"

	"github.com/terva-sh/git-ticket/ticket"
	"github.com/terva-sh/git-ticket/tui"
)

// revved is tktA with a revision, because the fixtures elsewhere are
// built tickets and a built ticket has none. The action tests are
// exactly about what travels with the write, so here it matters.
func revved() *ticket.Ticket {
	t := mk(tktA.ID, "ready", "high", tktA.Title)
	t.Revision = "sha256:aaaa"
	return t
}

func TestClaimCarriesRefAndRevision(t *testing.T) {
	tk := revved()
	var gotRef, gotRev string
	v := NewListView(fixed(tk), Actions{
		Claim: func(ref, rev string) error { gotRef, gotRev = ref, rev; return nil },
	})
	v.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'c'})
	if gotRef != tk.ID || gotRev != "sha256:aaaa" {
		t.Fatalf("Claim got (%q, %q)", gotRef, gotRev)
	}
	if foot := footerOf(v); !strings.Contains(foot, "claimed TKT-01ARZ3ND") {
		t.Fatalf("footer = %q", foot)
	}
}

func TestReleaseCarriesRefAndRevision(t *testing.T) {
	tk := revved()
	var gotRef, gotRev string
	v := NewListView(fixed(tk), Actions{
		Release: func(ref, rev string) error { gotRef, gotRev = ref, rev; return nil },
	})
	v.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'u'})
	if gotRef != tk.ID || gotRev != "sha256:aaaa" {
		t.Fatalf("Release got (%q, %q)", gotRef, gotRev)
	}
	if foot := footerOf(v); !strings.Contains(foot, "released") {
		t.Fatalf("footer = %q", foot)
	}
}

func TestUnwiredActionsSaySo(t *testing.T) {
	v := newTestList(fixed(revved()))
	v.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'c'})
	if foot := footerOf(v); !strings.Contains(foot, "not wired") {
		t.Fatalf("footer = %q", foot)
	}
	v.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'u'})
	if foot := footerOf(v); !strings.Contains(foot, "not wired") {
		t.Fatalf("footer = %q", foot)
	}
}

func TestStaleRevisionReloadsAndRepresents(t *testing.T) {
	lists := 0
	relist := func() ([]*ticket.Ticket, error) { lists++; return []*ticket.Ticket{revved()}, nil }
	v := NewListView(relist, Actions{
		Claim: func(ref, rev string) error {
			return &ticket.Error{Code: ticket.CodeStaleRevision, Message: "ticket changed since it was read"}
		},
	})
	before := lists
	v.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'c'})
	if lists != before+1 {
		t.Fatalf("stale revision did not reload: %d lists", lists)
	}
	if foot := footerOf(v); !strings.Contains(foot, "changed by another writer") {
		t.Fatalf("footer = %q", foot)
	}
}

func TestActionErrorLandsInTheFooter(t *testing.T) {
	v := NewListView(fixed(revved()), Actions{
		Claim: func(ref, rev string) error { return errors.New("the store said no") },
	})
	v.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'c'})
	if foot := footerOf(v); !strings.Contains(foot, "the store said no") {
		t.Fatalf("footer = %q", foot)
	}
}

func footerOf(v *ListView) string {
	rows := plain(v, 120, 10)
	return rows[len(rows)-1]
}

// ---- the status picker ----

func TestStatusPickerOffersThePermittedTransitions(t *testing.T) {
	p := NewStatusPicker(revved()) // ready
	body := strings.Join(renderPicker(p, 80, 12), "\n")
	for _, want := range ticket.PermittedTransitions("ready") {
		if !strings.Contains(body, want) {
			t.Fatalf("picker lacks %q:\n%s", want, body)
		}
	}
	if !strings.Contains(body, "move from ready") {
		t.Fatalf("picker header:\n%s", body)
	}
}

func TestStatusPickerAppliesOnEnter(t *testing.T) {
	p := NewStatusPicker(revved())
	// ready permits [draft, in-progress, blocked, archived]; move to
	// the second and apply.
	p.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'j'})
	act := p.HandleKey(tui.Key{Kind: tui.KeyEnter})
	if !act.Apply || act.Status != "in-progress" || act.Reason != "" {
		t.Fatalf("action = %+v", act)
	}
}

// TestStatusPickerReasonRequiredTransitionsAsk holds the picker to
// ticket.ReasonRequired for every reason-requiring pair, not only
// blocked. Draft to done is the plan 6.2.1 addition, and done to
// in-progress is the reopening case the hardcoded check used to miss:
// the picker applied it bare and the write bounced after the fact.
func TestStatusPickerReasonRequiredTransitionsAsk(t *testing.T) {
	cases := []struct {
		name string
		from string
		to   string
	}{
		{"draft to done", "draft", "done"},
		{"done to in-progress", "done", "in-progress"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := NewStatusPicker(mk("TKT-01ARZ3NDEKTSV4RRFFQ69G5FAV", tc.from, "normal", "Arrived finished"))
			moveTo(t, p, tc.to)
			act := p.HandleKey(tui.Key{Kind: tui.KeyEnter})
			if act.Apply {
				t.Fatalf("%s applied without a reason", tc.name)
			}
			if body := strings.Join(renderPicker(p, 80, 12), "\n"); !strings.Contains(body, "reason is required") {
				t.Fatalf("no reason prompt:\n%s", body)
			}
			for _, r := range "shipped elsewhere" {
				p.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: r})
			}
			act = p.HandleKey(tui.Key{Kind: tui.KeyEnter})
			if !act.Apply || act.Status != tc.to || act.Reason != "shipped elsewhere" {
				t.Fatalf("action = %+v", act)
			}
		})
	}
}

func TestStatusPickerBlockedAsksForAReason(t *testing.T) {
	p := NewStatusPicker(revved())
	moveTo(t, p, ticket.StatusBlocked)
	act := p.HandleKey(tui.Key{Kind: tui.KeyEnter})
	if act.Apply {
		t.Fatalf("blocked applied without a reason")
	}
	if body := strings.Join(renderPicker(p, 80, 12), "\n"); !strings.Contains(body, "reason is required") {
		t.Fatalf("no reason prompt:\n%s", body)
	}
	// An empty confirm re-prompts.
	if act := p.HandleKey(tui.Key{Kind: tui.KeyEnter}); act.Apply {
		t.Fatalf("empty reason applied")
	}
	for _, r := range "vendor down" {
		p.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: r})
	}
	act = p.HandleKey(tui.Key{Kind: tui.KeyEnter})
	if !act.Apply || act.Status != ticket.StatusBlocked || act.Reason != "vendor down" {
		t.Fatalf("action = %+v", act)
	}
}

func TestStatusPickerReasonEscReturnsToTheChoices(t *testing.T) {
	p := NewStatusPicker(revved())
	moveTo(t, p, ticket.StatusBlocked)
	p.HandleKey(tui.Key{Kind: tui.KeyEnter})
	p.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'x'})
	if act := p.HandleKey(tui.Key{Kind: tui.KeyEsc}); act.Cancel || act.Apply {
		t.Fatalf("Esc in the prompt should only close the prompt: %+v", act)
	}
	if act := p.HandleKey(tui.Key{Kind: tui.KeyEsc}); !act.Cancel {
		t.Fatalf("second Esc should cancel the picker: %+v", act)
	}
}

func TestStatusPickerCancelAndQuitKeys(t *testing.T) {
	p := NewStatusPicker(revved())
	if act := p.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'q'}); !act.Cancel {
		t.Fatalf("q should cancel: %+v", act)
	}
	if act := p.HandleKey(tui.Key{Kind: tui.KeyCtrlC}); !act.Quit {
		t.Fatalf("ctrl+c should quit: %+v", act)
	}
}

func TestAppStatusFlowEndToEnd(t *testing.T) {
	tk := revved()
	var got struct{ ref, rev, status, reason string }
	a := NewApp(fixed(tk), Actions{
		SetStatus: func(ref, rev, status, reason string) error {
			got.ref, got.rev, got.status, got.reason = ref, rev, status, reason
			return nil
		},
	})
	a.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 's'})
	if body := strings.Join(renderApp(a, 100, 14), "\n"); !strings.Contains(body, "move from ready") {
		t.Fatalf("s did not open the picker:\n%s", body)
	}
	a.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'j'}) // in-progress
	a.HandleKey(tui.Key{Kind: tui.KeyEnter})
	if got.ref != tk.ID || got.rev != "sha256:aaaa" || got.status != "in-progress" || got.reason != "" {
		t.Fatalf("SetStatus got %+v", got)
	}
	body := strings.Join(renderApp(a, 100, 14), "\n")
	if !strings.Contains(body, "STATUS") || !strings.Contains(body, "is now in-progress") {
		t.Fatalf("apply did not land back on the list with the message:\n%s", body)
	}
}

func TestAppStatusKeyUnwiredSaysSo(t *testing.T) {
	a := NewApp(fixed(revved()), Actions{})
	a.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 's'})
	body := strings.Join(renderApp(a, 100, 14), "\n")
	if strings.Contains(body, "move from") {
		t.Fatalf("picker opened with no SetStatus wired:\n%s", body)
	}
	if !strings.Contains(body, "not wired") {
		t.Fatalf("no degradation message:\n%s", body)
	}
}

func moveTo(t *testing.T, p *StatusPicker, status string) {
	t.Helper()
	for i, o := range p.options {
		if o == status {
			p.list.SetCursor(i)
			return
		}
	}
	t.Fatalf("%q is not among the options %v", status, p.options)
}

func renderPicker(p *StatusPicker, cols, rows int) []string {
	out := p.Render(cols, rows)
	for i := range out {
		out[i] = tui.StripANSI(out[i])
	}
	return out
}
