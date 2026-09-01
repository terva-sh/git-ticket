package ticket

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestCheckStores runs check over every store fixture and compares the report
// with the expectation recorded beside it. Findings sort by file, then code,
// then field, so the two lists are compared directly rather than as sets.
func TestCheckStores(t *testing.T) {
	cases, err := os.ReadDir(filepath.Join(corpusDir, "stores"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) == 0 {
		t.Fatal("no store fixtures")
	}
	for _, c := range cases {
		if !c.IsDir() {
			continue
		}
		t.Run(c.Name(), func(t *testing.T) {
			dir := filepath.Join(corpusDir, "stores", c.Name())
			exp := loadExpectation(t, filepath.Join(dir, "expected.json"))

			// The case directory stands in for the repository root, so a
			// references path resolves against the fixture and not against
			// wherever this repository happens to be checked out.
			root, err := filepath.Abs(dir)
			if err != nil {
				t.Fatal(err)
			}
			s, err := OpenWith(filepath.Join(dir, "store"), OpenOptions{
				Now:  fixedClock(),
				Root: root,
			})
			if err != nil {
				t.Fatalf("open store: %v", err)
			}
			report, err := s.Check(context.Background())
			if err != nil {
				t.Fatalf("check: %v", err)
			}
			compareFindings(t, "errors", exp.Errors, report.Errors)
			compareFindings(t, "warnings", exp.Warnings, report.Warnings)
			if want := len(exp.Errors) == 0; report.OK() != want {
				t.Errorf("report.OK() = %v, want %v", report.OK(), want)
			}
		})
	}
}

// TestEveryFindingMatchesItsPublishedSeverity keeps CheckErrorCodes and
// CheckWarningCodes honest. `git ticket schema` publishes them as the severity
// of each code, and a caller reading a report has only the code to go on, so a
// code reclassified in check.go and not in errors.go would make that published
// answer wrong.
//
// It walks all three parts of the corpus, because no one part covers section 11
// alone. The store fixtures reach what needs a whole store, such as
// duplicate_id and the two cycles. The roundtrip fixtures reach what one
// readable file can be wrong about, such as an invalid enum. The reject
// fixtures reach the three codes that mean the file did not parse at all, which
// Check reports from f.Err rather than from checkTicket. Together they cover all
// 19 codes, which is what makes the coverage half of this test worth asserting.
func TestEveryFindingMatchesItsPublishedSeverity(t *testing.T) {
	sev := map[string]string{}
	for _, c := range CheckErrorCodes {
		sev[c] = "error"
	}
	for _, c := range CheckWarningCodes {
		if old, ok := sev[c]; ok {
			t.Fatalf("%s is published as both %s and warning", c, old)
		}
		sev[c] = "warning"
	}

	cases, err := os.ReadDir(filepath.Join(corpusDir, "stores"))
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, c := range cases {
		if !c.IsDir() {
			continue
		}
		dir := filepath.Join(corpusDir, "stores", c.Name())
		root, err := filepath.Abs(dir)
		if err != nil {
			t.Fatal(err)
		}
		s, err := OpenWith(filepath.Join(dir, "store"), OpenOptions{Now: fixedClock(), Root: root})
		if err != nil {
			t.Fatalf("%s: open store: %v", c.Name(), err)
		}
		report, err := s.Check(context.Background())
		if err != nil {
			t.Fatalf("%s: check: %v", c.Name(), err)
		}
		for _, f := range report.Errors {
			seen[f.Code] = true
			if sev[f.Code] != "error" {
				t.Errorf("%s: check reports %s as an error, published as %q",
					c.Name(), f.Code, sev[f.Code])
			}
		}
		for _, f := range report.Warnings {
			seen[f.Code] = true
			if sev[f.Code] != "warning" {
				t.Errorf("%s: check reports %s as a warning, published as %q",
					c.Name(), f.Code, sev[f.Code])
			}
		}
	}

	// The parse fixtures are the other half. checkTicket is called the way
	// TestCheckParseFixtures calls it: no config and no repository root, since
	// one file cannot answer the label allowlist or resolve a reference path.
	for _, path := range fixtures(t, filepath.Join(corpusDir, "parse", "roundtrip")) {
		name := filepath.Base(path)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		tk, err := Parse(data)
		if err != nil {
			// A fixture that does not parse is TestCheckParseFixtures' problem.
			continue
		}
		errs, warns := checkTicket(tk, name, DefaultConfig(), "", referenceInstant)
		for _, f := range errs {
			seen[f.Code] = true
			if sev[f.Code] != "error" {
				t.Errorf("%s: checkTicket reports %s as an error, published as %q",
					name, f.Code, sev[f.Code])
			}
		}
		for _, f := range warns {
			seen[f.Code] = true
			if sev[f.Code] != "warning" {
				t.Errorf("%s: checkTicket reports %s as a warning, published as %q",
					name, f.Code, sev[f.Code])
			}
		}
	}

	// A file that does not parse never reaches checkTicket. Check reports the
	// parse error itself as one error finding and moves on, so the code a
	// reject fixture produces is an error by construction.
	for _, path := range fixtures(t, filepath.Join(corpusDir, "parse", "reject")) {
		name := filepath.Base(path)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Parse(data); err == nil {
			t.Errorf("%s: in reject/ but it parses", name)
			continue
		}
		var e *Error
		if !asTicketError(err, &e) {
			t.Errorf("%s: error is not a *ticket.Error: %v", name, err)
			continue
		}
		seen[e.Code] = true
		if sev[e.Code] != "error" {
			t.Errorf("%s: a parse failure yields %s, published as %q",
				name, e.Code, sev[e.Code])
		}
	}

	// The three parts together cover every code, so anything published and
	// never emitted is either a code that no longer exists or a gap in the
	// fixtures. Both are worth failing on.
	for code := range sev {
		if !seen[code] {
			t.Errorf("%s is published but no fixture produces it", code)
		}
	}
}

// TestCheckParseFixtures holds the file-scoped checks to the same corpus. A
// parse fixture is never checked against its filename, so this exercises
// checkTicket rather than opening a store: minimal.md is not named after the
// ticket it contains and never should be.
func TestCheckParseFixtures(t *testing.T) {
	for _, path := range fixtures(t, filepath.Join(corpusDir, "parse", "roundtrip")) {
		name := filepath.Base(path)
		t.Run(name, func(t *testing.T) {
			exp := loadExpectation(t, expectedPath(path))
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			tk, err := Parse(data)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			// No config and no repository root: a parse sidecar records only
			// what reading one file can find, so the label allowlist and the
			// reference paths are out of scope.
			errs, warns := checkTicket(tk, name, DefaultConfig(), "", referenceInstant)
			sortFindings(errs)
			sortFindings(warns)
			compareFindings(t, "errors", exp.Errors, errs)
			compareFindings(t, "warnings", exp.Warnings, warns)
		})
	}
}

// TestCheckSkipsReferencesOutsideARepository is the other half of plan 5.5: a
// store with no repository root reports nothing rather than measuring a path
// against a guessed root.
func TestCheckSkipsReferencesOutsideARepository(t *testing.T) {
	dir := filepath.Join(corpusDir, "stores", "reference-unresolved")
	s, err := OpenWith(filepath.Join(dir, "store"), OpenOptions{Now: fixedClock(), NoRoot: true})
	if err != nil {
		t.Fatal(err)
	}
	report, err := s.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range report.Warnings {
		if f.Code == CodeReferencePathUnresolved {
			t.Errorf("reported %s with no repository root to resolve against", f.Code)
		}
	}
}

// TestCheckTerminatesOnACycle is the teeth of the dependency-cycle fixture: a
// naive walk of the graph would not return.
func TestCheckTerminatesOnACycle(t *testing.T) {
	dir := filepath.Join(corpusDir, "stores", "dependency-cycle")
	s, err := OpenWith(filepath.Join(dir, "store"), OpenOptions{Now: fixedClock(), NoRoot: true})
	if err != nil {
		t.Fatal(err)
	}
	report, err := s.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Errors) != 3 {
		t.Errorf("got %d errors, want one per cycle member", len(report.Errors))
	}
}

// TestCycleMembersExcludesTheApproach is the case that rules out peeling by
// out-degree: D depends on a cycle but is not part of one.
func TestCycleMembersExcludesTheApproach(t *testing.T) {
	edges := map[string][]string{
		"A": {"B"},
		"B": {"C"},
		"C": {"A"},
		"D": {"A"},
		"E": {"E"},
	}
	got := cycleMembers([]string{"A", "B", "C", "D", "E"}, edges)
	for _, id := range []string{"A", "B", "C"} {
		if !got[id] {
			t.Errorf("%s is in the cycle but was not reported", id)
		}
	}
	if got["D"] {
		t.Error("D depends on a cycle but is not in one")
	}
	if !got["E"] {
		t.Error("a self-dependency is a cycle of one")
	}
}

func compareFindings(t *testing.T, kind string, want []finding, got []Finding) {
	t.Helper()
	if len(want) != len(got) {
		t.Errorf("%s: got %d, want %d", kind, len(got), len(want))
		for _, f := range got {
			t.Logf("  got:  %s %s %s %s", f.Code, f.File, f.Ticket, f.Field)
		}
		for _, f := range want {
			t.Logf("  want: %s %s %s %s", f.Code, f.File, deref(f.Ticket), deref(f.Field))
		}
		return
	}
	for i := range want {
		w, g := want[i], got[i]
		if g.Code != w.Code || g.File != w.File || g.Ticket != deref(w.Ticket) || g.Field != deref(w.Field) {
			t.Errorf("%s[%d]:\n  got  %s %s %s %s\n  want %s %s %s %s",
				kind, i,
				g.Code, g.File, g.Ticket, g.Field,
				w.Code, w.File, deref(w.Ticket), deref(w.Field))
		}
	}
}
