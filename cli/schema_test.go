package cli

import (
	"strings"
	"testing"

	"github.com/terva-sh/git-ticket/ticket"
)

// strsOf pulls a JSON array of strings out of a decoded envelope.
func strsOf(t *testing.T, env map[string]any, key string) []string {
	t.Helper()
	raw, ok := env[key].([]any)
	if !ok {
		t.Fatalf("%s is %T, want an array", key, env[key])
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		s, ok := v.(string)
		if !ok {
			t.Fatalf("%s holds a %T, want strings", key, v)
		}
		out = append(out, s)
	}
	return out
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestSchemaNeedsNoStore is the point of the command. A consumer asks what
// values are legal before it has anything to ask about, so schema must answer
// outside a repository and before init has run.
func TestSchemaNeedsNoStore(t *testing.T) {
	got := runCLI(t, t.TempDir(), nil, "schema")
	if got.code != exitOK {
		t.Fatalf("schema outside a store: %s", got.stderr)
	}
	if !strings.Contains(got.stdout, "transitions") {
		t.Errorf("the human form does not list the transitions: %q", got.stdout)
	}
}

// TestSchemaReportsWhatTheLibraryEnforces is the whole value of the command.
// Every list in the envelope comes from the code that enforces it, so a value
// added to the library and not to the plan still shows up here.
func TestSchemaReportsWhatTheLibraryEnforces(t *testing.T) {
	got := runCLI(t, t.TempDir(), nil, "--json", "schema")
	if got.code != exitOK {
		t.Fatalf("schema: %s", got.stderr)
	}
	env := decode(t, got.stdout)

	if env["kind"] != "schema" {
		t.Errorf("kind = %v, want schema", env["kind"])
	}
	if env["ticketSchema"] != float64(ticket.SchemaVersion) {
		t.Errorf("ticketSchema = %v, want %d", env["ticketSchema"], ticket.SchemaVersion)
	}

	for _, c := range []struct {
		key  string
		want []string
	}{
		{"statuses", ticket.Statuses},
		{"openStatuses", ticket.OpenStatuses},
		{"types", ticket.Types},
		{"priorities", ticket.Priorities},
	} {
		if got := strsOf(t, env, c.key); !sameStrings(got, c.want) {
			t.Errorf("%s = %v, want %v", c.key, got, c.want)
		}
	}

	// Every status has an entry, and each entry is what SetStatus will allow.
	trans, ok := env["transitions"].(map[string]any)
	if !ok {
		t.Fatalf("transitions is %T, want an object", env["transitions"])
	}
	if len(trans) != len(ticket.Statuses) {
		t.Errorf("transitions has %d entries, want one per status (%d)", len(trans), len(ticket.Statuses))
	}
	for _, s := range ticket.Statuses {
		if _, ok := trans[s]; !ok {
			t.Errorf("transitions has no entry for %s", s)
			continue
		}
		want := ticket.PermittedTransitions(s)
		if got := strsOf(t, trans, s); !sameStrings(got, want) {
			t.Errorf("transitions[%s] = %v, want %v", s, got, want)
		}
	}

	// usage is the CLI's own, per plan section 10, so it is here and not in
	// the library's list.
	codes := strsOf(t, env, "errorCodes")
	if len(codes) != len(ticket.OperationCodes)+1 || codes[len(codes)-1] != codeUsage {
		t.Errorf("errorCodes = %v, want the library's codes then %s", codes, codeUsage)
	}

	severity := map[string]string{}
	raw, ok := env["findingCodes"].([]any)
	if !ok {
		t.Fatalf("findingCodes is %T, want an array", env["findingCodes"])
	}
	for _, v := range raw {
		f := v.(map[string]any)
		severity[f["code"].(string)] = f["severity"].(string)
	}
	for _, c := range ticket.CheckErrorCodes {
		if severity[c] != "error" {
			t.Errorf("%s published as %q, want error", c, severity[c])
		}
	}
	for _, c := range ticket.CheckWarningCodes {
		if severity[c] != "warning" {
			t.Errorf("%s published as %q, want warning", c, severity[c])
		}
	}
	if want := len(ticket.CheckErrorCodes) + len(ticket.CheckWarningCodes); len(severity) != want {
		t.Errorf("findingCodes has %d entries, want %d", len(severity), want)
	}
}

// TestEveryEmittedKindIsPublished is the drift guard on the kinds list. The
// list is written out by hand because it is a contract rather than a fact about
// a Go type, so this runs one command per kind and holds the answers to it. A
// new kind added without touching envelopeKinds fails here.
//
// The guard is only as complete as the table below. `version` shipped in
// v0.4.0 absent from both envelopeKinds and this table, so the two hand-written
// lists agreed with each other and not with plan 10.1, and nothing failed. Add
// the command that emits a kind here in the same change that adds the kind.
func TestEveryEmittedKindIsPublished(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir))

	published := map[string]bool{}
	for _, k := range strsOf(t, decode(t, runCLI(t, dir, nil, "--json", "schema").stdout), "kinds") {
		published[k] = true
	}

	seen := map[string]bool{}
	for _, c := range []struct {
		what string
		args []string
	}{
		{"a ticket", []string{"--json", "show", id}},
		{"a listing", []string{"--json", "list"}},
		{"a mutation", []string{"--json", "update", id, "--priority", "high", "--actor", "human:sothr"}},
		{"a check", []string{"--json", "check"}},
		{"a failure", []string{"--json", "show", "TKT-01ZZZZZZZZZZZZZZZZZZZZZZZZ"}},
		{"the schema", []string{"--json", "schema"}},
		{"the instructions", []string{"--json", "instructions"}},
		// --version is top level only, so it goes before any subcommand.
		{"the version", []string{"--json", "--version"}},
	} {
		out := runCLI(t, dir, nil, c.args...).stdout
		kind, ok := decode(t, out)["kind"].(string)
		if !ok {
			t.Errorf("%s emitted no kind: %q", c.what, out)
			continue
		}
		seen[kind] = true
		if !published[kind] {
			t.Errorf("%s emits kind %q, which schema does not publish", c.what, kind)
		}
	}

	for k := range published {
		if !seen[k] {
			t.Errorf("schema publishes kind %q, but no command in this test emits it", k)
		}
	}
}

func TestSchemaTakesNoArguments(t *testing.T) {
	got := runCLI(t, t.TempDir(), nil, "--json", "schema", "extra")
	if got.code != exitError {
		t.Fatal("a stray argument should be refused")
	}
	if code := errCode(t, got); code != codeUsage {
		t.Errorf("code = %v, want %s", code, codeUsage)
	}
}
