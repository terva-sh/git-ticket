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
