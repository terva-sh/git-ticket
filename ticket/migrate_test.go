package ticket

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// schema1Store returns a store one level behind this binary, which is what a
// migration has to operate on. Init writes SchemaVersion, so the level is
// lowered deliberately here rather than found.
func schema1Store(t *testing.T) *Store {
	t.Helper()
	s := newTestStore(t)
	s.config.Schema = SchemaVersion - 1
	if err := os.WriteFile(filepath.Join(s.path, configFile), RenderConfig(s.config), 0o644); err != nil {
		t.Fatal(err)
	}
	return s
}

func storeSchemaLine(t *testing.T, s *Store) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(s.path, configFile))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "schema:") {
			return line
		}
	}
	t.Fatalf("config.yml declares no schema:\n%s", data)
	return ""
}

func fileLine(t *testing.T, path, prefix string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, prefix) {
			return line
		}
	}
	return ""
}

// insertUnknownField puts a key into a ticket file after extensions, which is
// where 5.3 renders unknown keys, so the file still round-trips.
func insertUnknownField(t *testing.T, path, text string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	const anchor = "extensions: {}\n"
	if !strings.Contains(string(data), anchor) {
		t.Fatalf("%s has no %q to anchor on", path, anchor)
	}
	out := strings.Replace(string(data), anchor, anchor+text, 1)
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestMigrateRaisesTheStore is plan 12.5. One pass converts config.yml and
// every ticket, and the fields the new level adds appear.
func TestMigrateRaisesTheStore(t *testing.T) {
	s := schema1Store(t)
	a := mustCreate(t, s, "First ticket to carry the new level")
	b := mustCreate(t, s, "Second ticket to carry the new level")

	// The provenance pair before the migration, so the assertion below is
	// against what the tickets actually said rather than against a guess.
	updatedA := fileLine(t, a.Path, "updated_at:")
	updatedByA := fileLine(t, a.Path, "updated_by:")
	if updatedA == "" {
		t.Fatal("the fixture ticket carries no updated_at")
	}

	res, err := s.Migrate(context.Background(), MigrateOptions{})
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if res.From != SchemaVersion-1 || res.To != SchemaVersion {
		t.Errorf("migrated %d to %d, want %d to %d", res.From, res.To, SchemaVersion-1, SchemaVersion)
	}
	if !res.ConfigChanged {
		t.Error("ConfigChanged is false on the run that raised the store")
	}
	if len(res.Tickets) != 2 {
		t.Errorf("rewrote %d tickets, want 2: %v", len(res.Tickets), res.Tickets)
	}
	if res.Skipped != 0 {
		t.Errorf("skipped %d, want 0", res.Skipped)
	}

	if got, want := storeSchemaLine(t, s), "schema: 2"; got != want {
		t.Errorf("config.yml says %q, want %q", got, want)
	}
	for _, tk := range []*Ticket{a, b} {
		if got, want := fileLine(t, tk.Path, "schema:"), "schema: 2"; got != want {
			t.Errorf("%s says %q, want %q", tk.ID, got, want)
		}
		// The field schema 2 adds is present and null, in the position 5.1
		// gives it. Its absence would mean the renderer ignored the new level.
		if got := fileLine(t, tk.Path, "origin:"); got != "origin: null" {
			t.Errorf("%s origin line = %q, want %q", tk.ID, got, "origin: null")
		}
	}

	// A migration is not a mutation by an actor. Stamping the pair would claim
	// the whole store was touched at one instant, which destroys the recency
	// order section 8 sorts by and puts a name on work nobody did.
	if got := fileLine(t, a.Path, "updated_at:"); got != updatedA {
		t.Errorf("updated_at moved: %q, want %q", got, updatedA)
	}
	if got := fileLine(t, a.Path, "updated_by:"); got != updatedByA {
		t.Errorf("updated_by moved: %q, want %q", got, updatedByA)
	}
}

// TestMigrateIsIdempotent is the 12.5 rule that a run interrupted by a crash or
// a full disk is finished by running it again. A ticket already at the target
// is skipped rather than rewritten.
func TestMigrateIsIdempotent(t *testing.T) {
	s := schema1Store(t)
	a := mustCreate(t, s, "A ticket that gets migrated once")

	if _, err := s.Migrate(context.Background(), MigrateOptions{}); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	first, err := os.ReadFile(a.Path)
	if err != nil {
		t.Fatal(err)
	}

	res, err := s.Migrate(context.Background(), MigrateOptions{})
	if err != nil {
		t.Fatalf("second migrate: %v", err)
	}
	if res.ConfigChanged {
		t.Error("the second run rewrote config.yml")
	}
	if len(res.Tickets) != 0 {
		t.Errorf("the second run rewrote %v", res.Tickets)
	}
	if res.Skipped != 1 {
		t.Errorf("skipped %d, want 1", res.Skipped)
	}

	second, err := os.ReadFile(a.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Errorf("the second run changed the bytes:\n%s", diffLines(string(first), string(second)))
	}
}

// TestMigrateWritesConfigBeforeTickets is the ordering 12.5 fixes. The two
// failure modes are not symmetric, so an interrupted migration must leave a
// store an old reader refuses outright rather than one it reads with tickets
// missing.
//
// The interruption is a ticket this pass cannot convert, which is deterministic
// and needs no permission trick that a root CI container would skip.
func TestMigrateWritesConfigBeforeTickets(t *testing.T) {
	s := schema1Store(t)
	a := mustCreate(t, s, "A ticket whose preserved key cannot be promoted")

	// origin is unknown at schema 1 and defined at schema 2, so migrating has
	// to promote it. A mapping is not a value the field can hold, so the pass
	// stops here rather than writing something nobody meant.
	insertUnknownField(t, a.Path, "origin:\n  not: a scalar\n")

	if _, err := s.Migrate(context.Background(), MigrateOptions{}); err == nil {
		t.Fatal("migrate succeeded on a ticket it cannot convert")
	}

	// The declaration moved even though the ticket did not. That is the point:
	// an old reader now refuses the whole store loudly instead of reading it
	// with this ticket silently missing.
	if got, want := storeSchemaLine(t, s), "schema: 2"; got != want {
		t.Errorf("config.yml says %q, want %q: it was not written first", got, want)
	}
	if got, want := fileLine(t, a.Path, "schema:"), "schema: 1"; got != want {
		t.Errorf("the unconvertible ticket says %q, want %q", got, want)
	}

	// And the run is resumable in the sense that matters: the config is already
	// where it belongs, so fixing the ticket and running again finishes the job.
	if got, want := storeSchemaLine(t, s), "schema: 2"; got != want {
		t.Errorf("config.yml = %q, want %q", got, want)
	}
}

// TestMigratePromotesAPreservedField covers the case that would otherwise write
// a duplicate key. A schema-1 file may carry origin as an unknown field, per
// 5.4, and at schema 2 that key is defined, so the migration has to move it
// into the field rather than leave it preserved beside a fresh null.
func TestMigratePromotesAPreservedField(t *testing.T) {
	s := schema1Store(t)
	a := mustCreate(t, s, "A ticket carrying tomorrow's field today")
	const source = "TKT-01K3ZYRB40N8QVD2M5T7XCFH3J"
	insertUnknownField(t, a.Path, "origin: "+source+"\n")

	if _, err := s.Migrate(context.Background(), MigrateOptions{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	data, err := os.ReadFile(a.Path)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(data), "origin:"); n != 1 {
		t.Errorf("the file carries %d origin keys, want 1:\n%s", n, data)
	}
	if got, want := fileLine(t, a.Path, "origin:"), "origin: "+source; got != want {
		t.Errorf("origin line = %q, want %q", got, want)
	}

	// It reads back as the field rather than as a preserved stranger.
	got, err := s.Get(context.Background(), a.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Origin == nil || *got.Origin != source {
		t.Errorf("Origin = %v, want %s", got.Origin, source)
	}
	if len(got.Unknown) != 0 {
		t.Errorf("origin is still an unknown field: %+v", got.Unknown)
	}
}

// TestFixDeclinesToMigrate is the one finding with exactly one correct repair
// that check --fix still declines, per plan 11 and 12.5. The repair is migrate,
// which rewrites every ticket in the store under the lock, and a store moves
// only through a migration a person runs. check --fix is what CI runs.
func TestFixDeclinesToMigrate(t *testing.T) {
	s := schema1Store(t)
	a := mustCreate(t, s, "A ticket the config left behind")

	// Put the declaration ahead of the files, which is exactly the shape 12.5's
	// ordering leaves when a migration is interrupted after config.yml.
	s.config.Schema = SchemaVersion
	if err := os.WriteFile(filepath.Join(s.path, configFile), RenderConfig(s.config), 0o644); err != nil {
		t.Fatal(err)
	}

	before, err := os.ReadFile(a.Path)
	if err != nil {
		t.Fatal(err)
	}

	report, err := s.Check(context.Background())
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !hasFinding(report.Warnings, CodeMigrationIncomplete) {
		t.Fatalf("check did not report %s: %+v", CodeMigrationIncomplete, report.Warnings)
	}
	if len(report.Errors) != 0 {
		t.Errorf("a store mid-migration reported errors: %+v", report.Errors)
	}

	res, err := s.Fix(context.Background(), FixOptions{})
	if err != nil {
		t.Fatalf("fix: %v", err)
	}
	for _, r := range res.Repairs {
		for _, c := range r.Codes {
			if c == CodeMigrationIncomplete {
				t.Errorf("--fix repaired %s, which is migrate's job", c)
			}
		}
	}

	// The file is untouched and the finding survives, so the store still says
	// what is wrong with it.
	after, err := os.ReadFile(a.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Errorf("--fix rewrote the ticket:\n%s", diffLines(string(before), string(after)))
	}
	if !hasFinding(res.Report.Warnings, CodeMigrationIncomplete) {
		t.Errorf("the finding vanished without being repaired: %+v", res.Report.Warnings)
	}
}

func hasFinding(fs []Finding, code string) bool {
	for _, f := range fs {
		if f.Code == code {
			return true
		}
	}
	return false
}

// TestMigrateRefusesADowngrade is 12.5: a field added in a later schema has
// nowhere to go in an earlier one, and a migration that quietly dropped it
// would lose work.
func TestMigrateRefusesADowngrade(t *testing.T) {
	s := schema1Store(t)
	mustCreate(t, s, "A ticket to carry along")
	if _, err := s.Migrate(context.Background(), MigrateOptions{}); err != nil {
		t.Fatalf("migrate up: %v", err)
	}

	_, err := s.Migrate(context.Background(), MigrateOptions{To: SchemaVersion - 1})
	if CodeOf(err) != CodeInvalidField {
		t.Errorf("downgrade = %v, want %s", err, CodeInvalidField)
	}
}

// TestMigrateRefusesALevelItDoesNotKnow keeps the target inside what this
// binary can write. Migrating to a level whose field set is unknown would
// stamp a number the reader cannot honour.
func TestMigrateRefusesALevelItDoesNotKnow(t *testing.T) {
	s := schema1Store(t)
	_, err := s.Migrate(context.Background(), MigrateOptions{To: SchemaVersion + 1})
	if CodeOf(err) != CodeSchemaUnsupported {
		t.Errorf("migrate to a future level = %v, want %s", err, CodeSchemaUnsupported)
	}
}

// TestMigrateDryRunWritesNothing plans the pass and touches no file, so a
// caller can see the size of the change before taking it.
func TestMigrateDryRunWritesNothing(t *testing.T) {
	s := schema1Store(t)
	a := mustCreate(t, s, "A ticket that stays where it is")
	before, err := os.ReadFile(a.Path)
	if err != nil {
		t.Fatal(err)
	}

	res, err := s.Migrate(context.Background(), MigrateOptions{DryRun: true})
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if len(res.Tickets) != 1 {
		t.Errorf("planned %v, want one ticket", res.Tickets)
	}
	if !res.ConfigChanged {
		t.Error("the plan does not include config.yml")
	}

	if got, want := storeSchemaLine(t, s), "schema: 1"; got != want {
		t.Errorf("config.yml says %q after a dry run, want %q", got, want)
	}
	after, err := os.ReadFile(a.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Errorf("a dry run rewrote the ticket:\n%s", diffLines(string(before), string(after)))
	}
}
