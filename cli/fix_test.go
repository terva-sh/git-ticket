package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// newRepoStore returns a store inside a real repository.
//
// A bare temp directory is not one, and displayPath has no root to measure
// against there, so every path in the report comes back absolute. A test that
// asserts the repository-relative form has to be given a repository. Matching
// the suffix instead would pass either way and prove nothing.
func newRepoStore(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, so there is no repository root to resolve against")
	}
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	if got := runCLI(t, dir, nil, "init", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("init: %s", got.stderr)
	}
	return dir
}

// misname moves a ticket file to a name that is not its ID, which is the
// filename_id_mismatch of plan section 11 and one of the two findings --fix
// repairs.
func misname(t *testing.T, dir, id, as string) (from, to string) {
	t.Helper()
	// Rename within whatever directory the status put it in, so this stays a
	// filename_id_mismatch alone and does not also become a location_mismatch.
	from = ticketFile(t, dir, id)
	to = filepath.Join(filepath.Dir(from), as)
	if err := os.Rename(from, to); err != nil {
		t.Fatal(err)
	}
	return to, from
}

// TestCheckFixRepairsAndNamesThePaths covers the whole contract of --fix. It is
// the first command that writes without being told which ticket to write, so
// the report has to name every file it touched the way a mutation does.
func TestCheckFixRepairsAndNamesThePaths(t *testing.T) {
	dir := newRepoStore(t)
	id := ticketID(t, createTicket(t, dir))
	wrong, right := misname(t, dir, id, "notes-about-auth.md")

	got := runCLI(t, dir, nil, "--json", "check", "--fix")
	if got.code != exitOK {
		t.Fatalf("the store should be clean once repaired: %s%s", got.stdout, got.stderr)
	}
	env := decode(t, got.stdout)
	if env["kind"] != "check-report" {
		t.Errorf("kind = %v, want check-report", env["kind"])
	}
	if env["dryRun"] != false {
		t.Errorf("dryRun = %v, want false", env["dryRun"])
	}

	repairs, ok := env["repairs"].([]any)
	if !ok || len(repairs) != 1 {
		t.Fatalf("repairs = %v, want one", env["repairs"])
	}
	r := repairs[0].(map[string]any)
	if r["ticket"] != id {
		t.Errorf("ticket = %v, want %s", r["ticket"], id)
	}
	// The ticket is a draft, so both ends are in draft/, per section 4. A repair
	// that crossed directories would be a location_mismatch too, and the codes
	// below assert this one is not.
	if r["from"] != ".tickets/draft/notes-about-auth.md" {
		t.Errorf("from = %v", r["from"])
	}
	if r["to"] != ".tickets/draft/"+id+".md" {
		t.Errorf("to = %v", r["to"])
	}
	codes := r["codes"].([]any)
	if len(codes) != 1 || codes[0] != "filename_id_mismatch" {
		t.Errorf("codes = %v, want filename_id_mismatch alone", codes)
	}

	// Both ends of the move, because both files changed.
	paths := env["pathsChanged"].([]any)
	if len(paths) != 2 {
		t.Fatalf("pathsChanged = %v, want both ends of the move", paths)
	}

	if _, err := os.Stat(right); err != nil {
		t.Errorf("the file is not at its repaired path: %v", err)
	}
	if _, err := os.Stat(wrong); !os.IsNotExist(err) {
		t.Error("the file is still at its old path")
	}
}

// TestCheckFixDryRunWritesNothing is what makes --fix safe to reach for. The
// repairs are reported and the findings they would clear are still in errors,
// because nothing was written and the store still has them.
func TestCheckFixDryRunWritesNothing(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir))
	wrong, _ := misname(t, dir, id, "notes-about-auth.md")

	got := runCLI(t, dir, nil, "--json", "check", "--fix", "--dry-run")
	if got.code != exitError {
		t.Fatal("a dry run repairs nothing, so the store still fails")
	}
	env := decode(t, got.stdout)
	if env["dryRun"] != true {
		t.Errorf("dryRun = %v, want true", env["dryRun"])
	}
	if repairs := env["repairs"].([]any); len(repairs) != 1 {
		t.Errorf("a dry run should still plan the repair: %v", repairs)
	}
	// Nothing moved, so nothing changed.
	if paths := env["pathsChanged"].([]any); len(paths) != 0 {
		t.Errorf("pathsChanged = %v, want empty on a dry run", paths)
	}
	errs := env["errors"].([]any)
	if len(errs) != 1 || errs[0].(map[string]any)["code"] != "filename_id_mismatch" {
		t.Errorf("errors = %v, want the finding still standing", errs)
	}

	if _, err := os.Stat(wrong); err != nil {
		t.Errorf("a dry run moved the file: %v", err)
	}
}

// TestCheckWithoutFixReportsNoRepairs holds the two new fields to the rule that
// an absent collection is an empty array and never omitted, so a consumer never
// has to tell missing from empty.
func TestCheckWithoutFixReportsNoRepairs(t *testing.T) {
	dir := newStore(t)
	createTicket(t, dir)

	got := runCLI(t, dir, nil, "--json", "check")
	if got.code != exitOK {
		t.Fatalf("check: %s%s", got.stdout, got.stderr)
	}
	env := decode(t, got.stdout)
	for _, key := range []string{"repairs", "pathsChanged"} {
		v, ok := env[key]
		if !ok {
			t.Errorf("%s is missing from the report", key)
			continue
		}
		if list, ok := v.([]any); !ok || len(list) != 0 {
			t.Errorf("%s = %v, want an empty array", key, v)
		}
	}
	if env["dryRun"] != false {
		t.Errorf("dryRun = %v, want false", env["dryRun"])
	}
}

// TestCheckDryRunNeedsFix refuses a flag that would otherwise do nothing and
// look like a clean store, which is a worse answer than a refusal.
func TestCheckDryRunNeedsFix(t *testing.T) {
	dir := newStore(t)

	got := runCLI(t, dir, nil, "--json", "check", "--dry-run")
	if got.code != exitError {
		t.Fatal("--dry-run without --fix should be refused")
	}
	if code := errCode(t, got); code != codeUsage {
		t.Errorf("code = %v, want %s", code, codeUsage)
	}
}

// TestCheckFixLeavesAJudgementAlone is the scope of --fix in one test. Two
// files holding one ID both belong at the same path, and which is the real
// ticket is a question only a person can answer, so neither moves.
func TestCheckFixLeavesAJudgementAlone(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir))

	real := ticketFile(t, dir, id)
	data, err := os.ReadFile(real)
	if err != nil {
		t.Fatal(err)
	}
	impostor := filepath.Join(filepath.Dir(real), "copy-of-it.md")
	if err := os.WriteFile(impostor, data, 0o644); err != nil {
		t.Fatal(err)
	}

	got := runCLI(t, dir, nil, "--json", "check", "--fix")
	if got.code != exitError {
		t.Fatal("a duplicate id is not repairable, so the check still fails")
	}
	env := decode(t, got.stdout)
	if repairs := env["repairs"].([]any); len(repairs) != 0 {
		t.Errorf("nothing should move onto an occupied path: %v", repairs)
	}
	if _, err := os.Stat(impostor); err != nil {
		t.Errorf("the second file was moved anyway: %v", err)
	}
	if _, err := os.Stat(real); err != nil {
		t.Errorf("the first file was moved anyway: %v", err)
	}
}
