package ticket

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mustCreateEpic files an epic, which mustCreate cannot do because it takes a
// title and nothing else.
func mustCreateEpic(t *testing.T, s *Store, title string) *Ticket {
	t.Helper()
	res, err := s.Create(context.Background(), CreateOptions{
		Title:       title,
		Type:        "epic",
		Description: "Created by a test.",
		Actor:       testActor,
	})
	if err != nil {
		t.Fatalf("create epic: %v", err)
	}
	return res.Ticket
}

// TestEpicsIndexNamesOpenEpicsOnly pins the filter plan 15 settled. The index
// excludes done and archived rather than listing the statuses that qualify, so
// a status added to 6.1 later appears here with no edit to this code.
func TestEpicsIndexNamesOpenEpicsOnly(t *testing.T) {
	all := []*Ticket{
		{ID: "TKT-01K3ZZAAA000000000000001", Title: "Open epic", Type: "epic", Status: StatusReady},
		{ID: "TKT-01K3ZZAAA000000000000002", Title: "Finished epic", Type: "epic", Status: StatusDone},
		{ID: "TKT-01K3ZZAAA000000000000003", Title: "Filed away", Type: "epic", Status: StatusArchived},
		{ID: "TKT-01K3ZZAAA000000000000004", Title: "Not an epic", Type: "task", Status: StatusReady},
	}
	got := string(renderEpicsIndex(all))

	if !strings.Contains(got, "Open epic") {
		t.Errorf("an open epic is missing from the index:\n%s", got)
	}
	for _, absent := range []string{"Finished epic", "Filed away", "Not an epic"} {
		if strings.Contains(got, absent) {
			t.Errorf("%q should not be in the index:\n%s", absent, got)
		}
	}
}

// TestEpicsIndexOrdersByID pins the stability the whole design rests on. An
// index that rendered in store order would come back different on a machine
// that read the directory differently, and every check after it would report
// the file it just wrote as stale.
func TestEpicsIndexOrdersByID(t *testing.T) {
	first := &Ticket{ID: "TKT-01K3ZZAAA000000000000001", Title: "First by ID", Type: "epic", Status: StatusReady}
	second := &Ticket{ID: "TKT-01K3ZZAAA000000000000002", Title: "Second by ID", Type: "epic", Status: StatusReady}

	forward := string(renderEpicsIndex([]*Ticket{first, second}))
	backward := string(renderEpicsIndex([]*Ticket{second, first}))
	if forward != backward {
		t.Fatalf("input order changed the output:\n%s", diffLines(forward, backward))
	}
	if strings.Index(forward, "First by ID") > strings.Index(forward, "Second by ID") {
		t.Errorf("rows are not in ID order:\n%s", forward)
	}
}

// TestEpicsIndexLinksWhereTheTicketSits covers the reason renderEpicsIndex
// takes the directory from statusDir rather than from the file's current path.
// Fix plans this rewrite before it moves anything, so a link built from where a
// file sits right now would point at the old directory.
func TestEpicsIndexLinksWhereTheTicketSits(t *testing.T) {
	all := []*Ticket{
		{ID: "TKT-01K3ZZAAA000000000000001", Title: "Still a draft", Type: "epic", Status: StatusDraft},
		{ID: "TKT-01K3ZZAAA000000000000002", Title: "Startable", Type: "epic", Status: StatusReady},
	}
	got := string(renderEpicsIndex(all))

	for _, want := range []string{
		"(draft/TKT-01K3ZZAAA000000000000001.md)",
		"(tickets/TKT-01K3ZZAAA000000000000002.md)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing link %s:\n%s", want, got)
		}
	}
}

// TestEpicsIndexEscapesAPipeInATitle covers the one character that breaks the
// table. A title is free text and nothing else in the format validates it.
func TestEpicsIndexEscapesAPipeInATitle(t *testing.T) {
	all := []*Ticket{{
		ID:     "TKT-01K3ZZAAA000000000000001",
		Title:  "Pipe | through\nand a newline",
		Type:   "epic",
		Status: StatusReady,
	}}
	got := string(renderEpicsIndex(all))

	var row string
	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "TKT-01K3ZZAAA000000000000001") {
			row = line
		}
	}
	if row == "" {
		t.Fatalf("no row for the epic:\n%s", got)
	}
	if strings.Count(row, "|")-strings.Count(row, "\\|") != 4 {
		t.Errorf("the row does not have four unescaped cell edges: %q", row)
	}
	if !strings.Contains(row, "and a newline") {
		t.Errorf("the newline split the row: %q", row)
	}
}

// TestAFreshStoreHasACurrentEpicsIndex is why Init writes the file. A missing
// index is stale, so without that write every store would report a warning on
// the first check it ever ran.
func TestAFreshStoreHasACurrentEpicsIndex(t *testing.T) {
	s := newTestStore(t)

	if _, err := os.Stat(filepath.Join(s.path, epicsFile)); err != nil {
		t.Fatalf("init did not write %s: %v", epicsFile, err)
	}
	report, err := s.Check(context.Background())
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if has(codesOf(report.Warnings), CodeEpicsIndexStale) {
		t.Errorf("a store Init just wrote reports its own index stale")
	}
}

// TestFixRewritesAStaleEpicsIndex covers the repair end to end, and the shape
// 10.3 gives a rewrite: no ticket, no origin, and the file named in To.
func TestFixRewritesAStaleEpicsIndex(t *testing.T) {
	s := newTestStore(t)
	epic := mustCreateEpic(t, s, "Move the fleet onto short-lived credentials")

	report, err := s.Check(context.Background())
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !has(codesOf(report.Warnings), CodeEpicsIndexStale) {
		t.Fatalf("filing an epic left the index current: %v", codesOf(report.Warnings))
	}

	res := mustFix(t, s, FixOptions{})
	var rewrite *Repair
	for i := range res.Repairs {
		if res.Repairs[i].Kind == RepairRewrite {
			rewrite = &res.Repairs[i]
		}
	}
	if rewrite == nil {
		t.Fatalf("no rewrite repair: %+v", res.Repairs)
	}
	if rewrite.To != epicsFile {
		t.Errorf("to = %q, want %q", rewrite.To, epicsFile)
	}
	if rewrite.From != "" || rewrite.Ticket != "" {
		t.Errorf("a rewrite named an origin or a ticket: from=%q ticket=%q", rewrite.From, rewrite.Ticket)
	}
	if !has(rewrite.Codes, CodeEpicsIndexStale) {
		t.Errorf("codes = %v", rewrite.Codes)
	}

	data, err := os.ReadFile(filepath.Join(s.path, epicsFile))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), epic.ID) {
		t.Errorf("the rewritten index does not name the epic:\n%s", data)
	}

	// A second pass has nothing to do. An index that reported itself stale
	// forever would fail every --strict run no matter how often it was fixed.
	again := mustFix(t, s, FixOptions{})
	for _, r := range again.Repairs {
		if r.Kind == RepairRewrite {
			t.Errorf("the index is still stale after a fix pass: %+v", r)
		}
	}
}

// TestFixDryRunLeavesTheEpicsIndexAlone holds the rewrite to the same promise
// every move already makes.
func TestFixDryRunLeavesTheEpicsIndexAlone(t *testing.T) {
	s := newTestStore(t)
	mustCreateEpic(t, s, "Planned, not written")

	before, err := os.ReadFile(filepath.Join(s.path, epicsFile))
	if err != nil {
		t.Fatal(err)
	}
	res := mustFix(t, s, FixOptions{DryRun: true})

	var planned bool
	for _, r := range res.Repairs {
		if r.Kind == RepairRewrite {
			planned = true
		}
	}
	if !planned {
		t.Errorf("a dry run planned no rewrite: %+v", res.Repairs)
	}
	after, err := os.ReadFile(filepath.Join(s.path, epicsFile))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Errorf("a dry run wrote the index:\n%s", diffLines(string(before), string(after)))
	}
}
