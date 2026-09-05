package view

import (
	"strings"
	"testing"

	"github.com/terva-sh/git-ticket/ticket"
	"github.com/terva-sh/git-ticket/tui"
)

func detailTicket() *ticket.Ticket {
	milestone := "v1"
	t := mk("TKT-01ARZ3NDEKTSV4RRFFQ69G5FAV", "in-progress", "high", "Restore the flux capacitor")
	t.Type = "task"
	t.Labels = []string{"power", "delorean"}
	t.Milestone = &milestone
	t.Body = ticket.Body{
		Description:        "The capacitor drifts under load and the drift compounds.",
		AcceptanceCriteria: "- [ ] drift stays under one jigowatt\n- [x] bench test recorded",
		Notes:              "2026-09-04 field observation: drift doubles in the rain.",
		Extra:              []ticket.Section{{Heading: "Vendor quotes", Text: "DMC wants twice the budget."}},
	}
	return t
}

func plainDetail(d *DetailView, cols, rows int) []string {
	out := d.Render(cols, rows)
	for i := range out {
		out[i] = tui.StripANSI(out[i])
	}
	return out
}

func TestDetailRendersHeaderAndSectionsInOrder(t *testing.T) {
	d := NewDetailView(detailTicket())
	rows := plainDetail(d, 100, 24)
	body := strings.Join(rows, "\n")

	if !strings.Contains(rows[0], "TKT-01ARZ3NDEKTSV4RRFFQ69G5FAV") {
		t.Fatalf("header lacks the full ID: %q", rows[0])
	}
	if !strings.Contains(rows[1], "Restore the flux capacitor") {
		t.Fatalf("header lacks the title: %q", rows[1])
	}
	if !strings.Contains(rows[2], "in-progress") || !strings.Contains(rows[2], "labels: power, delorean") || !strings.Contains(rows[2], "milestone: v1") {
		t.Fatalf("meta line = %q", rows[2])
	}

	// Section order per plan 5.3, Extra last.
	wantOrder := []string{
		"Description", "drift compounds",
		"Acceptance criteria", "• [ ] drift stays", "• [x] bench test",
		"Notes", "field observation",
		"Vendor quotes", "twice the budget",
	}
	pos := 0
	for _, want := range wantOrder {
		i := strings.Index(body[pos:], want)
		if i < 0 {
			t.Fatalf("missing or out of order: %q\n%s", want, body)
		}
		pos += i
	}
	if strings.Contains(body, "Implementation plan") {
		t.Fatalf("empty section rendered:\n%s", body)
	}
}

func TestDetailWrapsLongProseToTheWidth(t *testing.T) {
	tk := detailTicket()
	tk.Body.Description = strings.Repeat("wide prose keeps flowing ", 20)
	d := NewDetailView(tk)
	// The wrap promise is about the body prose. The header and footer
	// are single rows the Frame clips, per the Render contract.
	rows := plainDetail(d, 40, 20)
	for i, row := range rows[4 : len(rows)-1] {
		if w := tui.VisibleWidth(row); w > 40 {
			t.Fatalf("body row %d is %d cells wide: %q", i, w, row)
		}
	}
}

func TestDetailScrolls(t *testing.T) {
	tk := detailTicket()
	var lines []string
	for i := 0; i < 40; i++ {
		lines = append(lines, "note line "+string(rune('A'+i%26)))
	}
	tk.Body.Notes = strings.Join(lines, "\n")
	d := NewDetailView(tk)

	top := strings.Join(plainDetail(d, 80, 12), "\n")
	if !strings.Contains(top, "Description") {
		t.Fatalf("top of the body missing:\n%s", top)
	}
	d.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'G'})
	bottom := strings.Join(plainDetail(d, 80, 12), "\n")
	if strings.Contains(bottom, "Description") {
		t.Fatalf("G did not scroll the description away:\n%s", bottom)
	}
	d.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'g'})
	if again := strings.Join(plainDetail(d, 80, 12), "\n"); !strings.Contains(again, "Description") {
		t.Fatalf("g did not return to the top:\n%s", again)
	}
}

func TestDetailBackAndQuitKeys(t *testing.T) {
	for _, k := range []tui.Key{
		{Kind: tui.KeyEsc},
		{Kind: tui.KeyBackspace},
		{Kind: tui.KeyRune, Rune: 'q'},
		{Kind: tui.KeyRune, Rune: 'h'},
	} {
		d := NewDetailView(detailTicket())
		back, quit := d.HandleKey(k)
		if !back || quit {
			t.Fatalf("key %+v: back=%v quit=%v, want back only", k, back, quit)
		}
	}
	d := NewDetailView(detailTicket())
	if back, quit := d.HandleKey(tui.Key{Kind: tui.KeyCtrlC}); back || !quit {
		t.Fatalf("ctrl+c should quit")
	}
}

func TestDetailWithNoBodySaysSo(t *testing.T) {
	d := NewDetailView(mk("TKT-01BX5ZZKBKACTAV9WEVGEMMVRZ", "ready", "low", "Bare ticket"))
	body := strings.Join(plainDetail(d, 80, 10), "\n")
	if !strings.Contains(body, "no body sections") {
		t.Fatalf("bare ticket view:\n%s", body)
	}
}

func TestAppOpensDetailOnEnterAndReturnsOnEsc(t *testing.T) {
	a := NewApp(fixed(tktA, tktB), Actions{})
	if got := strings.Join(renderApp(a, 100, 12), "\n"); !strings.Contains(got, "STATUS") {
		t.Fatalf("app did not start on the list:\n%s", got)
	}
	a.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'j'})
	a.HandleKey(tui.Key{Kind: tui.KeyEnter})
	got := strings.Join(renderApp(a, 100, 12), "\n")
	if !strings.Contains(got, tktB.ID) {
		t.Fatalf("Enter did not open the selected ticket:\n%s", got)
	}
	// q in the detail goes back to the list, and does not quit the app.
	if a.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'q'}) {
		t.Fatalf("q in the detail quit the app")
	}
	if got := strings.Join(renderApp(a, 100, 12), "\n"); !strings.Contains(got, "STATUS") {
		t.Fatalf("q did not return to the list:\n%s", got)
	}
	// q on the list quits.
	if !a.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'q'}) {
		t.Fatalf("q on the list did not quit")
	}
}

func TestAppEnterOnAnEmptyListDoesNothing(t *testing.T) {
	a := NewApp(fixed(), Actions{})
	if a.HandleKey(tui.Key{Kind: tui.KeyEnter}) {
		t.Fatalf("Enter on an empty list quit")
	}
	if got := strings.Join(renderApp(a, 80, 10), "\n"); !strings.Contains(got, "No open work.") {
		t.Fatalf("empty list view lost:\n%s", got)
	}
}

func renderApp(a *App, cols, rows int) []string {
	out := a.Render(cols, rows)
	for i := range out {
		out[i] = tui.StripANSI(out[i])
	}
	return out
}
