package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// The before-image of the interchange move, per TKT-01M298KEG (Move the export
// and import interchange into the ticket library).
//
// An export is deterministic once three things are held still: the clock, the
// sender's git identity, and the tickets themselves. runCLI pins the clock to
// referenceInstant, newFixtureStore pins the identity in the repository it
// builds, and the fixture corpus supplies tickets whose IDs and timestamps were
// written by a person rather than minted at test time. What is left is the wire
// format, which is the thing the move must not change.
//
// There is deliberately no -update flag. A golden test usually has one, and here
// it would defeat the point: these bytes are evidence from before the move, and
// a flag that rewrites them after the fact turns a failed comparison into a
// one-key formality. If the move changes the artifact, that is either a defect
// to fix or a decision to record in the ticket, and both need a person.

// beforeImage is where the pinned artifact lives, relative to this package.
const beforeImage = "testdata/export-before-image"

// The two tickets exported into the before-image. They come from the clean
// fixture store and are chosen for what they carry: one has a Notes section and
// one has Acceptance criteria, so the pinned diff covers both a work record and
// a checklist rather than frontmatter alone.
const (
	beforeImageNotes     = "TKT-01K3ZZ67Q0PT427VFD1F4WFWSH"
	beforeImageChecklist = "TKT-01K3ZZ82A0YPGSE71EY0N5NCH6"
)

// copyTree copies src over dst, making directories as it goes.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("read %s: %v", src, err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		from, to := filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())
		if e.IsDir() {
			copyTree(t, from, to)
			continue
		}
		data, err := os.ReadFile(from)
		if err != nil {
			t.Fatalf("read %s: %v", from, err)
		}
		if err := os.WriteFile(to, data, 0o644); err != nil {
			t.Fatalf("write %s: %v", to, err)
		}
	}
}

// newFixtureStore is a git repository whose store is a fixture from the corpus,
// with the identity export reads pinned.
//
// The identity matters more than it looks. exportIdentity asks git for
// user.name and user.email, so without a repository-local setting the From:
// header carries whoever ran the suite, and the artifact differs on every
// machine. newGitStore does not need this, because it supplies an identity to
// commit through the environment, which export never reads.
func newFixtureStore(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	exportGit(t, dir, "init", "-q", "-b", "main")
	copyTree(t, filepath.Join("..", "testdata", "stores", name, "store"), filepath.Join(dir, ".tickets"))
	exportGit(t, dir, "config", "user.name", "Pinned Sender")
	exportGit(t, dir, "config", "user.email", "pinned@example.com")
	exportGit(t, dir, "add", "-A")
	exportGit(t, dir, "commit", "-qm", "store")
	return dir
}

// TestExportMatchesThePinnedBeforeImage is the sixth acceptance criterion of
// TKT-01M298KEG: the artifact is byte-identical across the move of the
// interchange into the library.
//
// It asserts on the bytes, which the rest of the export suite deliberately does
// not: those tests apply the patch with real git, because a patch that looks
// right and does not apply is the failure worth catching. This one is the other
// half. git will happily apply an artifact whose cover letter lost a line, and
// a refactor that quietly reformats the wire format is exactly what a
// round-trip test cannot see.
func TestExportMatchesThePinnedBeforeImage(t *testing.T) {
	dir := newFixtureStore(t, "clean")
	out := filepath.Join(t.TempDir(), "export")

	got := runCLI(t, dir, nil, "export", beforeImageNotes, beforeImageChecklist, "--out", out)
	if got.code != exitOK {
		t.Fatalf("export: %s%s", got.stdout, got.stderr)
	}

	for _, name := range []string{exportCoverName, exportTicketPatch} {
		want, err := os.ReadFile(filepath.Join(beforeImage, name))
		if err != nil {
			t.Fatalf("read the pinned %s: %v", name, err)
		}
		have, err := os.ReadFile(filepath.Join(out, name))
		if err != nil {
			t.Fatalf("read the produced %s: %v", name, err)
		}
		if string(have) != string(want) {
			t.Errorf("%s moved.\n--- pinned ---\n%s\n--- produced ---\n%s", name, want, have)
		}
	}
}
