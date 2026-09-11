package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The point of an export is that somebody else can apply it, so these tests
// assert against git rather than against the bytes we wrote. A patch that looks
// right and does not apply is the failure worth catching, and only git knows the
// difference.

// exportGit runs git with a fixed identity, failing the test if it errors.
func exportGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// newExportSource is a store with one ticket, committed.
func newExportSource(t *testing.T, title string) (dir, id string) {
	t.Helper()
	dir = newGitStore(t)
	id = crossCreate(t, dir, title, "human:sothr")
	if got := runCLI(t, dir, nil, "status", id, "ready", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("status ready: %s%s", got.stdout, got.stderr)
	}
	exportGit(t, dir, "add", "-A")
	exportGit(t, dir, "commit", "-qm", "store")
	return dir, id
}

// newExportDest is a repository with a store, ready to receive.
func newExportDest(t *testing.T) string {
	t.Helper()
	return newGitStore(t)
}

// applyExport applies every patch in an export with plain `git am`.
//
// No flags. That is the assertion: the artifact needs nothing but git, and it
// needs no git newer than whatever is running the suite. A cover letter named
// .patch would break this line, which is why it is not.
func applyExport(t *testing.T, dest, exportDir string) {
	t.Helper()
	patches, err := filepath.Glob(filepath.Join(exportDir, "*.patch"))
	if err != nil || len(patches) == 0 {
		t.Fatalf("no patches in %s (err %v)", exportDir, err)
	}
	exportGit(t, dest, append([]string{"am"}, patches...)...)
}

// TestExportRoundTripsABareTicket is the first acceptance criterion: a ticket
// with no code change still exports, as a patch that adds the ticket file where
// its status says it belongs, and the file arrives byte for byte.
func TestExportRoundTripsABareTicket(t *testing.T) {
	src, id := newExportSource(t, "A bare ticket")
	out := filepath.Join(t.TempDir(), "out")

	if got := runCLI(t, src, nil, "export", id, "--out", out); got.code != exitOK {
		t.Fatalf("export: %s%s", got.stdout, got.stderr)
	}

	// The cover is not a .patch, so the glob a reader types skips it.
	if _, err := os.Stat(filepath.Join(out, "0000-cover-letter.txt")); err != nil {
		t.Errorf("no cover letter: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "0000-cover-letter.patch")); err == nil {
		t.Error("the cover letter is named .patch, so `git am *.patch` will stop on it")
	}

	dest := newExportDest(t)
	applyExport(t, dest, out)

	// ready is the status set above, so the file belongs in tickets/.
	landed := filepath.Join(dest, ".tickets", "tickets", id+".md")
	got, err := os.ReadFile(landed)
	if err != nil {
		t.Fatalf("ticket did not land at %s: %v", landed, err)
	}
	want, err := os.ReadFile(filepath.Join(src, ".tickets", "tickets", id+".md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Error("the applied ticket differs from the source")
	}

	// And the receiving store is consistent, which is the whole point of
	// choosing the directory from the status at export time.
	if res := runCLI(t, dest, nil, "check", "--strict"); res.code != exitOK {
		t.Errorf("check on the receiving store: %s%s", res.stdout, res.stderr)
	}
}

// TestExportCarriesADraftIntoItsOwnDirectory covers the other side of the status
// rule. A draft is exportable, and it lands in draft/ rather than anywhere else,
// so location_mismatch cannot fire on arrival.
func TestExportCarriesADraftIntoItsOwnDirectory(t *testing.T) {
	src := newGitStore(t)
	id := crossCreate(t, src, "Still a draft", "human:sothr")
	exportGit(t, src, "add", "-A")
	exportGit(t, src, "commit", "-qm", "store")
	out := filepath.Join(t.TempDir(), "out")

	if got := runCLI(t, src, nil, "export", id, "--out", out); got.code != exitOK {
		t.Fatalf("export: %s%s", got.stdout, got.stderr)
	}
	dest := newExportDest(t)
	applyExport(t, dest, out)

	if _, err := os.Stat(filepath.Join(dest, ".tickets", "draft", id+".md")); err != nil {
		t.Errorf("a draft did not land in draft/: %v", err)
	}
	if res := runCLI(t, dest, nil, "check", "--strict"); res.code != exitOK {
		t.Errorf("check on the receiving store: %s%s", res.stdout, res.stderr)
	}
}

// TestExportComposesWithFormatPatch is the second acceptance criterion: several
// tickets in one export, and code travelling with them.
//
// Export runs no git, so it does not carry the code itself. It reserves numbers
// 0 and 1 and leaves the rest to git's own format-patch, which is the tool for
// that job and is already in everybody's hands. The test asserts the composition
// rather than a flag: two tickets from export, one commit from format-patch, and
// a single `git am *.patch` that applies all of it with the original commit
// message intact.
func TestExportComposesWithFormatPatch(t *testing.T) {
	src, first := newExportSource(t, "First ticket")
	second := crossCreate(t, src, "Second ticket", "human:sothr")
	exportGit(t, src, "add", "-A")
	exportGit(t, src, "commit", "-qm", "second")

	// A branch with one commit, which format-patch will carry alongside.
	if err := os.WriteFile(filepath.Join(src, "app.txt"), []byte("before\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	exportGit(t, src, "add", "-A")
	exportGit(t, src, "commit", "-qm", "app")
	exportGit(t, src, "switch", "-qc", "fix")
	if err := os.WriteFile(filepath.Join(src, "app.txt"), []byte("after\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	exportGit(t, src, "commit", "-qam", "Change the app")
	exportGit(t, src, "switch", "-q", "main")

	out := filepath.Join(t.TempDir(), "out")
	if got := runCLI(t, src, nil, "export", first, second, "--out", out); got.code != exitOK {
		t.Fatalf("export: %s%s", got.stdout, got.stderr)
	}
	// The sender attaches the code the ordinary way. Numbering from 2 is what
	// export leaves room for.
	exportGit(t, src, "format-patch", "--start-number", "2", "-o", out, "main..fix")

	dest := newExportDest(t)
	// The destination needs the file the code patch changes.
	if err := os.WriteFile(filepath.Join(dest, "app.txt"), []byte("before\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	exportGit(t, dest, "add", "-A")
	exportGit(t, dest, "commit", "-qm", "app")

	applyExport(t, dest, out)

	for _, id := range []string{first, second} {
		if _, err := os.Stat(filepath.Join(dest, ".tickets", "tickets", id+".md")); err != nil {
			if _, err2 := os.Stat(filepath.Join(dest, ".tickets", "draft", id+".md")); err2 != nil {
				t.Errorf("ticket %s did not land: %v", id, err)
			}
		}
	}
	if body, err := os.ReadFile(filepath.Join(dest, "app.txt")); err != nil || string(body) != "after\n" {
		t.Errorf("the code change did not arrive: %q %v", body, err)
	}
	// The original commit message survives rather than being squashed away.
	if log := exportGit(t, dest, "log", "--oneline"); !strings.Contains(log, "Change the app") {
		t.Errorf("the range's commit message was lost:\n%s", log)
	}
}

// TestExportSaysHowToAttachCode holds the one thing standing in for a --patch
// flag. Plan 12.8 has export run no git, so code composes in from outside, and
// section 15 records that the discoverability cost of composing was paid here
// rather than by admitting format-patch to the 7.4 table.
//
// A sender who never learns to compose does not get an error. They ship an
// export with no code in it and find out from the receiver, which is the failure
// this text exists to prevent. So the line carries the real directory rather
// than a placeholder, and it is printed beside the `git am` line every export
// already prints.
func TestExportSaysHowToAttachCode(t *testing.T) {
	src, id := newExportSource(t, "A ticket")
	out := filepath.Join(t.TempDir(), "out")
	got := runCLI(t, src, nil, "export", id, "--out", out)
	if got.code != exitOK {
		t.Fatalf("export: %s%s", got.stdout, got.stderr)
	}
	// The directory is filled in, so the line can be run rather than edited.
	want := "git format-patch --start-number 2 -o " + out
	if !strings.Contains(got.stderr, want) {
		t.Errorf("export did not say how to attach code.\nwant a line containing: %s\ngot:\n%s", want, got.stderr)
	}
	if !strings.Contains(got.stderr, "git am "+out) {
		t.Errorf("export stopped naming how to apply it:\n%s", got.stderr)
	}

	// --help has to carry it too, because the flag list alone reads as though an
	// export can only ever hold tickets.
	help := runCLI(t, src, nil, "export", "--help")
	if !strings.Contains(help.stdout, "format-patch --start-number 2") {
		t.Errorf("export --help does not name the composition:\n%s", help.stdout)
	}
}

// TestExportWarnsAboutEdgesUnderJSON keeps a warning from being swallowed by the
// mode most likely to miss it. An edge naming a ticket the export does not carry
// is parent_missing or dependency_missing on arrival, which are errors, and a
// caller passing --json is the one least likely to be reading the artifact by
// eye. Warnings go to stderr in both modes, as the actor warning already does.
func TestExportWarnsAboutEdgesUnderJSON(t *testing.T) {
	src, stays := newExportSource(t, "Ticket that travels")
	behind := crossCreate(t, src, "Ticket left behind", "human:sothr")
	if got := runCLI(t, src, nil, "link", stays, "--depends-on", behind, "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("link: %s%s", got.stdout, got.stderr)
	}
	out := filepath.Join(t.TempDir(), "out")
	got := runCLI(t, src, nil, "--json", "export", stays, "--out", out)
	if got.code != exitOK {
		t.Fatalf("export --json: %s%s", got.stdout, got.stderr)
	}
	if !strings.Contains(got.stderr, behind) {
		t.Errorf("--json swallowed the dangling edge warning:\n%s", got.stderr)
	}
	// And stdout stays one envelope, so the warning did not land in the JSON.
	if !strings.HasPrefix(strings.TrimSpace(got.stdout), "{") {
		t.Errorf("stdout is not a bare envelope:\n%s", got.stdout)
	}
}

// TestExportRefusesANonEmptyDirectory keeps two exports from being mixed under
// one set of numbers, where `git am *.patch` would apply both silently.
func TestExportRefusesANonEmptyDirectory(t *testing.T) {
	src, id := newExportSource(t, "A ticket")
	out := filepath.Join(t.TempDir(), "out")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "0001-old.patch"), []byte("stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := runCLI(t, src, nil, "export", id, "--out", out)
	if got.code == exitOK {
		t.Fatal("export wrote into a directory that already held a patch")
	}
	if !strings.Contains(got.stderr, "not empty") {
		t.Errorf("stderr = %q, want it to say the directory is not empty", got.stderr)
	}
}

// TestExportNeedsATicket covers the usage error, so the message stays a sentence
// rather than a flag dump.
func TestExportNeedsATicket(t *testing.T) {
	src, _ := newExportSource(t, "A ticket")
	got := runCLI(t, src, nil, "export")
	if got.code == exitOK {
		t.Fatal("export with no ID succeeded")
	}
	if !strings.Contains(got.stderr, "at least one ticket ID") {
		t.Errorf("stderr = %q, want it to name what is missing", got.stderr)
	}
}

// TestExportJSONReportsWhatItWrote keeps the scripted path honest: the envelope
// names every file, because a caller that cannot see stdout still has to find
// the artifact.
func TestExportJSONReportsWhatItWrote(t *testing.T) {
	src, id := newExportSource(t, "A ticket")
	out := filepath.Join(t.TempDir(), "out")
	got := runCLI(t, src, nil, "--json", "export", id, "--out", out)
	if got.code != exitOK {
		t.Fatalf("export --json: %s%s", got.stdout, got.stderr)
	}
	env := decode(t, got.stdout)
	if env["kind"] != "mutation-result" {
		t.Errorf("kind = %v, want mutation-result", env["kind"])
	}
	paths, _ := env["pathsChanged"].([]any)
	if len(paths) != 2 {
		t.Fatalf("pathsChanged = %v, want the cover and one patch", paths)
	}
	if first, _ := paths[0].(string); !strings.HasSuffix(first, ".txt") {
		t.Errorf("first path = %v, want the cover letter first", paths[0])
	}
}

// TestBlobSHAMatchesGit checks the index line against git's own hashing. A wrong
// blob name still applies, but it takes away the object git needs for a --3way
// fallback, and the failure would only appear on a patch that conflicts.
func TestBlobSHAMatchesGit(t *testing.T) {
	dir := newGitStore(t)
	body := []byte("---\nid: TKT-1\n---\n\n## Description\n\nhi\n")
	path := filepath.Join(dir, "probe.md")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	want := strings.TrimSpace(exportGit(t, dir, "hash-object", path))
	if got := blobSHA(body); got != want {
		t.Errorf("blobSHA = %s, want %s", got, want)
	}
}

// TestExportWarnsAboutEdgesThatWillNotTravel covers the failure the bootstrap
// delivery found: `git am` applies a ticket verbatim, so a parent or dependency
// naming a ticket outside the export arrives pointing at nothing, and both are
// errors rather than warnings. The artifact applies without complaint and leaves
// the receiving store broken, so the warning has to happen while the sender can
// still act on it.
func TestExportWarnsAboutEdgesThatWillNotTravel(t *testing.T) {
	src, kept := newExportSource(t, "Ticket that travels")
	left := crossCreate(t, src, "Ticket left behind", "human:sothr")
	if got := runCLI(t, src, nil, "link", kept, "--depends-on", left, "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("link: %s%s", got.stdout, got.stderr)
	}
	exportGit(t, src, "add", "-A")
	exportGit(t, src, "commit", "-qm", "second")

	out := filepath.Join(t.TempDir(), "out")
	got := runCLI(t, src, nil, "export", kept, "--out", out)
	if got.code != exitOK {
		t.Fatalf("export: %s%s", got.stdout, got.stderr)
	}
	if !strings.Contains(got.stderr, left) {
		t.Errorf("stderr did not name the edge that will not travel:\n%s", got.stderr)
	}
	if !strings.Contains(got.stderr, "git ticket import") {
		t.Errorf("stderr did not name the way out:\n%s", got.stderr)
	}
}
