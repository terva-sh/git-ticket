package cli

import (
	"strings"
	"testing"
)

// seriesEnvelopeOf runs a series command and returns the decoded envelope,
// after checking it is the kind 10.9 promises.
func seriesEnvelopeOf(t *testing.T, dir string, args ...string) map[string]any {
	t.Helper()
	got := runCLI(t, dir, nil, append([]string{"--json", "series"}, args...)...)
	if got.code != exitOK {
		t.Fatalf("series %v exited %d: %s%s", args, got.code, got.stdout, got.stderr)
	}
	envelope := decode(t, got.stdout)
	if envelope["kind"] != "series" {
		t.Fatalf("kind = %v, want series", envelope["kind"])
	}
	return envelope
}

// seriesList reads the list out of a series envelope.
func seriesList(t *testing.T, envelope map[string]any) []string {
	t.Helper()
	raw, ok := envelope["series"].([]any)
	if !ok {
		t.Fatalf("no series array in %v", envelope)
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		s, _ := v.(string)
		out = append(out, s)
	}
	return out
}

// TestSeriesAnswersOneKindForAReadAndTwoWrites is the shape 10.9 settles. A
// caller that ran add to make sure a series exists wants exactly what the bare
// read returns, so all three answer the same way and `changed` is what
// separates them.
func TestSeriesAnswersOneKindForAReadAndTwoWrites(t *testing.T) {
	dir := newStore(t)

	read := seriesEnvelopeOf(t, dir)
	if got := strings.Join(seriesList(t, read), ","); got != "TKT" {
		t.Errorf("a fresh store lists %q, want TKT", got)
	}
	if read["changed"] != false {
		t.Errorf("a read reported changed=%v, want false", read["changed"])
	}
	if paths, _ := read["pathsChanged"].([]any); len(paths) != 0 {
		t.Errorf("a read reported pathsChanged=%v, want empty", paths)
	}

	added := seriesEnvelopeOf(t, dir, "add", "IDEA")
	if got := strings.Join(seriesList(t, added), ","); got != "TKT,IDEA" {
		t.Fatalf("after add the store lists %q, want TKT,IDEA", got)
	}
	if added["changed"] != true {
		t.Errorf("add reported changed=%v, want true", added["changed"])
	}
	paths, _ := added["pathsChanged"].([]any)
	if len(paths) != 1 || !strings.HasSuffix(paths[0].(string), "config.yml") {
		t.Errorf("add reported pathsChanged=%v, want config.yml", paths)
	}

	// The same add again is a no-op rather than an error, so a caller running
	// it to be sure does not have to tell one kind of success from another.
	again := seriesEnvelopeOf(t, dir, "add", "IDEA")
	if again["changed"] != false {
		t.Errorf("a repeated add reported changed=%v, want false", again["changed"])
	}
	if got := strings.Join(seriesList(t, again), ","); got != "TKT,IDEA" {
		t.Errorf("a repeated add lists %q, want TKT,IDEA", got)
	}

	removed := seriesEnvelopeOf(t, dir, "remove", "IDEA")
	if got := strings.Join(seriesList(t, removed), ","); got != "TKT" {
		t.Errorf("after remove the store lists %q, want TKT", got)
	}
	if removed["changed"] != true {
		t.Errorf("remove reported changed=%v, want true", removed["changed"])
	}
}

// TestSeriesAddIsWhatCreateThenAccepts closes the loop. A declaration nothing
// downstream honours is a config edit with no consequence, so the test that
// matters is that create refuses before and succeeds after.
func TestSeriesAddIsWhatCreateThenAccepts(t *testing.T) {
	dir := newStore(t)

	before := runCLI(t, dir, nil, "--json", "create",
		"--title", "An idea, before the store declares one",
		"--series", "IDEA", "--actor", "human:sothr")
	if before.code != exitError {
		t.Fatalf("create --series IDEA should be refused first, got exit %d", before.code)
	}
	if code := errCode(t, before); code != "unknown_series" {
		t.Errorf("code = %v, want unknown_series", code)
	}

	seriesEnvelopeOf(t, dir, "add", "IDEA")

	after := runCLI(t, dir, nil, "--json", "create",
		"--title", "An idea, once the store declares one",
		"--series", "IDEA", "--actor", "human:sothr")
	if after.code != exitOK {
		t.Fatalf("create --series IDEA after declaring it: %s%s", after.stdout, after.stderr)
	}
	id := ticketID(t, decode(t, after.stdout))
	if !strings.HasPrefix(id, "IDEA-") {
		t.Errorf("created %s, want an IDEA prefix", id)
	}
}

// TestSeriesRemoveRefusesWhileTicketsCarryIt. Undeclaring a series in use would
// report every one of those tickets as unknown_series at the next check, so the
// refusal names the count and the caller decides.
func TestSeriesRemoveRefusesWhileTicketsCarryIt(t *testing.T) {
	dir := newStore(t)
	seriesEnvelopeOf(t, dir, "add", "IDEA")
	makeTicket(t, dir, "An idea worth keeping", "--series", "IDEA")

	got := runCLI(t, dir, nil, "--json", "series", "remove", "IDEA")
	if got.code != exitError {
		t.Fatalf("remove should be refused while a ticket carries it, got exit %d", got.code)
	}
	if code := errCode(t, got); code != "validation_failed" {
		t.Errorf("code = %v, want validation_failed", code)
	}
	if !strings.Contains(got.stderr, "1 ticket") {
		t.Errorf("the refusal does not name the count: %s", got.stderr)
	}
}

// TestSeriesRefusesWhatIsNotACommand keeps the argument shapes honest. A typo
// in the subcommand must not be read as a series name, and a name with no
// subcommand must not be read as a bare read.
func TestSeriesRefusesWhatIsNotACommand(t *testing.T) {
	dir := newStore(t)
	for _, args := range [][]string{
		{"delete", "IDEA"},
		{"add"},
		{"remove"},
		{"add", "IDEA", "extra"},
	} {
		got := runCLI(t, dir, nil, append([]string{"--json", "series"}, args...)...)
		if got.code != exitError {
			t.Errorf("series %v exited %d, want a refusal", args, got.code)
			continue
		}
		if code := errCode(t, got); code != codeUsage {
			t.Errorf("series %v: code = %v, want %s", args, code, codeUsage)
		}
	}

	// A malformed name is invalid_field rather than usage: the shape of the
	// command was right and the value was not.
	bad := runCLI(t, dir, nil, "--json", "series", "add", "2FA")
	if bad.code != exitError {
		t.Fatalf("a leading digit should be refused, got exit %d", bad.code)
	}
	if code := errCode(t, bad); code != "invalid_field" {
		t.Errorf("code = %v, want invalid_field", code)
	}
}

// TestSeriesUppercasesTheName. The grammar is uppercase and a reference
// resolves case-insensitively, per 5.5, so somebody typing `series add idea`
// meant IDEA and gets it rather than a grammar complaint.
func TestSeriesUppercasesTheName(t *testing.T) {
	dir := newStore(t)
	added := seriesEnvelopeOf(t, dir, "add", "idea")
	if got := strings.Join(seriesList(t, added), ","); got != "TKT,IDEA" {
		t.Errorf("series add idea produced %q, want TKT,IDEA", got)
	}
}

// TestSchemaPublishesTheSeriesGrammar is 10.4. The bounds are a fact about the
// binary rather than about a store, so a consumer learns what a legal prefix
// looks like before it opens anything, which is why this reads without a store.
func TestSchemaPublishesTheSeriesGrammar(t *testing.T) {
	dir := t.TempDir()
	got := runCLI(t, dir, nil, "--json", "schema")
	if got.code != exitOK {
		t.Fatalf("schema outside a store exited %d: %s", got.code, got.stderr)
	}
	limits, ok := decode(t, got.stdout)["seriesLimits"].(map[string]any)
	if !ok {
		t.Fatal("schema publishes no seriesLimits")
	}
	if limits["minLength"] != float64(2) || limits["maxLength"] != float64(8) {
		t.Errorf("bounds = %v and %v, want 2 and 8", limits["minLength"], limits["maxLength"])
	}
	if limits["pattern"] != "^[A-Z][A-Z0-9]{1,7}$" {
		t.Errorf("pattern = %v", limits["pattern"])
	}
}
