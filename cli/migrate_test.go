package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/terva-sh/git-ticket/ticket"
)

// lowerSchema puts a store at schema 1. init writes the current level, so the
// declaration is lowered first and the tickets are made after, because create
// stamps what the store declares, per plan 12.5.
func lowerSchema(t *testing.T, dir string) string {
	t.Helper()
	cfg := filepath.Join(dir, ".tickets", "config.yml")
	data, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	now := fmt.Sprintf("schema: %d", ticket.SchemaVersion)
	// Schema 1, not one level behind. These tests migrate a store and assert on
	// what the run reported, and a relative level stopped meaning "the oldest
	// store" as soon as schema 3 existed.
	was := "schema: 1"
	out := strings.Replace(string(data), now, was, 1)
	if out == string(data) {
		t.Fatalf("config.yml does not declare %q:\n%s", now, data)
	}
	if err := os.WriteFile(cfg, []byte(out), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func schema1Store(t *testing.T) string {
	t.Helper()
	return lowerSchema(t, newStore(t))
}

// TestMigrateEnvelope is the migrate-result kind of plan 10.8. Section 10 wants
// a test per kind, and this is the one for this kind.
func TestMigrateEnvelope(t *testing.T) {
	// A real repository, because displayPath has no root to measure against in
	// a bare temp directory and every path would come back absolute. Matching a
	// suffix instead would pass either way and prove nothing, which is the
	// reason newRepoStore exists.
	dir := lowerSchema(t, newRepoStore(t))
	createTicket(t, dir)

	got := runCLI(t, dir, nil, "--json", "migrate")
	if got.code != exitOK {
		t.Fatalf("migrate failed: %s%s", got.stdout, got.stderr)
	}
	env := decode(t, got.stdout)

	if env["kind"] != "migrate-result" {
		t.Errorf("kind = %v, want migrate-result", env["kind"])
	}
	// 1, because lowerSchema puts the store there rather than one level back.
	if from, want := env["from"], float64(1); from != want {
		t.Errorf("from = %v, want %v", from, want)
	}
	if to, want := env["to"], float64(ticket.SchemaVersion); to != want {
		t.Errorf("to = %v, want %v", to, want)
	}
	if env["configChanged"] != true {
		t.Errorf("configChanged = %v, want true", env["configChanged"])
	}
	if env["skipped"] != float64(0) {
		t.Errorf("skipped = %v, want 0", env["skipped"])
	}
	// Absent collections are [] and never omitted, per section 10, so a
	// consumer never distinguishes missing from empty.
	for _, key := range []string{"tickets", "unreadable"} {
		if _, ok := env[key].([]any); !ok {
			t.Errorf("%s = %v, want an array", key, env[key])
		}
	}

	tickets := strsOf(t, env, "tickets")
	if len(tickets) != 1 {
		t.Fatalf("tickets = %v, want one file", tickets)
	}
	// Every path in the envelope is relative to the repository root, per
	// section 10, and the library reports these relative to the store.
	if !strings.HasPrefix(tickets[0], ".tickets/") {
		t.Errorf("tickets[0] = %q, want a repository-relative path", tickets[0])
	}
}

// TestMigrateDryRunExitsZero is the ruling in 10.8. A pending migration is not
// a failure, because 12.5 makes a store move only through a migration a person
// runs, so gating a job here would gate on a decision no job may take. check is
// where CI learns a store is behind.
func TestMigrateDryRunExitsZero(t *testing.T) {
	dir := schema1Store(t)
	createTicket(t, dir)

	got := runCLI(t, dir, nil, "--json", "migrate", "--dry-run")
	if got.code != exitOK {
		t.Fatalf("a pending migration under --dry-run exited %d, want %d", got.code, exitOK)
	}
	env := decode(t, got.stdout)
	if len(strsOf(t, env, "tickets")) != 1 {
		t.Errorf("the dry run planned %v, want one ticket", env["tickets"])
	}

	// And it wrote nothing, so the store still declares the old level.
	after := runCLI(t, dir, nil, "--json", "migrate", "--dry-run")
	if decode(t, after.stdout)["configChanged"] != true {
		t.Error("the dry run changed the store")
	}
}

// TestMigrateSecondRunHasNothingToDo is the idempotence of 12.5 seen through
// the envelope: the same level twice, nothing rewritten, and the tickets
// counted as skipped.
func TestMigrateSecondRunHasNothingToDo(t *testing.T) {
	dir := schema1Store(t)
	createTicket(t, dir)
	if got := runCLI(t, dir, nil, "--json", "migrate"); got.code != exitOK {
		t.Fatalf("first migrate: %s%s", got.stdout, got.stderr)
	}

	got := runCLI(t, dir, nil, "--json", "migrate")
	if got.code != exitOK {
		t.Fatalf("second migrate: %s%s", got.stdout, got.stderr)
	}
	env := decode(t, got.stdout)
	if env["from"] != env["to"] {
		t.Errorf("from %v to %v, want the same level twice", env["from"], env["to"])
	}
	if env["configChanged"] != false {
		t.Error("the second run rewrote config.yml")
	}
	if n := len(strsOf(t, env, "tickets")); n != 0 {
		t.Errorf("the second run rewrote %d tickets, want 0", n)
	}
	if env["skipped"] != float64(1) {
		t.Errorf("skipped = %v, want 1", env["skipped"])
	}
}

// TestMigrateHumanOutputNamesWhatMoved keeps the human form useful: a person
// running this wants to know the store moved and which files it touched.
func TestMigrateHumanOutputNamesWhatMoved(t *testing.T) {
	dir := schema1Store(t)
	createTicket(t, dir)

	got := runCLI(t, dir, nil, "migrate")
	if got.code != exitOK {
		t.Fatalf("migrate failed: %s%s", got.stdout, got.stderr)
	}
	moved := fmt.Sprintf("migrated schema 1 to %d", ticket.SchemaVersion)
	for _, want := range []string{moved, "config.yml", ".tickets/"} {
		if !strings.Contains(got.stdout, want) {
			t.Errorf("output does not mention %q:\n%s", want, got.stdout)
		}
	}

	// The second run says so rather than printing nothing, because silence
	// reads as a failure to a person who just asked for work to happen.
	again := runCLI(t, dir, nil, "migrate")
	if !strings.Contains(again.stdout, fmt.Sprintf("already at schema %d", ticket.SchemaVersion)) {
		t.Errorf("a second run said:\n%s", again.stdout)
	}
}

// TestMigrateRefusesADowngrade is 12.5. A field a later schema added has
// nowhere to go in an earlier one.
func TestMigrateRefusesADowngrade(t *testing.T) {
	dir := schema1Store(t)
	createTicket(t, dir)
	if got := runCLI(t, dir, nil, "migrate"); got.code != exitOK {
		t.Fatalf("migrate up: %s%s", got.stdout, got.stderr)
	}

	got := runCLI(t, dir, nil, "--json", "migrate", "--to", "1")
	if got.code != exitError {
		t.Fatalf("a downgrade exited %d, want %d", got.code, exitError)
	}
	if code := errCode(t, got); code != "invalid_field" {
		t.Errorf("code = %s, want invalid_field", code)
	}
}

// TestMigrateTakesNoArguments matches every other whole-store command: it
// converts the store, so there is nothing to name.
func TestMigrateTakesNoArguments(t *testing.T) {
	dir := newStore(t)
	got := runCLI(t, dir, nil, "--json", "migrate", "extra")
	if got.code != exitError {
		t.Fatal("a stray argument should be refused")
	}
	if code := errCode(t, got); code != codeUsage {
		t.Errorf("code = %v, want %s", code, codeUsage)
	}
}
