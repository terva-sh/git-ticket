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
