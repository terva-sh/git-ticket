package ticket

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// These tests keep the corpus honest as an artifact, which is a different job
// from the tests that run the parser over it. They replace scripts/check-corpus.py,
// which enforced the same rules from outside until there was a real parser to
// enforce them from inside.

// corpusMarkdown returns every ticket fixture in the corpus.
func corpusMarkdown(t *testing.T) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(corpusDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".md") || filepath.Base(path) == "README.md" {
			return nil
		}
		out = append(out, path)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Fatal("no fixtures found")
	}
	return out
}

// TestCorpusFileShape holds every fixture to the two rules 5.3 makes part of
// the format. A fixture saved any other way fails the round-trip test for a
// reason that has nothing to do with what it was written to cover.
func TestCorpusFileShape(t *testing.T) {
	for _, path := range corpusMarkdown(t) {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		if strings.Contains(text, "\r\n") {
			t.Errorf("%s: CRLF line endings, 5.3 requires LF", path)
		}
		if !strings.HasSuffix(text, "\n") || strings.HasSuffix(text, "\n\n") {
			t.Errorf("%s: 5.3 requires exactly one trailing newline", path)
		}
	}
}

var ticketIDPattern = regexp.MustCompile(`TKT-[0-9A-Z]+`)

// TestCorpusIDsAreWellFormed checks every ID-shaped string in the corpus,
// including the ones in prose and in dependency lists, not only the id field.
func TestCorpusIDsAreWellFormed(t *testing.T) {
	for _, path := range corpusMarkdown(t) {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, id := range ticketIDPattern.FindAllString(string(data), -1) {
			if !ValidID(id) {
				t.Errorf("%s: %s is not a 26-character Crockford ULID", path, id)
			}
		}
	}
}

// TestCorpusIDsAreUniqueAcrossCases catches a fixture copied from another one
// without changing its ID. Only duplicate-id is allowed to reuse an ID, which
// is the condition it exists to record.
func TestCorpusIDsAreUniqueAcrossCases(t *testing.T) {
	seen := map[string][]string{}
	for _, path := range corpusMarkdown(t) {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		tk, err := Parse(data)
		if err != nil || tk.ID == "" {
			continue // a reject fixture; its own test covers it
		}
		seen[tk.ID] = append(seen[tk.ID], path)
	}
	for id, paths := range seen {
		if len(paths) == 1 {
			continue
		}
		for _, p := range paths {
			if !strings.Contains(p, "duplicate-id") {
				t.Errorf("id %s is used by %v, and only the duplicate-id case may reuse one", id, paths)
				break
			}
		}
	}
}

// TestCorpusSidecarsAndFixturesPair is the rule that an absent expectation is
// not an assertion. A fixture with no sidecar could mean "this is clean" or
// "somebody forgot", and a reader cannot tell which.
func TestCorpusSidecarsAndFixturesPair(t *testing.T) {
	parseDir := filepath.Join(corpusDir, "parse")
	err := filepath.WalkDir(parseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		switch {
		case strings.HasSuffix(path, ".md"):
			if _, err := os.Stat(expectedPath(path)); err != nil {
				t.Errorf("%s: no sidecar; an absent expectation is not an assertion", path)
			}
		case strings.HasSuffix(path, ".expected.json"):
			md := strings.TrimSuffix(path, ".expected.json") + ".md"
			if _, err := os.Stat(md); err != nil {
				t.Errorf("%s: sidecar with no fixture", path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	cases, err := os.ReadDir(filepath.Join(corpusDir, "stores"))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		if !c.IsDir() {
			continue
		}
		for _, required := range []string{"expected.json", filepath.Join("store", "config.yml")} {
			p := filepath.Join(corpusDir, "stores", c.Name(), required)
			if _, err := os.Stat(p); err != nil {
				t.Errorf("%s: missing %s", c.Name(), required)
			}
		}
	}
}

// corpusSidecars returns every expectation file in the corpus.
func corpusSidecars(t *testing.T) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(corpusDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".json") {
			out = append(out, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// TestCorpusFindingShape pins the four keys of a finding. A sidecar that
// omitted a null would describe the format in a dialect the format rejects.
func TestCorpusFindingShape(t *testing.T) {
	want := map[string]bool{"code": true, "file": true, "ticket": true, "field": true}
	for _, path := range corpusSidecars(t) {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var doc struct {
			Errors   []map[string]json.RawMessage `json:"errors"`
			Warnings []map[string]json.RawMessage `json:"warnings"`
		}
		if err := json.Unmarshal(data, &doc); err != nil {
			t.Errorf("%s: invalid JSON: %v", path, err)
			continue
		}
		for _, bucket := range [][]map[string]json.RawMessage{doc.Errors, doc.Warnings} {
			for _, finding := range bucket {
				if len(finding) != len(want) {
					t.Errorf("%s: a finding must carry exactly code, file, ticket, field; got %v", path, keysOf(finding))
					continue
				}
				for k := range finding {
					if !want[k] {
						t.Errorf("%s: a finding carries an unexpected key %q", path, k)
					}
				}
			}
		}
	}
}

func keysOf(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// planCodePattern reads the code column of the tables in section 11.
var planCodePattern = regexp.MustCompile("(?m)^\\| `([a-z_]+)` \\|")

// TestCorpusCoversEveryPlanCode is the coupling that makes the corpus and the
// spec one artifact: every code section 11 defines has a fixture, and no
// sidecar names a code the plan does not define. Renaming a code in the plan
// fails here until the sidecars follow, which is intended.
func TestCorpusCoversEveryPlanCode(t *testing.T) {
	plan, err := os.ReadFile(filepath.Join("..", "docs", "plan.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(plan)
	start := strings.Index(text, "## 11. Validation")
	end := strings.Index(text, "## 12. Interfaces")
	if start < 0 || end < 0 || end < start {
		t.Fatal("cannot find section 11 in docs/plan.md")
	}
	defined := map[string]bool{}
	for _, m := range planCodePattern.FindAllStringSubmatch(text[start:end], -1) {
		defined[m[1]] = true
	}
	if len(defined) == 0 {
		t.Fatal("section 11 defines no codes, which cannot be right")
	}

	used := map[string]bool{}
	for _, path := range corpusSidecars(t) {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var doc struct {
			Errors   []finding `json:"errors"`
			Warnings []finding `json:"warnings"`
		}
		if err := json.Unmarshal(data, &doc); err != nil {
			continue // TestCorpusFindingShape reports the bad JSON
		}
		for _, f := range append(doc.Errors, doc.Warnings...) {
			used[f.Code] = true
		}
	}

	for code := range used {
		if !defined[code] {
			t.Errorf("a sidecar uses %q, which section 11 does not define", code)
		}
	}
	for code := range defined {
		if !used[code] {
			t.Errorf("section 11 defines %q, and no fixture covers it", code)
		}
	}
}
