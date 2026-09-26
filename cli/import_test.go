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
	if !strings.Contains(got.stdout, "--adopt") {
		t.Errorf("preview did not name the flag that writes:\n%s", got.stdout)
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
	if got := runCLI(t, dest, nil, "import", dir, "--adopt",
		"--from-store", "terva-sh/elsewhere", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("import --adopt: %s%s", got.stdout, got.stderr)
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

// TestImportAdoptsDespiteASharedSeries is the correction this file exists to
// carry. An earlier version refused when the receiving store already declared
// the incoming series, on the theory that a shared series meant a shared
// project. It does not: TKT is the default every store has, so two strangers
// share it by default and the refusal fired hardest on the case import was
// built for. The tool no longer guesses. It advises in the preview and adopts
// when told to.
func TestImportAdoptsDespiteASharedSeries(t *testing.T) {
	src, id := newExportSource(t, "An ordinary TKT ticket")
	dir := filepath.Join(t.TempDir(), "out")
	if got := runCLI(t, src, nil, "export", id, "--out", dir); got.code != exitOK {
		t.Fatalf("export: %s%s", got.stdout, got.stderr)
	}
	dest := newGitStore(t)

	// The preview offers git am as the better route without insisting on it.
	preview := runCLI(t, dest, nil, "import", dir)
	if !strings.Contains(preview.stdout, "git am") {
		t.Errorf("preview did not offer git am for a shared series:\n%s", preview.stdout)
	}

	got := runCLI(t, dest, nil, "import", dir, "--adopt", "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("import --adopt refused a shared series: %s%s", got.stdout, got.stderr)
	}
	rows := crossRows(t, runCLI(t, dest, nil, "--json", "list", "--all"))
	if len(rows) != 1 {
		t.Fatalf("filed %d tickets, want 1", len(rows))
	}
	if newID, _ := rows[0]["id"].(string); newID == id {
		t.Error("adopt kept the original ID; it should mint a fresh one")
	}
	if res := runCLI(t, dest, nil, "check", "--strict"); res.code != exitOK {
		t.Errorf("check after adopt: %s%s", res.stdout, res.stderr)
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
	if !strings.Contains(got.stderr, "digest does not match") {
		t.Errorf("stderr = %q, want the sidecar to reject the changed patch first", got.stderr)
	}
	if err := os.Remove(filepath.Join(dir, exportReferenceLookup)); err != nil {
		t.Fatal(err)
	}
	legacy := runCLI(t, dest, nil, "import", dir)
	if legacy.code == exitOK || !strings.Contains(legacy.stderr, "blob name") {
		t.Errorf("old export must still catch the altered blob: %s%s", legacy.stdout, legacy.stderr)
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
	if !strings.Contains(preview.stdout, "dependency "+behind+": not carried") {
		t.Errorf("the preview did not warn about the dropped edge:\n%s", preview.stdout)
	}
	got := runCLI(t, dest, nil, "import", dir, "--adopt", "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("import --adopt: %s%s", got.stdout, got.stderr)
	}
	if !strings.Contains(got.stderr, "dependency "+behind+" not carried") {
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

// TestImportDropsVocabularyThisStoreDoesNotShare is the fix for what the
// bootstrap delivery found. A sender's labels and reference paths are theirs,
// not the receiver's: an undeclared label is label_unknown and a path that does
// not resolve is reference_path_unresolved, both warnings, and a receiver whose
// CI runs check --strict fails on warnings.
//
// Carrying somebody's vocabulary into a store that never agreed to it is not a
// kindness. Dropping it is the reconciliation git am structurally cannot do.
func TestImportDropsVocabularyThisStoreDoesNotShare(t *testing.T) {
	src := newGitStore(t)
	if got := runCLI(t, src, nil, "series", "add", "LIVE", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("series add: %s%s", got.stdout, got.stderr)
	}
	got := runCLI(t, src, nil, "--json", "create", "--title", "Carries a foreign vocabulary",
		"--series", "LIVE", "--label", "sender-only", "--actor", "human:sothr")
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
	id := env.Ticket.ID
	// A reference to a file that exists here and will not exist there.
	if err := os.WriteFile(filepath.Join(src, "SENDER.md"), []byte("only here\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := runCLI(t, src, nil, "link", id, "--ref", "doc:sender", "--path", "SENDER.md",
		"--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("link: %s%s", got.stdout, got.stderr)
	}
	exportGit(t, src, "add", "-A")
	exportGit(t, src, "commit", "-qm", "store")
	dir := filepath.Join(t.TempDir(), "out")
	if got := runCLI(t, src, nil, "export", id, "--out", dir); got.code != exitOK {
		t.Fatalf("export: %s%s", got.stdout, got.stderr)
	}

	// A receiving store with its own allowlist, which does not include the
	// sender's label, and without the file the reference points at.
	dest := newGitStore(t)
	cfgPath := filepath.Join(dest, ".tickets", "config.yml")
	cfg, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, []byte(strings.Replace(string(cfg),
		"labels: []", "labels:\n  - receiver-only", 1)), 0o644); err != nil {
		t.Fatal(err)
	}

	preview := runCLI(t, dest, nil, "import", dir)
	if !strings.Contains(preview.stdout, `label "sender-only": not carried`) {
		t.Errorf("preview did not warn about the foreign label:\n%s", preview.stdout)
	}
	if !strings.Contains(preview.stdout, "reference doc:sender: the path is not carried") {
		t.Errorf("preview did not warn about the unresolvable path:\n%s", preview.stdout)
	}

	adopt := runCLI(t, dest, nil, "import", dir, "--adopt", "--actor", "human:sothr")
	if adopt.code != exitOK {
		t.Fatalf("import --adopt: %s%s", adopt.stdout, adopt.stderr)
	}
	if !strings.Contains(adopt.stderr, `label "sender-only": not carried`) {
		t.Errorf("stderr did not name the dropped label:\n%s", adopt.stderr)
	}

	// The whole point: the receiving store passes --strict, which it would not
	// have done had the vocabulary come across.
	if res := runCLI(t, dest, nil, "check", "--strict"); res.code != exitOK {
		t.Errorf("check --strict after adopt: %s%s", res.stdout, res.stderr)
	}
	// The reference survives without its path, because what the sender pointed
	// at is still worth knowing.
	rows := crossRows(t, runCLI(t, dest, nil, "--json", "list", "--all"))
	full := decode(t, runCLI(t, dest, nil, "--json", "show", rows[0]["id"].(string)).stdout)
	tk, _ := full["ticket"].(map[string]any)
	refs, _ := tk["references"].([]any)
	var kept bool
	for _, r := range refs {
		m, _ := r.(map[string]any)
		if m["ref"] == "doc:sender" {
			kept = true
			if m["path"] != nil {
				t.Errorf("the unresolvable path survived: %v", m["path"])
			}
		}
	}
	if !kept {
		t.Error("the reference itself was dropped; only its path should have been")
	}
}

// TestAdoptCarriesTheStatementOfTheWork is the regression for the defect the
// bootstrap delivery shipped with: adopt filed a ticket from Title, Type,
// Priority, Labels, Assignees, Description and ImplementationPlan, and every
// other thing a ticket says went on the floor without a word.
//
// Measured against the five tickets of that delivery, whose own APPLYING.md told
// the receiver to adopt them: 24 checkbox lines in, 0 out, and every Notes and
// Summary section with them. `git am` was lossless and --adopt was not, which
// inverts the argument for import existing beside it.
//
// The rule this asserts is the one reconcile is built on. The statement of the
// work travels. What the receiver never agreed to does not, and is named.
func TestAdoptCarriesTheStatementOfTheWork(t *testing.T) {
	src := newGitStore(t)
	if got := runCLI(t, src, nil, "series", "add", "LIVE", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("series add: %s%s", got.stdout, got.stderr)
	}
	created := decode(t, runCLI(t, src, nil, "--json", "create",
		"--title", "A defect worth carrying",
		"--description", "The repro is here.",
		"--series", "LIVE", "--actor", "human:sothr").stdout)
	id, _ := created["ticket"].(map[string]any)["id"].(string)
	if id == "" {
		t.Fatal("create returned no id")
	}
	run := func(args ...string) {
		t.Helper()
		if got := runCLI(t, src, nil, append(args, "--actor", "human:sothr")...); got.code != exitOK {
			t.Fatalf("%v: %s%s", args, got.stdout, got.stderr)
		}
	}
	run("ac", id, "--add", "the first criterion")
	run("ac", id, "--add", "the second criterion")
	// Ticked at the origin, which is the sender's evidence about the sender's
	// work and says nothing about this store.
	run("ac", id, "--check", "1")
	run("dod", id, "--add", "the suite is green")
	run("note", id, "the root cause was a symlinked path")
	run("summary", id, "where this one landed in the end")
	run("update", id, "--milestone", "sender-roadmap", "--due-on", "2026-12-01")

	exportGit(t, src, "add", "-A")
	exportGit(t, src, "commit", "-qm", "store")
	dir := filepath.Join(t.TempDir(), "out")
	if got := runCLI(t, src, nil, "export", id, "--out", dir); got.code != exitOK {
		t.Fatalf("export: %s%s", got.stdout, got.stderr)
	}

	// The receiving store declares a milestone list of its own. Without one the
	// allowlist is empty, plan 4.1 says an empty allowlist permits everything,
	// and the sender's milestone would rightly travel. That is the same reason
	// the label case seeds an allowlist before asserting a label is refused.
	dest := newGitStore(t)
	declareMilestone(t, dest, "receiver-roadmap")
	adopt := runCLI(t, dest, nil, "import", dir, "--adopt",
		"--from-store", "flywheel/ledger", "--actor", "human:sothr")
	if adopt.code != exitOK {
		t.Fatalf("import --adopt: %s%s", adopt.stdout, adopt.stderr)
	}

	rows := crossRows(t, runCLI(t, dest, nil, "--json", "list", "--all"))
	if len(rows) != 1 {
		t.Fatalf("want one adopted ticket, got %d", len(rows))
	}
	full := decode(t, runCLI(t, dest, nil, "--json", "show", rows[0]["id"].(string)).stdout)
	tk, _ := full["ticket"].(map[string]any)

	// The checklists arrive whole, and every box arrives empty.
	lists, _ := tk["checklists"].(map[string]any)
	for _, tc := range []struct {
		field string
		want  []string
	}{
		{"acceptanceCriteria", []string{"the first criterion", "the second criterion"}},
		{"definitionOfDone", []string{"the suite is green"}},
	} {
		items, _ := lists[tc.field].([]any)
		if len(items) != len(tc.want) {
			t.Errorf("%s: %d items, want %d", tc.field, len(items), len(tc.want))
			continue
		}
		for i, want := range tc.want {
			item, _ := items[i].(map[string]any)
			if item["text"] != want {
				t.Errorf("%s[%d] = %v, want %q", tc.field, i, item["text"], want)
			}
			if item["checked"] != false {
				t.Errorf("%s[%d] arrived ticked; the sender's evidence is not this store's", tc.field, i)
			}
		}
	}

	// The sending store's work record travels whole, in one note that says where
	// it came from rather than restamping it with whoever ran import.
	body, _ := tk["body"].(map[string]any)
	notes, _ := body["notes"].(string)
	for _, want := range []string{
		"the root cause was a symlinked path",
		"where this one landed in the end",
		"flywheel/ledger",
		id,
	} {
		if !strings.Contains(notes, want) {
			t.Errorf("the carried work record does not mention %q:\n%s", want, notes)
		}
	}

	// What the receiver never agreed to does not travel, and is named on the way
	// past rather than dropped in silence.
	if tk["milestone"] != nil {
		t.Errorf("milestone = %v, want none: this store does not declare it", tk["milestone"])
	}
	if tk["dueOn"] != nil {
		t.Errorf("dueOn = %v, want none: a deadline this store never agreed to", tk["dueOn"])
	}
	for _, want := range []string{
		`milestone "sender-roadmap": not carried`,
		"due date 2026-12-01: not carried",
		"acceptance criteria: 2 carried, every box unchecked",
		"summary, notes and comments: carried as one note",
	} {
		if !strings.Contains(adopt.stderr, want) {
			t.Errorf("adopt did not report %q:\n%s", want, adopt.stderr)
		}
	}

	// The preview has to promise exactly what the write then does, which is why
	// both read one reconciliation rather than each working it out.
	second := newGitStore(t)
	declareMilestone(t, second, "receiver-roadmap")
	preview := runCLI(t, second, nil, "import", dir)
	for _, want := range []string{
		`milestone "sender-roadmap": not carried`,
		"due date 2026-12-01: not carried",
		"acceptance criteria: 2 carried, every box unchecked",
	} {
		if !strings.Contains(preview.stdout, want) {
			t.Errorf("the preview did not promise %q:\n%s", want, preview.stdout)
		}
	}

	if res := runCLI(t, dest, nil, "check", "--strict"); res.code != exitOK {
		t.Errorf("check --strict after adopt: %s%s", res.stdout, res.stderr)
	}
}

// declareMilestone gives a store a milestone allowlist holding one name, so that
// every other milestone is outside it. An empty allowlist permits everything per
// plan 4.1, so a test about refusing a foreign milestone has to seed one or it
// asserts nothing.
func declareMilestone(t *testing.T, store, name string) {
	t.Helper()
	path := filepath.Join(store, ".tickets", "config.yml")
	cfg, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	out := strings.Replace(string(cfg), "milestones: []", "milestones:\n  - "+name, 1)
	if out == string(cfg) {
		t.Fatalf("config.yml has no empty milestones list to seed:\n%s", cfg)
	}
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		t.Fatal(err)
	}
}

// newOwnedExport is one owner's store, holding a ticket that was worked and
// finished there, exported to a directory.
//
// It carries a parent that stays behind, a ticked and an unticked criterion, a
// ticked definition-of-done item, and a milestone the receiver will not declare,
// so one export exercises both halves of the rule: the evidence that travels
// under --same-owner and the vocabulary that still does not.
func newOwnedExport(t *testing.T) (dir, parentID, childID string) {
	t.Helper()
	src := newGitStore(t)
	mint := func(args ...string) string {
		t.Helper()
		got := runCLI(t, src, nil, append([]string{"--json", "create"}, append(args, "--actor", "human:sothr")...)...)
		if got.code != exitOK {
			t.Fatalf("create %v: %s%s", args, got.stdout, got.stderr)
		}
		id, _ := decode(t, got.stdout)["ticket"].(map[string]any)["id"].(string)
		if id == "" {
			t.Fatalf("create %v returned no id", args)
		}
		return id
	}
	parentID = mint("--title", "The epic that stays home", "--type", "epic")
	childID = mint("--title", "The ticket that moves", "--parent", parentID)

	run := func(args ...string) {
		t.Helper()
		if got := runCLI(t, src, nil, append(args, "--actor", "human:sothr")...); got.code != exitOK {
			t.Fatalf("%v: %s%s", args, got.stdout, got.stderr)
		}
	}
	run("ac", childID, "--add", "the criterion that was met")
	run("ac", childID, "--add", "the criterion that was not")
	run("ac", childID, "--check", "1")
	run("dod", childID, "--add", "the suite is green")
	run("dod", childID, "--check", "1")
	run("update", childID, "--milestone", "sender-roadmap")
	run("status", childID, "ready")
	run("status", childID, "in-progress")
	run("status", childID, "done")

	exportGit(t, src, "add", "-A")
	exportGit(t, src, "commit", "-qm", "store")
	dir = filepath.Join(t.TempDir(), "out")
	if got := runCLI(t, src, nil, "export", childID, "--out", dir); got.code != exitOK {
		t.Fatalf("export: %s%s", got.stdout, got.stderr)
	}
	return dir, parentID, childID
}

// box is one checklist item as these tests compare them: the text and whether
// it is ticked, which is the whole of what --same-owner is about.
type box struct {
	Text    string
	Checked bool
}

// boxes reads one checklist section back out of the JSON envelope. The name
// avoids cli.checklist, which renders the envelope rather than reading it.
func boxes(t *testing.T, tk map[string]any, field string) []box {
	t.Helper()
	lists, _ := tk["checklists"].(map[string]any)
	items, _ := lists[field].([]any)
	var out []box
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		text, _ := item["text"].(string)
		checked, _ := item["checked"].(bool)
		out = append(out, box{text, checked})
	}
	return out
}

// TestSameOwnerCarriesTheEvidence is the second case of the interchange rule.
//
// Where TestAdoptCarriesTheStatementOfTheWork asserts that a stranger's evidence
// does not travel, this asserts that one owner's own evidence does, and that the
// vocabulary half of the rule is untouched by it. Both must hold at once, which
// is why the milestone assertion sits in this test rather than in its own: the
// failure worth catching is --same-owner being read as "carry everything".
func TestSameOwnerCarriesTheEvidence(t *testing.T) {
	dir, parentID, childID := newOwnedExport(t)

	dest := newGitStore(t)
	declareMilestone(t, dest, "receiver-roadmap")
	adopt := runCLI(t, dest, nil, "import", dir, "--adopt", "--same-owner",
		"--from-store", "flywheel/ledger", "--actor", "human:sothr")
	if adopt.code != exitOK {
		t.Fatalf("import --adopt --same-owner: %s%s", adopt.stdout, adopt.stderr)
	}

	rows := crossRows(t, runCLI(t, dest, nil, "--json", "list", "--all"))
	if len(rows) != 1 {
		t.Fatalf("want one adopted ticket, got %d", len(rows))
	}
	newID, _ := rows[0]["id"].(string)
	tk, _ := decode(t, runCLI(t, dest, nil, "--json", "show", newID).stdout)["ticket"].(map[string]any)

	// The ticks travel exactly as the owner left them. Not all of them and not
	// none of them: a flag that ticked every box would pass an assertion that
	// only looked at the first.
	for _, tc := range []struct {
		field string
		want  []box
	}{
		{"acceptanceCriteria", []box{
			{"the criterion that was met", true},
			{"the criterion that was not", false},
		}},
		{"definitionOfDone", []box{{"the suite is green", true}}},
	} {
		got := boxes(t, tk, tc.field)
		if len(got) != len(tc.want) {
			t.Errorf("%s: %d items, want %d", tc.field, len(got), len(tc.want))
			continue
		}
		for i := range tc.want {
			if got[i] != tc.want[i] {
				t.Errorf("%s[%d] = %+v, want %+v", tc.field, i, got[i], tc.want[i])
			}
		}
	}

	// Status travels, because this owner finished this work.
	if tk["status"] != "done" {
		t.Errorf("status = %v, want done: the same owner finished it", tk["status"])
	}

	// The parent stayed home, so the link is kept where the edge cannot go.
	if tk["parent"] != nil {
		t.Errorf("parent = %v, want none: the edge would name a ticket this store does not have", tk["parent"])
	}
	var refs []string
	for _, raw := range tk["references"].([]any) {
		ref, _ := raw.(map[string]any)["ref"].(string)
		refs = append(refs, ref)
	}
	for _, want := range []string{
		"origin-ticket:" + childID,
		"origin-store:flywheel/ledger",
		"origin-parent:" + parentID,
	} {
		if !slicesContains(refs, want) {
			t.Errorf("references %v do not carry %q", refs, want)
		}
	}

	// The vocabulary half of the rule does not move. One owner keeping two
	// stores is not one owner keeping two allowlists.
	if tk["milestone"] != nil {
		t.Errorf("milestone = %v, want none: --same-owner says nothing about vocabulary", tk["milestone"])
	}

	// Every carry is named. Evidence that arrives unannounced is as hard to
	// trust later as evidence that vanishes.
	for _, want := range []string{
		"acceptance criteria: 1 ticked at the origin, carried ticked",
		"definition of done: 1 ticked at the origin, carried ticked",
		"status done: carried",
		"parent " + parentID + ": kept as an origin-parent reference",
		`milestone "sender-roadmap": not carried`,
	} {
		if !strings.Contains(adopt.stderr, want) {
			t.Errorf("adopt did not report %q:\n%s", want, adopt.stderr)
		}
	}

	// A parent kept as provenance is not also a dropped edge. The first real run
	// of this flag reported it as both, one line after the other.
	if strings.Contains(adopt.stderr, "parent "+parentID+" not carried") {
		t.Errorf("the parent is reported as kept and as dropped at once:\n%s", adopt.stderr)
	}

	if res := runCLI(t, dest, nil, "check", "--strict"); res.code != exitOK {
		t.Errorf("check --strict after adopt: %s%s", res.stdout, res.stderr)
	}
}

// TestSameOwnerLeavesTheDefaultAlone is the control.
//
// The same export adopted without the flag must behave exactly as it did before
// the flag existed. Without this, every assertion above would still pass if
// --same-owner had quietly become the only behaviour.
func TestSameOwnerLeavesTheDefaultAlone(t *testing.T) {
	dir, parentID, _ := newOwnedExport(t)

	dest := newGitStore(t)
	adopt := runCLI(t, dest, nil, "import", dir, "--adopt", "--actor", "human:sothr")
	if adopt.code != exitOK {
		t.Fatalf("import --adopt: %s%s", adopt.stdout, adopt.stderr)
	}
	rows := crossRows(t, runCLI(t, dest, nil, "--json", "list", "--all"))
	tk, _ := decode(t, runCLI(t, dest, nil, "--json", "show", rows[0]["id"].(string)).stdout)["ticket"].(map[string]any)

	for _, item := range boxes(t, tk, "acceptanceCriteria") {
		if item.Checked {
			t.Errorf("%q arrived ticked without --same-owner", item.Text)
		}
	}
	if tk["status"] != "draft" {
		t.Errorf("status = %v, want draft without --same-owner", tk["status"])
	}
	for _, raw := range tk["references"].([]any) {
		if ref, _ := raw.(map[string]any)["ref"].(string); strings.HasPrefix(ref, "origin-parent:") {
			t.Errorf("a stranger's parent ID arrived as %q, which resolves nowhere the receiver can follow", ref)
		}
	}
	for _, want := range []string{
		"acceptance criteria: 2 carried, every box unchecked",
		"parent " + parentID + " not carried, this export does not include it",
		"filed as draft",
	} {
		if !strings.Contains(adopt.stderr, want) {
			t.Errorf("the default no longer reports %q:\n%s", want, adopt.stderr)
		}
	}
}

// TestSameOwnerLandsAnUnpromotableStatusInDraft holds the one place the flag
// stops short of the sender's state.
//
// 6.2.1 lets done and archived arrive directly and refuses every other status,
// because promotion out of draft is a human call. So a ticket that left
// in-progress lands in draft, and the duty is to say so: that silent gap is the
// report this whole flag came from.
func TestSameOwnerLandsAnUnpromotableStatusInDraft(t *testing.T) {
	src := newGitStore(t)
	got := runCLI(t, src, nil, "--json", "create", "--title", "Still mid-flight", "--actor", "human:sothr")
	id, _ := decode(t, got.stdout)["ticket"].(map[string]any)["id"].(string)
	for _, args := range [][]string{
		{"ac", id, "--add", "a criterion already met"},
		{"ac", id, "--check", "1"},
		{"status", id, "ready"},
		{"status", id, "in-progress"},
	} {
		if r := runCLI(t, src, nil, append(args, "--actor", "human:sothr")...); r.code != exitOK {
			t.Fatalf("%v: %s%s", args, r.stdout, r.stderr)
		}
	}
	exportGit(t, src, "add", "-A")
	exportGit(t, src, "commit", "-qm", "store")
	dir := filepath.Join(t.TempDir(), "out")
	if r := runCLI(t, src, nil, "export", id, "--out", dir); r.code != exitOK {
		t.Fatalf("export: %s%s", r.stdout, r.stderr)
	}

	dest := newGitStore(t)
	adopt := runCLI(t, dest, nil, "import", dir, "--adopt", "--same-owner", "--actor", "human:sothr")
	if adopt.code != exitOK {
		t.Fatalf("adopt: %s%s", adopt.stdout, adopt.stderr)
	}
	rows := crossRows(t, runCLI(t, dest, nil, "--json", "list", "--all"))
	tk, _ := decode(t, runCLI(t, dest, nil, "--json", "show", rows[0]["id"].(string)).stdout)["ticket"].(map[string]any)

	if tk["status"] != "draft" {
		t.Errorf("status = %v, want draft: only done and archived may arrive directly", tk["status"])
	}
	// The evidence still travels. The status is refused by 6.2.1 and the ticks
	// are not, and conflating the two would lose the record for no reason.
	if items := boxes(t, tk, "acceptanceCriteria"); len(items) != 1 || !items[0].Checked {
		t.Errorf("acceptance criteria = %+v, want the one ticked item carried", items)
	}
	if !strings.Contains(adopt.stderr, "status in-progress: not carried, it lands in draft") {
		t.Errorf("the refused status was not named:\n%s", adopt.stderr)
	}
	// The origin is still open. A move marker keeps its dependents gated.
	if !strings.Contains(adopt.stderr, "git ticket move "+id+" --to-ref") || strings.Contains(adopt.stderr, "git ticket status "+id+" done") {
		t.Errorf("the source-side advice should use move, never a universal done transition:\n%s", adopt.stderr)
	}
}

// TestSameOwnerPreviewPromisesWhatTheAdoptDoes holds the preview to the write.
//
// --same-owner is legal without --adopt for exactly this reason. A preview that
// ignored the flag would describe a reconciliation nobody was going to run, and
// the preview's whole value is that it is the same answer.
func TestSameOwnerPreviewPromisesWhatTheAdoptDoes(t *testing.T) {
	dir, parentID, _ := newOwnedExport(t)

	preview := runCLI(t, newGitStore(t), nil, "import", dir, "--same-owner")
	if preview.code != exitOK {
		t.Fatalf("preview: %s%s", preview.stdout, preview.stderr)
	}
	for _, want := range []string{
		"acceptance criteria: 1 ticked at the origin, carried ticked",
		"definition of done: 1 ticked at the origin, carried ticked",
		"status done: carried",
		"parent " + parentID + ": kept as an origin-parent reference",
		"nothing here writes the sending store",
	} {
		if !strings.Contains(preview.stdout, want) {
			t.Errorf("the preview did not promise %q:\n%s", want, preview.stdout)
		}
	}
	// A preview writes nothing, flag or no flag.
	if rows := crossRows(t, runCLI(t, newGitStore(t), nil, "--json", "list", "--all")); len(rows) != 0 {
		t.Errorf("the preview filed %d tickets", len(rows))
	}
}

// TestSameOwnerNamesTheOriginItWillNotClose holds the boundary.
//
// import runs in the receiving store and never writes the sending one: 7.3
// forbids a sync helper rewriting another worktree, and the repository policy
// forbids writing a sibling at all. So the closing half is printed as advice,
// naming the adopted ID, which is the one thing the sender's copy cannot know.
func TestSameOwnerNamesTheOriginItWillNotClose(t *testing.T) {
	dir, _, childID := newOwnedExport(t)

	dest := newGitStore(t)
	adopt := runCLI(t, dest, nil, "import", dir, "--adopt", "--same-owner", "--actor", "human:sothr")
	if adopt.code != exitOK {
		t.Fatalf("adopt: %s%s", adopt.stdout, adopt.stderr)
	}
	newID, _ := crossRows(t, runCLI(t, dest, nil, "--json", "list", "--all"))[0]["id"].(string)

	want := `git ticket move ` + childID + ` --to-ref RECEIVER-NAMESPACE:` + newID
	if !strings.Contains(adopt.stderr, want) {
		t.Errorf("the advice does not carry %q:\n%s", want, adopt.stderr)
	}
	// This one left done, so there is no transition left to advise. Printing one
	// would be advice that fails when taken.
	if strings.Contains(adopt.stderr, "git ticket status "+childID+" done") {
		t.Errorf("advised closing a ticket that already arrived done:\n%s", adopt.stderr)
	}
}

// slicesContains is the membership test these assertions want, spelled out here
// rather than reached for, because cli has no other need of the generic.
func slicesContains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

// TestTheSharedSeriesLineAgreesInNumber covers the count of one, which is the
// case that was wrong. The sentence was written against the plural, so two and
// above always read correctly and nothing looked at one.
//
// It is cosmetic, but the line is advice about which of two routes to take, and
// a reader deciding whether to trust that advice reads the sentence carefully.
func TestTheSharedSeriesLineAgreesInNumber(t *testing.T) {
	src, first := newExportSource(t, "The first TKT ticket")

	for _, want := range []struct {
		count int
		line  string
	}{
		{1, "1 ticket already uses a series"},
		{2, "2 tickets already use a series"},
	} {
		ids := []string{first}
		if want.count == 2 {
			second := crossCreate(t, src, "The second TKT ticket", "human:sothr")
			exportGit(t, src, "add", "-A")
			exportGit(t, src, "commit", "-qm", "second")
			ids = append(ids, second)
		}

		dir := filepath.Join(t.TempDir(), "out")
		if got := runCLI(t, src, nil, append(append([]string{"export"}, ids...), "--out", dir)...); got.code != exitOK {
			t.Fatalf("export of %d: %s%s", want.count, got.stdout, got.stderr)
		}
		preview := runCLI(t, newGitStore(t), nil, "import", dir)
		if preview.code != exitOK {
			t.Fatalf("preview of %d: %s%s", want.count, preview.stdout, preview.stderr)
		}
		if !strings.Contains(preview.stdout, want.line) {
			t.Errorf("preview of %d tickets does not say %q:\n%s", want.count, want.line, preview.stdout)
		}
	}
}
