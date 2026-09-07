package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The install tests need a store inside a real repository, which newStore is
// deliberately not. newRepoStore in fix_test.go already makes one, and the
// bare-directory case is a test here rather than a gap.

// gitConfigValue reads one local key, and reports the empty string when it is
// unset, which is what `git config --get` does with status 1.
func gitConfigValue(t *testing.T, dir, key string) string {
	t.Helper()
	cmd := exec.Command("git", "config", "--local", "--get", key)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// TestInitWritesTheMergeAttribute covers the tracked half of plan 7.5. The
// attribute is inert until somebody configures a driver by that name, and
// having it committed is what lets a clone install one with a single command.
func TestInitWritesTheMergeAttribute(t *testing.T) {
	dir := newRepoStore(t)

	got, err := os.ReadFile(filepath.Join(dir, ".gitattributes"))
	if err != nil {
		t.Fatalf("init wrote no .gitattributes: %v", err)
	}
	want := ".tickets/**/*.md merge=gitticket"
	if !strings.Contains(string(got), want) {
		t.Errorf(".gitattributes does not carry %q:\n%s", want, got)
	}
}

// TestInitWritesTheEOLAttribute covers the other tracked line of plan 7.5, and
// it is the one that decides whether the store can be read at all.
//
// git converts text files to CRLF on a Windows checkout by default, and parse
// refuses a file whose fence line is "---\r". Without this line a store cloned
// on Windows answers every read with parse_error, so list reports nothing and
// names no cause. That is not hypothetical: it is what took this repository's
// own Windows lane red over 30 fixtures at once.
func TestInitWritesTheEOLAttribute(t *testing.T) {
	dir := newRepoStore(t)

	got, err := os.ReadFile(filepath.Join(dir, ".gitattributes"))
	if err != nil {
		t.Fatalf("init wrote no .gitattributes: %v", err)
	}
	want := ".tickets/**/*.md text eol=lf"
	if !strings.Contains(string(got), want) {
		t.Errorf(".gitattributes does not carry %q:\n%s", want, got)
	}
}

// TestInitAddsOnlyTheLineThatIsMissing is why 7.5 keeps the two attributes on
// two lines rather than combining them into one rule.
//
// A repository initialized before the eol line existed carries the merge line
// alone. ensureAttributes matches a whole line, so it adds what is absent and
// leaves what is there. A single combined `text eol=lf merge=gitticket` would
// match neither and append a duplicate rule beside the original.
func TestInitAddsOnlyTheLineThatIsMissing(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, so there is no repository root to write into")
	}
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}

	// Standing in for a repository from before the eol line, with a rule of
	// its own that must survive.
	attrPath := filepath.Join(dir, ".gitattributes")
	prior := "*.png binary\n.tickets/**/*.md merge=gitticket\n"
	if err := os.WriteFile(attrPath, []byte(prior), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := runCLI(t, dir, nil, "init", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("init: %s%s", got.stdout, got.stderr)
	}

	attr, err := os.ReadFile(attrPath)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(attr), "merge=gitticket"); n != 1 {
		t.Errorf("the merge line appears %d times, want the original left alone:\n%s", n, attr)
	}
	if n := strings.Count(string(attr), "text eol=lf"); n != 1 {
		t.Errorf("the eol line appears %d times, want exactly one added:\n%s", n, attr)
	}
	if !strings.Contains(string(attr), "*.png binary") {
		t.Errorf("init lost a rule the repository already had:\n%s", attr)
	}
}

// TestInitDuplicatesNeitherAttribute covers a repository that already carries
// both lines, which is a second store in a repository that has one, or a
// checkout of a template.
//
// init cannot run twice over one store, since the second attempt is
// store_exists, so this is the only route by which a line could double.
func TestInitDuplicatesNeitherAttribute(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, so there is no repository root to write into")
	}
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}

	attrPath := filepath.Join(dir, ".gitattributes")
	prior := ".tickets/**/*.md text eol=lf\n.tickets/**/*.md merge=gitticket\n"
	if err := os.WriteFile(attrPath, []byte(prior), 0o644); err != nil {
		t.Fatal(err)
	}

	got := runCLI(t, dir, nil, "--json", "init", "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("init: %s%s", got.stdout, got.stderr)
	}

	attr, err := os.ReadFile(attrPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(attr) != prior {
		t.Errorf(".gitattributes changed when both lines were already there:\n%s", attr)
	}

	// And it is not reported as written, because it was not written. A path in
	// pathsChanged that nothing changed is what makes the next reader stop
	// trusting the list.
	for _, p := range strsOf(t, decode(t, got.stdout), "pathsChanged") {
		if p == ".gitattributes" {
			t.Errorf("init reported .gitattributes as changed when it added nothing")
		}
	}
}

// TestInstallMergeDriverLeavesTheEOLLineAlone holds the command to its name.
//
// It repairs the tracked half of the merge driver, which is what it is called,
// and does not quietly add the line endings rule beside it. A store made before
// that line existed is plan 15's open question and not something to settle
// inside a command nobody ran for that reason.
func TestInstallMergeDriverLeavesTheEOLLineAlone(t *testing.T) {
	dir := newRepoStore(t)
	attrPath := filepath.Join(dir, ".gitattributes")

	// A repository carrying neither line, which is what a store from before
	// 7.5 looks like.
	if err := os.WriteFile(attrPath, []byte("*.png binary\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := runCLI(t, dir, nil, "install-merge-driver"); got.code != exitOK {
		t.Fatalf("install-merge-driver: %s%s", got.stdout, got.stderr)
	}

	attr, err := os.ReadFile(attrPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(attr), ".tickets/**/*.md merge=gitticket") {
		t.Errorf("install did not restore the merge line:\n%s", attr)
	}
	if strings.Contains(string(attr), "eol=lf") {
		t.Errorf("install-merge-driver wrote the eol line, which is init's to write:\n%s", attr)
	}
}

// commitAll stages and commits everything in a repository, with an identity on
// the command so the test does not depend on the machine having one.
func commitAll(t *testing.T, dir, message string) {
	t.Helper()
	if out, err := exec.Command("git", "-C", dir, "add", "-A").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	out, err := exec.Command("git", "-C", dir,
		"-c", "user.name=Test", "-c", "user.email=test@example.invalid",
		"commit", "-q", "-m", message).CombinedOutput()
	if err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
}

// cloneWithCRLF clones with core.autocrlf=true, which is what git does by
// default on Windows: it rewrites LF to CRLF on checkout. Forcing it here
// reproduces the exact conversion on any platform, so this is the Windows
// condition rather than a stand-in for it.
func cloneWithCRLF(t *testing.T, src string) string {
	t.Helper()
	dst := filepath.Join(t.TempDir(), "clone")
	out, err := exec.Command("git", "clone", "-q",
		"--config", "core.autocrlf=true", src, dst).CombinedOutput()
	if err != nil {
		t.Fatalf("git clone: %v\n%s", err, out)
	}
	return dst
}

// TestAStoreSurvivesACRLFClone is the criterion this whole change exists for,
// and it is an end-to-end run rather than an assertion about a line of text.
//
// A store is created, committed, and cloned with core.autocrlf=true. The clone
// must still parse. Checking that init wrote the attribute proves only that
// init wrote the attribute; this checks that the attribute does the job, which
// is the thing that was actually broken.
//
// The second half is the control. With the eol line removed the same clone
// comes back with CRLF and every read fails, so a green first half means the
// line did the work rather than something else in the setup.
func TestAStoreSurvivesACRLFClone(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}

	t.Run("with the attribute init writes", func(t *testing.T) {
		dir := newRepoStore(t)
		if got := runCLI(t, dir, nil, "create", "--title", "Survives a clone",
			"--actor", "human:sothr"); got.code != exitOK {
			t.Fatalf("create: %s%s", got.stdout, got.stderr)
		}
		commitAll(t, dir, "a store")

		clone := cloneWithCRLF(t, dir)
		assertNoCR(t, clone)

		got := runCLI(t, clone, nil, "--json", "list", "--all")
		if got.code != exitOK {
			t.Fatalf("list in the clone: %s%s", got.stdout, got.stderr)
		}
		if n := len(listedTickets(t, got.stdout)); n == 0 {
			// A store that fails to parse reports zero tickets and no error,
			// which is exactly the silent failure this guards.
			t.Errorf("the cloned store lists no tickets:\n%s", got.stdout)
		}
	})

	t.Run("without it, as a control", func(t *testing.T) {
		dir := newRepoStore(t)
		if got := runCLI(t, dir, nil, "create", "--title", "Does not survive",
			"--actor", "human:sothr"); got.code != exitOK {
			t.Fatalf("create: %s%s", got.stdout, got.stderr)
		}
		// Standing in for a store made before init wrote the line.
		if err := os.Remove(filepath.Join(dir, ".gitattributes")); err != nil {
			t.Fatal(err)
		}
		commitAll(t, dir, "a store with no attributes")

		clone := cloneWithCRLF(t, dir)
		if !hasCR(t, clone) {
			t.Skip("this git did not convert on checkout, so the control proves nothing")
		}
		got := runCLI(t, clone, nil, "--json", "list", "--all")
		if got.code == exitOK && len(listedTickets(t, got.stdout)) > 0 {
			t.Errorf("a CRLF store read fine, so the first half proves nothing:\n%s", got.stdout)
		}
	})
}

// listedTickets is the tickets array of a list envelope, which holds objects
// rather than the strings strsOf expects.
func listedTickets(t *testing.T, stdout string) []any {
	t.Helper()
	raw, ok := decode(t, stdout)["tickets"].([]any)
	if !ok {
		return nil
	}
	return raw
}

// ticketBytes reads the one ticket file a fresh store has.
func ticketBytes(t *testing.T, dir string) (string, []byte) {
	t.Helper()
	var found string
	root := filepath.Join(dir, ".tickets")
	if err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasPrefix(d.Name(), "TKT-") {
			found = p
		}
		return nil
	}); err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}
	if found == "" {
		t.Fatalf("no ticket file under %s", root)
	}
	b, err := os.ReadFile(found)
	if err != nil {
		t.Fatal(err)
	}
	return found, b
}

func hasCR(t *testing.T, dir string) bool {
	t.Helper()
	_, b := ticketBytes(t, dir)
	return strings.Contains(string(b), "\r")
}

func assertNoCR(t *testing.T, dir string) {
	t.Helper()
	path, b := ticketBytes(t, dir)
	if strings.Contains(string(b), "\r") {
		t.Errorf("%s came out of the clone with CRLF, which plan 5.3 forbids", path)
	}
}

// TestInitReportsTheAttributeItWrote holds init's own report to what it did.
// A file written and not named is a file the next person is surprised by in a
// diff, and plan 12.1 says every mutation reports its paths.
func TestInitReportsTheAttributeItWrote(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, so there is no repository root to write into")
	}
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}

	got := runCLI(t, dir, nil, "--json", "init", "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("init: %s%s", got.stdout, got.stderr)
	}
	paths := strsOf(t, decode(t, got.stdout), "pathsChanged")
	found := false
	for _, p := range paths {
		if p == ".gitattributes" {
			found = true
		}
	}
	if !found {
		t.Errorf("init wrote .gitattributes without reporting it; pathsChanged = %v", paths)
	}
}

// TestInitOutsideARepositoryWritesNoAttribute is the other side of the same
// rule. With no repository root there is nowhere the attribute would apply, so
// init leaves the directory alone rather than dropping a file that does
// nothing.
func TestInitOutsideARepositoryWritesNoAttribute(t *testing.T) {
	dir := newStore(t) // deliberately not a repository

	if _, err := os.Stat(filepath.Join(dir, ".gitattributes")); !os.IsNotExist(err) {
		t.Errorf("init wrote .gitattributes outside a repository (stat error: %v)", err)
	}
}

// TestInstallMergeDriverSetsBothKeys covers the untracked half of plan 7.5.
// Git refuses to take an executable name from a repository, so these two keys
// are the person's own decision, and this command is how they make it once.
func TestInstallMergeDriverSetsBothKeys(t *testing.T) {
	dir := newRepoStore(t)

	got := runCLI(t, dir, nil, "install-merge-driver")
	if got.code != exitOK {
		t.Fatalf("install-merge-driver: %s%s", got.stdout, got.stderr)
	}

	if name := gitConfigValue(t, dir, "merge.gitticket.name"); name != "git-ticket three-way merge" {
		t.Errorf("merge.gitticket.name is %q", name)
	}
	driver := gitConfigValue(t, dir, "merge.gitticket.driver")
	if !strings.HasSuffix(driver, " merge-driver %O %A %B") {
		t.Errorf("merge.gitticket.driver does not end in git's placeholders: %q", driver)
	}
	// An absolute path, because a bare name resolves against whatever PATH
	// git happens to have when it runs the driver, and the failure then looks
	// like a merge conflict rather than a missing command.
	exe := strings.TrimSuffix(driver, " merge-driver %O %A %B")
	exe = strings.Trim(exe, "'")
	if !filepath.IsAbs(exe) {
		t.Errorf("driver names a relative executable %q", exe)
	}
}

// TestInstallMergeDriverIsIdempotent holds the second run. Somebody who cannot
// tell whether the first run worked will run it again, and that has to be a
// report rather than an error or a duplicated attribute line.
func TestInstallMergeDriverIsIdempotent(t *testing.T) {
	dir := newRepoStore(t)

	first := runCLI(t, dir, nil, "install-merge-driver")
	if first.code != exitOK {
		t.Fatalf("first install: %s%s", first.stdout, first.stderr)
	}
	second := runCLI(t, dir, nil, "install-merge-driver")
	if second.code != exitOK {
		t.Fatalf("second install: %s%s", second.stdout, second.stderr)
	}
	if !strings.Contains(second.stdout, "already set merge.gitticket.driver") {
		t.Errorf("the second run does not report the driver as already set:\n%s", second.stdout)
	}

	attr, err := os.ReadFile(filepath.Join(dir, ".gitattributes"))
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(attr), "merge=gitticket"); n != 1 {
		t.Errorf("the attribute appears %d times after two installs:\n%s", n, attr)
	}
}

// TestInstallMergeDriverRestoresTheAttribute covers a store made before the
// attribute existed, and a repository where somebody removed the line. The
// command is the one place that knows both halves, so it repairs the tracked
// half rather than assuming init got there first.
func TestInstallMergeDriverRestoresTheAttribute(t *testing.T) {
	dir := newRepoStore(t)
	attrPath := filepath.Join(dir, ".gitattributes")

	// Standing in for a repository from before this feature, with a rule of
	// its own that must survive.
	if err := os.WriteFile(attrPath, []byte("*.png binary\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := runCLI(t, dir, nil, "install-merge-driver"); got.code != exitOK {
		t.Fatalf("install-merge-driver: %s%s", got.stdout, got.stderr)
	}

	attr, err := os.ReadFile(attrPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(attr), ".tickets/**/*.md merge=gitticket") {
		t.Errorf("install did not restore the attribute:\n%s", attr)
	}
	if !strings.Contains(string(attr), "*.png binary") {
		t.Errorf("install lost a rule the repository already had:\n%s", attr)
	}
}

// TestInstallMergeDriverNeedsARepository fails rather than half-installing.
// The config half has nowhere to go without a repository, and an install that
// wrote one half and reported success would be worse than a refusal.
func TestInstallMergeDriverNeedsARepository(t *testing.T) {
	dir := newStore(t) // no repository

	got := runCLI(t, dir, nil, "install-merge-driver")
	if got.code != exitError {
		t.Fatalf("install-merge-driver succeeded outside a repository: %s%s", got.stdout, got.stderr)
	}
	if !strings.Contains(got.stderr, "Git repository") {
		t.Errorf("the error does not say what is missing: %s", got.stderr)
	}
}
