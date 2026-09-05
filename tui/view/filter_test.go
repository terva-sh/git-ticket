package view

import (
	"strings"
	"testing"

	"github.com/terva-sh/git-ticket/ticket"
	"github.com/terva-sh/git-ticket/tui"
)

// filterFixtures is a store shape the filter clauses can bite on.
func filterFixtures() []*ticket.Ticket {
	a := mk("TKT-01ARZ3NDEKTSV4RRFFQ69G5FAV", "ready", "high", "Fix the flux capacitor")
	a.Type = "bug"
	a.Labels = []string{"power"}
	a.Assignees = []string{"human:sothr"}
	b := mk("TKT-01BX5ZZKBKACTAV9WEVGEMMVRZ", "in-progress", "normal", "Paint the shed")
	b.Type = "task"
	b.Labels = []string{"cosmetics"}
	b.Assignees = []string{"agent:terva/mieli"}
	c := mk("TKT-01CY5ZZKBKACTAV9WEVGEMMVS0", "ready", "low", "Paint the fence")
	c.Type = "epic"
	c.Labels = []string{"cosmetics", "power"}
	return []*ticket.Ticket{a, b, c}
}

// typeFilter opens the prompt and types s, ending with Enter.
func typeFilter(v *ListView, s string) {
	v.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: '/'})
	for _, r := range s {
		v.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: r})
	}
	v.HandleKey(tui.Key{Kind: tui.KeyEnter})
}

func shownTitles(v *ListView) string {
	var titles []string
	for _, t := range v.shown {
		titles = append(titles, t.Title)
	}
	return strings.Join(titles, "|")
}

func TestFilterByStatus(t *testing.T) {
	v := newTestList(fixed(filterFixtures()...))
	typeFilter(v, "status:ready")
	if got := shownTitles(v); got != "Fix the flux capacitor|Paint the fence" {
		t.Fatalf("shown = %q", got)
	}
	if foot := footerOf(v); !strings.Contains(foot, "2/3 match") {
		t.Fatalf("footer = %q", foot)
	}
}

func TestFilterByLabelAndAssignee(t *testing.T) {
	v := newTestList(fixed(filterFixtures()...))
	typeFilter(v, "label:cosmetics")
	if got := shownTitles(v); got != "Paint the shed|Paint the fence" {
		t.Fatalf("label filter shown = %q", got)
	}

	v = newTestList(fixed(filterFixtures()...))
	typeFilter(v, "assignee:agent:terva/mieli")
	if got := shownTitles(v); got != "Paint the shed" {
		t.Fatalf("assignee filter shown = %q", got)
	}
}

// TestFilterByTypeAndPriority covers the two tokens TKT-01M1RVH8 added.
// They follow the grammar the other fields set: alternatives within a
// field, conjunction across fields, and the same key:value spelling.
func TestFilterByTypeAndPriority(t *testing.T) {
	v := newTestList(fixed(filterFixtures()...))
	typeFilter(v, "type:epic")
	if got := shownTitles(v); got != "Paint the fence" {
		t.Fatalf("type filter shown = %q", got)
	}

	v = newTestList(fixed(filterFixtures()...))
	typeFilter(v, "priority:high")
	if got := shownTitles(v); got != "Fix the flux capacitor" {
		t.Fatalf("priority filter shown = %q", got)
	}

	// Two priorities are alternatives; the type must hold as well.
	v = newTestList(fixed(filterFixtures()...))
	typeFilter(v, "priority:high priority:low type:epic")
	if got := shownTitles(v); got != "Paint the fence" {
		t.Fatalf("combined filter shown = %q", got)
	}
}

func TestFilterConjunctionAcrossFieldsAlternativesWithin(t *testing.T) {
	v := newTestList(fixed(filterFixtures()...))
	// Two statuses are alternatives; the label must hold as well.
	typeFilter(v, "status:ready status:in-progress label:power")
	if got := shownTitles(v); got != "Fix the flux capacitor|Paint the fence" {
		t.Fatalf("shown = %q", got)
	}
}

func TestFilterByTitleWord(t *testing.T) {
	v := newTestList(fixed(filterFixtures()...))
	typeFilter(v, "PAINT fence")
	if got := shownTitles(v); got != "Paint the fence" {
		t.Fatalf("title words shown = %q", got)
	}
}

func TestFilterEditingCapturesTheLetterKeys(t *testing.T) {
	v := newTestList(fixed(filterFixtures()...))
	v.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: '/'})
	if !v.FilterEditing() {
		t.Fatalf("/ did not open the prompt")
	}
	// j and q are text now, not motion and not quit.
	if v.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'q'}) {
		t.Fatalf("q quit while the prompt had the keyboard")
	}
	v.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'j'})
	if v.filterText != "qj" {
		t.Fatalf("filterText = %q", v.filterText)
	}
	if v.SelectedID() != filterFixtures()[0].ID && v.list.Cursor() != 0 {
		t.Fatalf("cursor moved while typing")
	}
}

func TestFilterNarrowsLive(t *testing.T) {
	v := newTestList(fixed(filterFixtures()...))
	v.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: '/'})
	for _, r := range "paint" {
		v.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: r})
	}
	// Still editing, and already narrowed.
	if len(v.shown) != 2 {
		t.Fatalf("live narrowing shows %d rows", len(v.shown))
	}
	// Backspace widens again.
	for range "paint" {
		v.HandleKey(tui.Key{Kind: tui.KeyBackspace})
	}
	if len(v.shown) != 3 {
		t.Fatalf("backspace did not widen: %d rows", len(v.shown))
	}
}

func TestFilterEscSemantics(t *testing.T) {
	v := newTestList(fixed(filterFixtures()...))
	// Esc in the prompt drops the filter and the prompt.
	v.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: '/'})
	v.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'x'})
	v.HandleKey(tui.Key{Kind: tui.KeyEsc})
	if v.FilterEditing() || v.filterText != "" || len(v.shown) != 3 {
		t.Fatalf("Esc in the prompt: editing=%v text=%q shown=%d", v.FilterEditing(), v.filterText, len(v.shown))
	}
	// Esc with a confirmed filter clears it; a second Esc quits.
	typeFilter(v, "paint")
	if v.HandleKey(tui.Key{Kind: tui.KeyEsc}) {
		t.Fatalf("Esc quit instead of clearing the filter")
	}
	if v.filterText != "" || len(v.shown) != 3 {
		t.Fatalf("Esc did not clear: text=%q shown=%d", v.filterText, len(v.shown))
	}
	if !v.HandleKey(tui.Key{Kind: tui.KeyEsc}) {
		t.Fatalf("second Esc did not quit")
	}
}

func TestFilterSurvivesReload(t *testing.T) {
	v := newTestList(fixed(filterFixtures()...))
	typeFilter(v, "status:ready")
	v.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'r'})
	if len(v.shown) != 2 {
		t.Fatalf("reload dropped the filter: %d rows", len(v.shown))
	}
	if foot := footerOf(v); !strings.Contains(foot, "2/3 match") {
		t.Fatalf("footer after reload = %q", foot)
	}
}

func TestFilterSelectionFollowsIntoTheFilteredSet(t *testing.T) {
	v := newTestList(fixed(filterFixtures()...))
	v.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'G'}) // Paint the fence
	// The follow is per keystroke: each intermediate filter keeps the
	// selection only while it still matches. Every prefix of "fence"
	// keeps Paint the fence in the set, so the selection holds to the
	// end. A token like "status:ready" would shed it mid-typing at the
	// bare-word stage, and that loss is accepted behavior, not a bug.
	typeFilter(v, "fence")
	if got := v.SelectedTicket().Title; got != "Paint the fence" {
		t.Fatalf("selection after filtering = %q", got)
	}
	if len(v.shown) != 1 {
		t.Fatalf("shown = %d rows", len(v.shown))
	}
}

func TestFilterNothingMatchesSaysSo(t *testing.T) {
	v := newTestList(fixed(filterFixtures()...))
	typeFilter(v, "status:review")
	body := strings.Join(plain(v, 100, 8), "\n")
	if !strings.Contains(body, "Nothing matches the filter.") {
		t.Fatalf("empty filtered view:\n%s", body)
	}
	if v.SelectedTicket() != nil {
		t.Fatalf("a ticket is selected in an empty filtered view")
	}
}

func TestAppRoutesKeysToThePromptWhileEditing(t *testing.T) {
	a := NewApp(fixed(filterFixtures()...), Actions{})
	a.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: '/'})
	// s and l are text while the prompt has the keyboard.
	a.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 's'})
	a.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'l'})
	if a.top() != nil || a.picker != nil {
		t.Fatalf("a shortcut fired while the prompt had the keyboard")
	}
	if a.list.filterText != "sl" {
		t.Fatalf("filterText = %q", a.list.filterText)
	}
	// Enter confirms the filter rather than opening the detail.
	a.HandleKey(tui.Key{Kind: tui.KeyEnter})
	if a.top() != nil {
		t.Fatalf("Enter opened the detail from the prompt")
	}
	if a.list.FilterEditing() {
		t.Fatalf("Enter did not close the prompt")
	}
}

func TestParseFilterAssigneeKeepsItsColon(t *testing.T) {
	f := parseFilter("assignee:human:sothr status:ready stray:word bare")
	if len(f.assignee) != 1 || f.assignee[0] != "human:sothr" {
		t.Fatalf("assignee = %v", f.assignee)
	}
	if len(f.status) != 1 || f.status[0] != "ready" {
		t.Fatalf("status = %v", f.status)
	}
	// An unknown key is a title word, colon included.
	if len(f.words) != 2 || f.words[0] != "stray:word" || f.words[1] != "bare" {
		t.Fatalf("words = %v", f.words)
	}
}
