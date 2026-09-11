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
