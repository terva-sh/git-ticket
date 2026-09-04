package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/terva-sh/git-ticket/ticket"
)

// pad builds a title of exactly n characters.
func pad(n int) string { return strings.Repeat("x", n) }

// corrupt rewrites every ticket file in the store, replacing old with new. A
// hand edit is the only way to a finding on a ticket the CLI would not have
// written, which is the same shape the title-too-long fixture records.
func corrupt(t *testing.T, dir, old, new string) {
	t.Helper()
	var touched int
	root := filepath.Join(dir, ".tickets")
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".md" {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !strings.Contains(string(b), old) {
			return nil
		}
		touched++
		return os.WriteFile(path, []byte(strings.Replace(string(b), old, new, 1)), 0o644)
	})
	if err != nil {
		t.Fatalf("corrupting the store: %v", err)
	}
	if touched == 0 {
		// Otherwise the test passes for the wrong reason: no edit, no finding,
		// and an assertion that never had anything to catch.
		t.Fatalf("no ticket file contained %q", old)
	}
}

// TestTitleLengthBoundaries holds the two thresholds of plan 5.1 to the exact
// character. An off-by-one here is invisible in ordinary use and would only
// surface as a store that fails check for a title somebody was told was legal.
func TestTitleLengthBoundaries(t *testing.T) {
	for _, tc := range []struct {
		n    int
		want string
	}{
		{71, ""},
		{72, ""},
		{73, ticket.CodeTitleLong},
		{119, ticket.CodeTitleLong},
		{120, ticket.CodeTitleLong},
	} {
		dir := newStore(t)
		if got := runCLI(t, dir, nil, "create", "--title", pad(tc.n),
			"--actor", "human:sothr"); got.code != exitOK {
			t.Fatalf("%d: create failed: %s%s", tc.n, got.stdout, got.stderr)
		}
		got := runCLI(t, dir, nil, "--json", "check")
		codes := titleCodes(t, got.stdout)
		switch {
		case tc.want == "" && len(codes) != 0:
			t.Errorf("%d characters: want no finding, got %v", tc.n, codes)
		case tc.want != "" && (len(codes) != 1 || codes[0] != tc.want):
			t.Errorf("%d characters: want [%s], got %v", tc.n, tc.want, codes)
		}
	}
}

// titleCodes pulls the title findings out of a check report, across both
// severities, so a code that moved between them is still seen.
func titleCodes(t *testing.T, stdout string) []string {
	t.Helper()
	env := decode(t, stdout)
	var out []string
	for _, key := range []string{"errors", "warnings"} {
		list, _ := env[key].([]any)
		for _, f := range list {
			m, _ := f.(map[string]any)
			code, _ := m["code"].(string)
			if strings.HasPrefix(code, "title") {
				out = append(out, code)
			}
		}
	}
	return out
}

// TestTitleOverMaxIsRefusedOnWrite covers the write-path half of 5.1. The cap is
// enforced where an invalid priority is, so a caller who just typed it is told
// now rather than by a check somebody runs later.
func TestTitleOverMaxIsRefusedOnWrite(t *testing.T) {
	dir := newStore(t)
	id := createTicket(t, dir)["ticket"].(map[string]any)["id"].(string)

	for _, tc := range []struct {
		name string
		args []string
	}{
		{"create", []string{"create", "--title", pad(ticket.TitleMax + 1)}},
		{"update", []string{"update", id, "--title", pad(ticket.TitleMax + 1)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := runCLI(t, dir, nil, append([]string{"--json"}, append(tc.args, "--actor", "human:sothr")...)...)
			if got.code != exitError {
				t.Fatalf("expected a refusal, got %d: %s%s", got.code, got.stdout, got.stderr)
			}
			errObj, _ := decode(t, got.stdout)["error"].(map[string]any)
			if code, _ := errObj["code"].(string); code != ticket.CodeInvalidField {
				t.Errorf("code is %q, want %q", code, ticket.CodeInvalidField)
			}
			// The message carries the actual length, because a caller three
			// over needs to know by how much.
			if msg, _ := errObj["message"].(string); !strings.Contains(msg, "121") {
				t.Errorf("the message does not say how long it was: %q", msg)
			}
		})
	}

	// Exactly the maximum is legal. The refusal is "over", not "at".
	if got := runCLI(t, dir, nil, "create", "--title", pad(ticket.TitleMax),
		"--actor", "human:sothr"); got.code != exitOK {
		t.Errorf("a title of exactly %d was refused: %s%s", ticket.TitleMax, got.stdout, got.stderr)
	}
}

// TestCheckNamesTheTicketTitle covers the human rendering. A finding names a
// file whose only identifier is a ULID, so the title is the part a person can
// act on.
func TestCheckNamesTheTicketTitle(t *testing.T) {
	dir := newStore(t)
	const title = "Rotate the signing key before the audit"
	if got := runCLI(t, dir, nil, "create", "--title", title, "--priority", "high",
		"--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("create failed: %s%s", got.stdout, got.stderr)
	}
	// A hand edit is the only way to a finding on an otherwise valid ticket,
	// which is the same shape the title-too-long fixture records.
	corrupt(t, dir, "priority: high", "priority: bogus")

	got := runCLI(t, dir, nil, "check")
	if !strings.Contains(got.stdout, title) {
		t.Errorf("the finding does not name the title:\n%s", got.stdout)
	}
	if !strings.Contains(got.stdout, "invalid_priority") {
		t.Errorf("expected an invalid_priority finding:\n%s", got.stdout)
	}
}

// TestCheckAbbreviatesALongTitle keeps a finding scannable. A title may run to
// TitleMax, and a row that wide defeats the reason the title is on the line.
func TestCheckAbbreviatesALongTitle(t *testing.T) {
	dir := newStore(t)
	long := "Decide whether the readiness reason should also say which dependency is blocking"
	if got := runCLI(t, dir, nil, "create", "--title", long,
		"--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("create failed: %s%s", got.stdout, got.stderr)
	}
	got := runCLI(t, dir, nil, "check")
	if strings.Contains(got.stdout, long) {
		t.Errorf("the full %d-character title was printed:\n%s", len(long), got.stdout)
	}
	if !strings.Contains(got.stdout, "…") {
		t.Errorf("an abbreviated title should be marked as cut:\n%s", got.stdout)
	}
	// Enough of it survives to recognise the ticket.
	if !strings.Contains(got.stdout, "Decide whether the readiness") {
		t.Errorf("too little of the title survived:\n%s", got.stdout)
	}
}

// TestErrorNamesTheTicketAndTitle covers the other place an ID appears alone.
// The message is explicitly the mutable half of an Error, so naming the ticket
// there is allowed and the code a caller switches on does not move.
func TestErrorNamesTheTicketAndTitle(t *testing.T) {
	dir := newStore(t)
	const title = "Rotate the signing key"
	got := runCLI(t, dir, nil, "--json", "create", "--title", title, "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("create failed: %s%s", got.stdout, got.stderr)
	}
	id := decode(t, got.stdout)["ticket"].(map[string]any)["id"].(string)

	got = runCLI(t, dir, nil, "update", id, "--if-revision", "sha256:deadbeef",
		"--title", "something else", "--actor", "human:sothr")
	if got.code != exitError {
		t.Fatalf("expected stale_revision, got %d: %s%s", got.code, got.stdout, got.stderr)
	}
	if !strings.Contains(got.stderr, id) {
		t.Errorf("the error does not name the ticket: %q", got.stderr)
	}
	if !strings.Contains(got.stderr, title) {
		t.Errorf("the error does not name the title: %q", got.stderr)
	}
}

// TestTicketNotFoundNamesTheIDOnce guards a regression from adding the ID to
// Error(). The message used to carry the ID as well, so the failure printed it
// twice in one sentence.
func TestTicketNotFoundNamesTheIDOnce(t *testing.T) {
	dir := newStore(t)
	const missing = "TKT-01K400GBC0WHK14CSBHQ8WSEPE"
	got := runCLI(t, dir, nil, "update", missing, "--priority", "high", "--actor", "human:sothr")
	if got.code != exitError {
		t.Fatalf("expected ticket_not_found, got %d: %s%s", got.code, got.stdout, got.stderr)
	}
	if n := strings.Count(got.stderr, missing); n != 1 {
		t.Errorf("the ID appears %d times, want 1: %q", n, got.stderr)
	}
}

// TestSchemaPublishesTitleLimits holds the numbers to the code that enforces
// them, so a threshold changed in one place cannot leave `schema` lying.
func TestSchemaPublishesTitleLimits(t *testing.T) {
	dir := t.TempDir() // schema reads no store
	got := runCLI(t, dir, nil, "--json", "schema")
	if got.code != exitOK {
		t.Fatalf("schema failed: %s%s", got.stdout, got.stderr)
	}
	limits, ok := decode(t, got.stdout)["titleLimits"].(map[string]any)
	if !ok {
		t.Fatalf("no titleLimits in the envelope: %s", got.stdout)
	}
	if warn, _ := limits["warn"].(float64); int(warn) != ticket.TitleWarn {
		t.Errorf("warn is %v, want %d", limits["warn"], ticket.TitleWarn)
	}
	if max, _ := limits["max"].(float64); int(max) != ticket.TitleMax {
		t.Errorf("max is %v, want %d", limits["max"], ticket.TitleMax)
	}
}

func TestAbbreviate(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"short", "short"},
		{strings.Repeat("a", 10), strings.Repeat("a", 10)},
		{strings.Repeat("a", 11), strings.Repeat("a", 9) + "…"},
		// Counted in runes, so a title needing more bytes is not cut shorter.
		{strings.Repeat("é", 10), strings.Repeat("é", 10)},
		{strings.Repeat("é", 11), strings.Repeat("é", 9) + "…"},
	} {
		if got := abbreviate(tc.in, 10); got != tc.want {
			t.Errorf("abbreviate(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
