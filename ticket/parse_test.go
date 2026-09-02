package ticket

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// The corpus lives at the repository root rather than under this package,
// because the CLI in Phase 2 tests against the same fixtures.
const corpusDir = "../testdata"

// expectation is a fixture sidecar, the recorded answer for one fixture. Every
// fixture has one, including the clean ones: an absent file cannot be told
// apart from a forgotten one.
type expectation struct {
	Parse     string    `json:"parse"`
	RoundTrip *string   `json:"roundTrip"`
	Errors    []finding `json:"errors"`
	Warnings  []finding `json:"warnings"`
}

type finding struct {
	Code   string  `json:"code"`
	File   string  `json:"file"`
	Ticket *string `json:"ticket"`
	Field  *string `json:"field"`
}

func loadExpectation(t *testing.T, path string) expectation {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read sidecar: %v", err)
	}
	var e expectation
	if err := json.Unmarshal(data, &e); err != nil {
		t.Fatalf("parse sidecar %s: %v", path, err)
	}
	return e
}

func fixtures(t *testing.T, dir string) []string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		t.Fatalf("glob %s: %v", dir, err)
	}
	if len(paths) == 0 {
		t.Fatalf("no fixtures in %s", dir)
	}
	return paths
}

// TestRoundTrip is the property in plan 5.3: render(parse(f)) is f, byte for
// byte, for every fixture that parses.
func TestRoundTrip(t *testing.T) {
	for _, path := range fixtures(t, filepath.Join(corpusDir, "parse", "roundtrip")) {
		name := filepath.Base(path)
		t.Run(name, func(t *testing.T) {
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			exp := loadExpectation(t, expectedPath(path))
			if exp.Parse != "ok" {
				t.Fatalf("fixture in roundtrip/ has parse %q", exp.Parse)
			}

			tk, err := Parse(want)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			got := Render(tk)
			if string(got) != string(want) {
				t.Errorf("render(parse(f)) != f\n%s", diffLines(string(want), string(got)))
			}

			// The stronger form: rendering is idempotent, so a second pass
			// through the parser cannot drift.
			again, err := Parse(got)
			if err != nil {
				t.Fatalf("reparse: %v", err)
			}
			if string(Render(again)) != string(got) {
				t.Errorf("render is not idempotent\n%s", diffLines(string(got), string(Render(again))))
			}
		})
	}
}

// TestReject checks that an unreadable file fails with the code its sidecar
// names, and attributes the failure to the same ticket and field.
func TestReject(t *testing.T) {
	for _, path := range fixtures(t, filepath.Join(corpusDir, "parse", "reject")) {
		name := filepath.Base(path)
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			exp := loadExpectation(t, expectedPath(path))
			if exp.Parse != "reject" {
				t.Fatalf("fixture in reject/ has parse %q", exp.Parse)
			}
			if len(exp.Errors) != 1 {
				t.Fatalf("a rejected file yields exactly one finding, sidecar has %d", len(exp.Errors))
			}
			want := exp.Errors[0]

			tk, err := Parse(data)
			if err == nil {
				t.Fatalf("parse succeeded, want %s", want.Code)
			}
			if tk != nil {
				t.Errorf("a rejected file must not yield a ticket")
			}
			var e *Error
			if !asTicketError(err, &e) {
				t.Fatalf("error is not a *ticket.Error: %v", err)
			}
			if e.Code != want.Code {
				t.Errorf("code = %q, want %q", e.Code, want.Code)
			}
			if got, wantID := e.Ticket, deref(want.Ticket); got != wantID {
				t.Errorf("ticket = %q, want %q", got, wantID)
			}
			if got, wantField := e.Field, deref(want.Field); got != wantField {
				t.Errorf("field = %q, want %q", got, wantField)
			}
		})
	}
}

// TestEntriesReadsAStampedLog covers the derivation behind the comments view.
// It is a reading of the section text and never the other way round, like
// Checklist, so the two cannot disagree about what a ticket says.
func TestEntriesReadsAStampedLog(t *testing.T) {
	const log = "**human:sothr** at 2026-09-30T00:00:00Z\n\n" +
		"Second pair of eyes wanted.\n\n" +
		"**agent:terva/s1** at 2026-09-30T01:00:00Z\n\n" +
		"Looks right to me.\nAcross two lines."

	got := Entries(log)
	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2: %+v", len(got), got)
	}
	if got[0].Index != 1 || got[0].Actor != "human:sothr" ||
		got[0].At != "2026-09-30T00:00:00Z" || got[0].Text != "Second pair of eyes wanted." {
		t.Errorf("first entry = %+v", got[0])
	}
	// A blank line inside an entry belongs to that entry, not to the next one.
	if got[1].Index != 2 || got[1].Actor != "agent:terva/s1" ||
		got[1].Text != "Looks right to me.\nAcross two lines." {
		t.Errorf("second entry = %+v", got[1])
	}

	if Entries("") != nil || Entries("   \n\n") != nil {
		t.Error("an empty section has no entries")
	}
}

// TestEntriesKeepsWhatAPersonHandWrote is the case the format has to survive.
// A ticket is meant to be editable, so somebody will write prose into Comments
// without the stamp this tool emits. Dropping it would lose a comment a person
// actually left, so it comes back with no actor and no time instead.
func TestEntriesKeepsWhatAPersonHandWrote(t *testing.T) {
	got := Entries("Just a line somebody typed.")
	if len(got) != 1 {
		t.Fatalf("got %d entries, want 1: %+v", len(got), got)
	}
	if got[0].Actor != "" || got[0].At != "" {
		t.Errorf("an unstamped entry should carry neither: %+v", got[0])
	}
	if got[0].Text != "Just a line somebody typed." {
		t.Errorf("text = %q", got[0].Text)
	}

	// Prose above a stamped entry is its own entry, and the numbering runs
	// over both.
	mixed := Entries("Hand-written preamble.\n\n" +
		"**human:sothr** at 2026-09-30T00:00:00Z\n\nStamped.")
	if len(mixed) != 2 {
		t.Fatalf("got %d entries, want 2: %+v", len(mixed), mixed)
	}
	if mixed[0].Actor != "" || mixed[0].Text != "Hand-written preamble." || mixed[0].Index != 1 {
		t.Errorf("first = %+v", mixed[0])
	}
	if mixed[1].Actor != "human:sothr" || mixed[1].Text != "Stamped." || mixed[1].Index != 2 {
		t.Errorf("second = %+v", mixed[1])
	}

	// The other side of the same rule, and the one worth stating. An entry runs
	// to the next stamp, so prose appended below a stamped entry joins it and
	// reads as that author's. Splitting on the blank line would not fix the
	// attribution, because the fragment sits under that stamp either way, and it
	// would take every multi-paragraph comment apart.
	trailing := Entries("**human:sothr** at 2026-09-30T00:00:00Z\n\n" +
		"Stamped.\n\nAppended later by hand.")
	if len(trailing) != 1 {
		t.Fatalf("got %d entries, want 1: %+v", len(trailing), trailing)
	}
	if trailing[0].Text != "Stamped.\n\nAppended later by hand." {
		t.Errorf("trailing prose should join the entry above it: %q", trailing[0].Text)
	}
}

// TestEntriesMatchesWhatTheToolWrites ties the reading to the writer. Comments
// and Notes both go through appendEntry, so a change to the stamp that this
// derivation cannot read back fails here rather than in a consumer.
func TestEntriesMatchesWhatTheToolWrites(t *testing.T) {
	s := newTestStore(t)
	tk := mustCreate(t, s, "Two comments and a note")

	mustApply(t, s, tk.ID, AppendComment{Text: "First question."})
	res := mustApply(t, s, tk.ID, AppendComment{Text: "Second question."})

	got := Entries(res.Ticket.Body.Comments)
	if len(got) != 2 {
		t.Fatalf("got %d comments, want 2:\n%s", len(got), res.Ticket.Body.Comments)
	}
	for i, want := range []string{"First question.", "Second question."} {
		if got[i].Text != want {
			t.Errorf("comment %d text = %q, want %q", i+1, got[i].Text, want)
		}
		if got[i].Actor != testActor.ID {
			t.Errorf("comment %d actor = %q, want %q", i+1, got[i].Actor, testActor.ID)
		}
		if got[i].At == "" {
			t.Errorf("comment %d carries no time", i+1)
		}
	}

	// Notes use the same writer, so the same reading works on them.
	res = mustApply(t, s, tk.ID, AppendNote{Text: "The skew is 40s."})
	if notes := Entries(res.Ticket.Body.Notes); len(notes) != 1 || notes[0].Text != "The skew is 40s." {
		t.Errorf("notes = %+v", notes)
	}
}

// TestParsePreservesUnknowns pins the two drift behaviours of plan 5.4 and 5.2
// that a round-trip test alone would not explain if it failed.
func TestParsePreservesUnknowns(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(corpusDir, "parse", "roundtrip", "unknown-field.md"))
	if err != nil {
		t.Fatal(err)
	}
	tk, err := Parse(data)
	if err != nil {
		t.Fatalf("an unknown field is not a read error: %v", err)
	}
	if len(tk.Unknown) != 1 || tk.Unknown[0].Key != "severity" {
		t.Fatalf("unknown fields = %+v, want one key severity", tk.Unknown)
	}

	data, err = os.ReadFile(filepath.Join(corpusDir, "parse", "roundtrip", "unknown-section.md"))
	if err != nil {
		t.Fatal(err)
	}
	tk, err = Parse(data)
	if err != nil {
		t.Fatalf("an unknown section is not a read error: %v", err)
	}
	var headings []string
	for _, s := range tk.Body.Extra {
		headings = append(headings, s.Heading)
	}
	if len(headings) != 2 || headings[0] != "Risks" || headings[1] != "Open questions" {
		t.Errorf("extra sections = %v, want [Risks, Open questions] in order", headings)
	}
}

func expectedPath(mdPath string) string {
	return mdPath[:len(mdPath)-len(".md")] + ".expected.json"
}
