package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// import exists for the one case `git am` cannot serve: a ticket whose series
// the receiving store does not declare. These tests are built around that, and
// the first thing several of them assert is that the ordinary case is refused
// and pointed back at git.

// newForeignExport is a store declaring a series the receiver will not know,
// exported to a directory.
func newForeignExport(t *testing.T, titles ...string) (dir string, ids []string) {
	t.Helper()
	src := newGitStore(t)
	if got := runCLI(t, src, nil, "series", "add", "LIVE", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("series add: %s%s", got.stdout, got.stderr)
	}
	for _, title := range titles {
		got := runCLI(t, src, nil, "--json", "create", "--title", title,
			"--series", "LIVE", "--actor", "human:sothr")
		if got.code != exitOK {
			t.Fatalf("create %q: %s%s", title, got.stdout, got.stderr)
		}
		var env struct {
			Ticket struct {
				ID string `json:"id"`
			} `json:"ticket"`
		}
		if err := json.Unmarshal([]byte(got.stdout), &env); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, env.Ticket.ID)
	}
	exportGit(t, src, "add", "-A")
	exportGit(t, src, "commit", "-qm", "store")

	dir = filepath.Join(t.TempDir(), "out")
	args := append([]string{"export"}, ids...)
	if got := runCLI(t, src, nil, append(args, "--out", dir)...); got.code != exitOK {
		t.Fatalf("export: %s%s", got.stdout, got.stderr)
	}
	return dir, ids
}

// TestImportPreviewWritesNothing is the acceptance criterion that nothing
// auto-applies. The preview is the default because an export is somebody else's
// content, and reading it is not the same as agreeing to it.
func TestImportPreviewWritesNothing(t *testing.T) {
	dir, ids := newForeignExport(t, "A foreign ticket")
	dest := newGitStore(t)

	got := runCLI(t, dest, nil, "import", dir)
	if got.code != exitOK {
		t.Fatalf("import: %s%s", got.stdout, got.stderr)
	}
	if !strings.Contains(got.stdout, "remint") {
		t.Errorf("preview did not say it would remint:\n%s", got.stdout)
	}
	if !strings.Contains(got.stdout, ids[0]) {
		t.Errorf("preview did not name the incoming ticket:\n%s", got.stdout)
	}
	if list := runCLI(t, dest, nil, "list", "--all"); strings.Contains(list.stdout, "A foreign ticket") {
		t.Error("the preview wrote a ticket")
	}
}

// TestImportRemintsAndKeepsTheEdges is the heart of it: a foreign series is
// filed again under one this store declares, and the dependency and parent
// among the imported tickets are rewritten to the IDs just minted. Getting this
// wrong points the graph at IDs that do not exist here.
func TestImportRemintsAndKeepsTheEdges(t *testing.T) {
	src := newGitStore(t)
	if got := runCLI(t, src, nil, "series", "add", "LIVE", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("series add: %s%s", got.stdout, got.stderr)
	}
	mk := func(title, typ string) string {
		got := runCLI(t, src, nil, "--json", "create", "--title", title, "--type", typ,
			"--series", "LIVE", "--actor", "human:sothr")
		if got.code != exitOK {
			t.Fatalf("create: %s%s", got.stdout, got.stderr)
		}
		var env struct {
			Ticket struct {
				ID string `json:"id"`
			} `json:"ticket"`
		}
		if err := json.Unmarshal([]byte(got.stdout), &env); err != nil {
			t.Fatal(err)
		}
		return env.Ticket.ID
	}
	parent := mk("Parent epic", "epic")
	child := mk("Child ticket", "task")
	if got := runCLI(t, src, nil, "update", child, "--parent", parent, "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("parent: %s%s", got.stdout, got.stderr)
	}
	if got := runCLI(t, src, nil, "link", child, "--depends-on", parent, "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("link: %s%s", got.stdout, got.stderr)
	}
	exportGit(t, src, "add", "-A")
	exportGit(t, src, "commit", "-qm", "store")
	dir := filepath.Join(t.TempDir(), "out")
	if got := runCLI(t, src, nil, "export", child, parent, "--out", dir); got.code != exitOK {
		t.Fatalf("export: %s%s", got.stdout, got.stderr)
	}

	dest := newGitStore(t)
	if got := runCLI(t, dest, nil, "import", dir, "--apply",
		"--from-store", "terva-sh/elsewhere", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("import --apply: %s%s", got.stdout, got.stderr)
	}

	rows := crossRows(t, runCLI(t, dest, nil, "--json", "list", "--all"))
	if len(rows) != 2 {
		t.Fatalf("filed %d tickets, want 2", len(rows))
	}
	newParent := crossRow(t, rows, "Parent epic")
	newChild := crossRow(t, rows, "Child ticket")

	// Nothing may keep the foreign prefix, or check fails on unknown_series.
	for _, row := range rows {
		if id, _ := row["id"].(string); strings.HasPrefix(id, "LIVE-") {
			t.Errorf("ticket kept its foreign series: %s", id)
		}
	}
	// The epic stays an epic: reminting changes identity, not content.
	if newParent["type"] != "epic" {
		t.Errorf("parent type = %v, want epic", newParent["type"])
	}

	full := decode(t, runCLI(t, dest, nil, "--json", "show", newChild["id"].(string)).stdout)
	tk, _ := full["ticket"].(map[string]any)
	wantParent := newParent["id"]
	if tk["parent"] != wantParent {
		t.Errorf("parent = %v, want the reminted %v", tk["parent"], wantParent)
	}
	deps, _ := tk["dependencies"].([]any)
	if len(deps) != 1 || deps[0] != wantParent {
		t.Errorf("dependencies = %v, want [%v]", deps, wantParent)
	}
	// Provenance is a reference and never origin, because check verifies origin
	// against this store and a foreign ID cannot resolve.
	refs, _ := tk["references"].([]any)
	var seen []string
	for _, r := range refs {
		m, _ := r.(map[string]any)
		s, _ := m["ref"].(string)
		seen = append(seen, s)
	}
	joined := strings.Join(seen, " ")
	if !strings.Contains(joined, "origin-ticket:"+child) {
		t.Errorf("references = %v, want the source ID recorded", seen)
	}
	if !strings.Contains(joined, "origin-store:terva-sh/elsewhere") {
		t.Errorf("references = %v, want the source store recorded", seen)
	}
	if tk["origin"] != nil {
		t.Errorf("origin = %v, want nil; a foreign ID in origin fails check", tk["origin"])
	}

	// The whole point: the receiving store is valid, which it was not before.
	if res := runCLI(t, dest, nil, "check"); res.code != exitOK {
		t.Errorf("check after import: %s%s", res.stdout, res.stderr)
	}
}

// TestImportRefusesASeriesThisStoreKnows keeps import from quietly filing a
// second copy of a ticket that `git am` would have placed correctly, under its
// own name.
func TestImportRefusesASeriesThisStoreKnows(t *testing.T) {
	src, id := newExportSource(t, "An ordinary TKT ticket")
	dir := filepath.Join(t.TempDir(), "out")
	if got := runCLI(t, src, nil, "export", id, "--out", dir); got.code != exitOK {
		t.Fatalf("export: %s%s", got.stdout, got.stderr)
	}
	dest := newGitStore(t)
	got := runCLI(t, dest, nil, "import", dir, "--apply", "--actor", "human:sothr")
	if got.code == exitOK {
		t.Fatal("import filed a duplicate of a ticket git am would have placed")
	}
	if !strings.Contains(got.stderr, "git am") {
		t.Errorf("stderr = %q, want it to name the right tool", got.stderr)
	}
	if list := runCLI(t, dest, nil, "list", "--all"); strings.Contains(list.stdout, "An ordinary TKT ticket") {
		t.Error("the refusal still wrote a ticket")
	}
}

// TestImportDetectsATamperedPatch checks the blob name on the way in. The hash
// is already written on the way out, so verifying it costs nothing and turns a
// truncated or edited patch into an error here rather than a puzzling ticket
// later.
func TestImportDetectsATamperedPatch(t *testing.T) {
	dir, _ := newForeignExport(t, "A foreign ticket")
	patch := filepath.Join(dir, exportTicketPatch)
	data, err := os.ReadFile(patch)
	if err != nil {
		t.Fatal(err)
	}
	altered := strings.Replace(string(data), "+title: A foreign ticket", "+title: Something else", 1)
	if altered == string(data) {
		t.Fatal("the fixture did not contain the line to alter")
	}
	if err := os.WriteFile(patch, []byte(altered), 0o644); err != nil {
		t.Fatal(err)
	}

	dest := newGitStore(t)
	got := runCLI(t, dest, nil, "import", dir)
	if got.code == exitOK {
		t.Fatal("import accepted a patch whose content no longer matches its blob name")
	}
	if !strings.Contains(got.stderr, "blob name") {
		t.Errorf("stderr = %q, want it to say the content does not match", got.stderr)
	}
}

// TestImportReportsDroppedEdges covers the edge that cannot survive: a
// dependency on a ticket the export does not carry. It has to be dropped, since
// a dependency on an ID this store has never seen is dependency_missing, so the
// duty is to say so loudly.
func TestImportReportsDroppedEdges(t *testing.T) {
	src := newGitStore(t)
	if got := runCLI(t, src, nil, "series", "add", "LIVE", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("series add: %s%s", got.stdout, got.stderr)
	}
	mk := func(title string) string {
		got := runCLI(t, src, nil, "--json", "create", "--title", title,
			"--series", "LIVE", "--actor", "human:sothr")
		var env struct {
			Ticket struct {
				ID string `json:"id"`
			} `json:"ticket"`
		}
		if err := json.Unmarshal([]byte(got.stdout), &env); err != nil {
			t.Fatal(err)
		}
		return env.Ticket.ID
	}
	stays := mk("Ticket that travels")
	behind := mk("Ticket left behind")
	if got := runCLI(t, src, nil, "link", stays, "--depends-on", behind, "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("link: %s%s", got.stdout, got.stderr)
	}
	exportGit(t, src, "add", "-A")
	exportGit(t, src, "commit", "-qm", "store")
	dir := filepath.Join(t.TempDir(), "out")
	// Only one of the two travels, so its dependency cannot be resolved.
	if got := runCLI(t, src, nil, "export", stays, "--out", dir); got.code != exitOK {
		t.Fatalf("export: %s%s", got.stdout, got.stderr)
	}

	dest := newGitStore(t)
	preview := runCLI(t, dest, nil, "import", dir)
	if !strings.Contains(preview.stdout, "drops dependency "+behind) {
		t.Errorf("the preview did not warn about the dropped edge:\n%s", preview.stdout)
	}
	got := runCLI(t, dest, nil, "import", dir, "--apply", "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("import --apply: %s%s", got.stdout, got.stderr)
	}
	if !strings.Contains(got.stderr, "dropped dependency") {
		t.Errorf("stderr = %q, want the dropped edge named", got.stderr)
	}
	// Dropping it is what leaves a store that passes.
	if res := runCLI(t, dest, nil, "check"); res.code != exitOK {
		t.Errorf("check after import: %s%s", res.stdout, res.stderr)
	}
}

// TestImportNeedsAnExportDirectory keeps the error a sentence that names the
// repair, rather than a parse failure from somewhere deeper.
func TestImportNeedsAnExportDirectory(t *testing.T) {
	dest := newGitStore(t)
	got := runCLI(t, dest, nil, "import", t.TempDir())
	if got.code == exitOK {
		t.Fatal("import accepted a directory that holds no export")
	}
	if !strings.Contains(got.stderr, exportTicketPatch) {
		t.Errorf("stderr = %q, want it to name what was missing", got.stderr)
	}
}

// TestImportPointsAtTheCodePatchesItWillNotApply covers the boundary with the
// other half of a handoff. An export can carry code beside its tickets, import
// does not apply code, and saying nothing would leave the receiver believing the
// whole thing had arrived.
func TestImportPointsAtTheCodePatchesItWillNotApply(t *testing.T) {
	dir, _ := newForeignExport(t, "A foreign ticket")
	// A code patch, numbered the way format-patch would place it beside an export.
	extra := filepath.Join(dir, "0002-Some-code-change.patch")
	if err := os.WriteFile(extra, []byte("From 0000 Mon Sep 17 00:00:00 2001\nSubject: [PATCH] code\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	dest := newGitStore(t)
	got := runCLI(t, dest, nil, "import", dir)
	if got.code != exitOK {
		t.Fatalf("import: %s%s", got.stdout, got.stderr)
	}
	if !strings.Contains(got.stderr, "does not apply") {
		t.Errorf("stderr = %q, want it to say the code is not import's job", got.stderr)
	}
	if !strings.Contains(got.stderr, "git am") {
		t.Errorf("stderr = %q, want it to name the tool that does apply it", got.stderr)
	}
}
