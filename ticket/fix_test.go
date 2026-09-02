package ticket

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// moveFile puts a file somewhere it does not belong, which is how these tests
// build the two conditions Fix repairs.
func moveFile(t *testing.T, from, to string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(from, to); err != nil {
		t.Fatal(err)
	}
}

func mustFix(t *testing.T, s *Store, o FixOptions) *FixResult {
	t.Helper()
	res, err := s.Fix(context.Background(), o)
	if err != nil {
		t.Fatalf("fix: %v", err)
	}
	return res
}

// codesOf collects the codes of a finding list, so a test says which findings
// it expects rather than indexing into the slice.
func codesOf(fs []Finding) []string {
	out := make([]string, 0, len(fs))
	for _, f := range fs {
		out = append(out, f.Code)
	}
	return out
}

func has(codes []string, want string) bool {
	for _, c := range codes {
		if c == want {
			return true
		}
	}
	return false
}

// TestFixRenamesAFileToItsID covers filename_id_mismatch, which plan 4 leaves
// exactly one reading of: the file is named for the ID it holds.
//
// It also pins that the repair moves the bytes rather than re-rendering. Both
// findings are about where a file sits, so a pass that rewrote its contents
// would be doing something the caller did not ask for.
func TestFixRenamesAFileToItsID(t *testing.T) {
	s := newTestStore(t)
	tk := mustCreate(t, s, "Named wrong on disk")

	before, err := os.ReadFile(tk.Path)
	if err != nil {
		t.Fatal(err)
	}
	wrong := filepath.Join(s.TicketsDir(), "notes-about-auth.md")
	moveFile(t, tk.Path, wrong)

	// The condition is there before the repair.
	pre, err := s.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !has(codesOf(pre.Errors), CodeFilenameIDMismatch) {
		t.Fatalf("expected a filename_id_mismatch to repair, got %v", codesOf(pre.Errors))
	}

	res := mustFix(t, s, FixOptions{})
	if len(res.Repairs) != 1 {
		t.Fatalf("got %d repairs, want 1: %+v", len(res.Repairs), res.Repairs)
	}
	r := res.Repairs[0]
	if r.From != "tickets/notes-about-auth.md" || r.To != "tickets/"+tk.ID+".md" {
		t.Errorf("repair = %+v", r)
	}
	if !has(r.Codes, CodeFilenameIDMismatch) {
		t.Errorf("codes = %v, want filename_id_mismatch", r.Codes)
	}
	if !res.Report.OK() {
		t.Errorf("the store should be clean after the repair: %v", codesOf(res.Report.Errors))
	}

	after, err := os.ReadFile(filepath.Join(s.TicketsDir(), tk.ID+".md"))
	if err != nil {
		t.Fatalf("the file is not at its repaired path: %v", err)
	}
	if string(after) != string(before) {
		t.Error("the repair rewrote the ticket instead of moving it")
	}
	if _, err := os.Stat(wrong); !os.IsNotExist(err) {
		t.Error("the file is still at its old path")
	}
}

// TestFixMovesAFileTheStatusContradicts covers archive_location_mismatch. Plan
// 6.3 already rules that the status wins when the status and the directory
// disagree, so the move follows the status and there is nothing to decide.
func TestFixMovesAFileTheStatusContradicts(t *testing.T) {
	s := newTestStore(t)
	tk := mustCreate(t, s, "Archived but sitting in tickets")
	mustApply(t, s, tk.ID, SetStatus{Status: StatusReady})
	mustApply(t, s, tk.ID, SetStatus{Status: StatusInProgress})
	mustApply(t, s, tk.ID, SetStatus{Status: StatusDone})
	res := mustApply(t, s, tk.ID, ArchiveTicket{Reason: "shipped"})

	// Put it back where an archived ticket does not belong.
	stray := filepath.Join(s.TicketsDir(), tk.ID+".md")
	moveFile(t, res.Ticket.Path, stray)

	fixed := mustFix(t, s, FixOptions{})
	if len(fixed.Repairs) != 1 {
		t.Fatalf("got %d repairs, want 1: %+v", len(fixed.Repairs), fixed.Repairs)
	}
	r := fixed.Repairs[0]
	if r.From != "tickets/"+tk.ID+".md" || r.To != "archive/"+tk.ID+".md" {
		t.Errorf("repair = %+v", r)
	}
	if !has(r.Codes, CodeArchiveLocationMismatch) {
		t.Errorf("codes = %v, want archive_location_mismatch", r.Codes)
	}
	if !fixed.Report.OK() {
		t.Errorf("the store should be clean after the repair: %v", codesOf(fixed.Report.Errors))
	}
	if _, err := os.Stat(filepath.Join(s.ArchiveDir(), tk.ID+".md")); err != nil {
		t.Errorf("the ticket is not in the archive: %v", err)
	}
}

// TestFixSettlesBothFaultsInOneMove is why Repair carries a list of codes. A
// file can be in the wrong directory under the wrong name, and one move is the
// whole repair for both findings.
func TestFixSettlesBothFaultsInOneMove(t *testing.T) {
	s := newTestStore(t)
	tk := mustCreate(t, s, "Wrong name and wrong directory")

	moveFile(t, tk.Path, filepath.Join(s.ArchiveDir(), "stale-name.md"))

	res := mustFix(t, s, FixOptions{})
	if len(res.Repairs) != 1 {
		t.Fatalf("got %d repairs, want 1: %+v", len(res.Repairs), res.Repairs)
	}
	r := res.Repairs[0]
	if len(r.Codes) != 2 || !has(r.Codes, CodeFilenameIDMismatch) || !has(r.Codes, CodeArchiveLocationMismatch) {
		t.Errorf("codes = %v, want both findings", r.Codes)
	}
	if r.To != "tickets/"+tk.ID+".md" {
		t.Errorf("to = %q", r.To)
	}
	if !res.Report.OK() {
		t.Errorf("the store should be clean: %v", codesOf(res.Report.Errors))
	}
}

// TestFixDryRunPlansAndWritesNothing is what makes the command safe to run
// first. It is the only command that writes without being told which ticket to
// write, so a caller has to be able to see the moves before they happen.
func TestFixDryRunPlansAndWritesNothing(t *testing.T) {
	s := newTestStore(t)
	tk := mustCreate(t, s, "Named wrong on disk")
	wrong := filepath.Join(s.TicketsDir(), "notes-about-auth.md")
	moveFile(t, tk.Path, wrong)

	res := mustFix(t, s, FixOptions{DryRun: true})
	if len(res.Repairs) != 1 {
		t.Fatalf("a dry run should still plan the repair: %+v", res.Repairs)
	}
	if _, err := os.Stat(wrong); err != nil {
		t.Errorf("a dry run moved the file: %v", err)
	}
	// Nothing was written, so the finding is still there to report.
	if !has(codesOf(res.Report.Errors), CodeFilenameIDMismatch) {
		t.Errorf("errors = %v, want the finding still standing", codesOf(res.Report.Errors))
	}
}

// TestFixRefusesToClobber is the duplicate_id judgement it declines to make.
// Two files holding one ID both want the same destination, and which of them is
// the real ticket is a question only a person can answer. Neither moves, and
// the findings stay so the store still says what is wrong.
func TestFixRefusesToClobber(t *testing.T) {
	s := newTestStore(t)
	tk := mustCreate(t, s, "The real one")

	// A second file carrying the same ID, under a name that is not the ID.
	data, err := os.ReadFile(tk.Path)
	if err != nil {
		t.Fatal(err)
	}
	impostor := filepath.Join(s.TicketsDir(), "copy-of-the-real-one.md")
	if err := os.WriteFile(impostor, data, 0o644); err != nil {
		t.Fatal(err)
	}

	res := mustFix(t, s, FixOptions{})
	if len(res.Repairs) != 0 {
		t.Errorf("nothing should move onto an occupied path: %+v", res.Repairs)
	}
	if _, err := os.Stat(impostor); err != nil {
		t.Errorf("the second file was moved anyway: %v", err)
	}
	if _, err := os.Stat(tk.Path); err != nil {
		t.Errorf("the first file was moved anyway: %v", err)
	}
	if !has(codesOf(res.Report.Errors), CodeDuplicateID) {
		t.Errorf("errors = %v, want duplicate_id left standing", codesOf(res.Report.Errors))
	}
}

// TestFixLeavesEveryOtherFindingAlone holds the scope. Fix repairs the two
// findings that have one correct repair and reports the rest untouched, because
// a tool that guessed at the others would be wrong about half of them.
func TestFixLeavesEveryOtherFindingAlone(t *testing.T) {
	s := newTestStore(t)
	tk := mustCreate(t, s, "Waiting on nothing that exists")

	// Written by hand, because AddDependency refuses an ID that resolves to
	// nothing. The condition check reports is a file that got into this state
	// some other way, which is exactly what a repair pass has to meet.
	data, err := os.ReadFile(tk.Path)
	if err != nil {
		t.Fatal(err)
	}
	dangling := strings.Replace(string(data), "dependencies: []",
		"dependencies:\n  - TKT-01K400ZZZZZZZZZZZZZZZZZZZZ", 1)
	if dangling == string(data) {
		t.Fatal("the fixture did not carry an empty dependencies list to edit")
	}
	if err := os.WriteFile(tk.Path, []byte(dangling), 0o644); err != nil {
		t.Fatal(err)
	}

	res := mustFix(t, s, FixOptions{})
	if len(res.Repairs) != 0 {
		t.Errorf("a missing dependency is not repairable here: %+v", res.Repairs)
	}
	if !has(codesOf(res.Report.Errors), CodeDependencyMissing) {
		t.Errorf("errors = %v, want dependency_missing reported", codesOf(res.Report.Errors))
	}
}

// TestFixOnACleanStoreDoesNothing keeps a no-op quiet. A repair pass over a
// store with nothing wrong should report no moves rather than rewriting files
// to no effect.
func TestFixOnACleanStoreDoesNothing(t *testing.T) {
	s := newTestStore(t)
	mustCreate(t, s, "Perfectly fine")

	res := mustFix(t, s, FixOptions{})
	if len(res.Repairs) != 0 {
		t.Errorf("got %+v, want no repairs", res.Repairs)
	}
	if !res.Report.OK() {
		t.Errorf("a clean store should stay clean: %v", codesOf(res.Report.Errors))
	}
}
