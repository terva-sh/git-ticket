package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestMergeDriverHelper is not a test. It lets the test binary stand in for the
// installed executable, so a real `git merge` can invoke the real driver
// without a `go build` at test time.
//
// TestMergeDriverUnderRealGit points git's merge.*.driver at this binary with
// the environment variable set, and git runs it the way it would run anything
// else named in a config line.
func TestMergeDriverHelper(t *testing.T) {
	if os.Getenv("GIT_TICKET_DRIVER_HELPER") != "1" {
		t.Skip("not the helper process")
	}
	args := []string{}
	for i, a := range os.Args {
		if a == "--" {
			args = os.Args[i+1:]
			break
		}
	}
	os.Exit(Run(args, Env{
		Dir:    ".",
		Getenv: os.Getenv,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Now:    time.Now,
	}))
}

// TestMergeDriverUnderRealGit is the proof the spike asked for, run through git
// rather than reasoned about.
//
// Branch A sets priority and branch B adds a label. They disagree about
// nothing, and before the driver this merge stopped on updated_by, a field
// neither agent typed.
func TestMergeDriverUnderRealGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Skipf("no test executable to stand in for the driver: %v", err)
	}

	dir := t.TempDir()
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		dir = resolved
	}
	git := func(args ...string) (string, error) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_TICKET_DRIVER_HELPER=1",
			"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com")
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	mustGit := func(args ...string) string {
		t.Helper()
		out, err := git(args...)
		if err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
		return out
	}

	mustGit("init", "-q", "-b", "main")
	if got := runCLI(t, dir, nil, "init", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("init: %s", got.stderr)
	}
	got := runCLI(t, dir, nil, "--json", "create", "--title", "Shared", "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("create: %s", got.stderr)
	}
	id := ticketID(t, decode(t, got.stdout))
	if r := runCLI(t, dir, nil, "status", id, "ready", "--actor", "human:sothr"); r.code != exitOK {
		t.Fatalf("status: %s", r.stderr)
	}

	// The two halves of installing a driver, per plan 7.5. The attributes file
	// is tracked and the config line is not, which is the whole reason a
	// repository cannot set this up for itself.
	if err := os.WriteFile(filepath.Join(dir, ".gitattributes"),
		[]byte(".tickets/**/*.md merge=gitticket\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGit("config", "merge.gitticket.name", "git-ticket three-way merge")
	mustGit("config", "merge.gitticket.driver",
		exe+" -test.run=^TestMergeDriverHelper$ -- merge-driver %O %A %B")

	mustGit("add", "-A")
	mustGit("commit", "-qm", "base")

	mustGit("switch", "-qc", "agent-a")
	if r := runCLI(t, dir, nil, "update", id, "--priority", "high", "--actor", "agent:a"); r.code != exitOK {
		t.Fatalf("update a: %s", r.stderr)
	}
	mustGit("commit", "-qam", "a: priority")

	mustGit("switch", "-q", "main")
	mustGit("switch", "-qc", "agent-b")
	if r := runCLI(t, dir, nil, "update", id, "--add-label", "docs", "--actor", "agent:b"); r.code != exitOK {
		t.Fatalf("update b: %s", r.stderr)
	}
	mustGit("commit", "-qam", "b: label")

	// Assert the premise before the conclusion. If these two commits ever stop
	// touching updated_by, this test passes without exercising the driver.
	if out := mustGit("diff", "main", "agent-a", "--", ".tickets/"); !strings.Contains(out, "updated_by") {
		t.Fatalf("branch a did not touch updated_by, so this no longer covers the case:\n%s", out)
	}

	if out, err := git("merge", "agent-a"); err != nil {
		t.Fatalf("the merge did not resolve:\n%s", out)
	}

	// Both edits survived, and the file is a ticket rather than a marked one.
	shown := runCLI(t, dir, nil, "--json", "show", id)
	if shown.code != exitOK {
		t.Fatalf("show after the merge: %s", shown.stderr)
	}
	tk := decode(t, shown.stdout)["ticket"].(map[string]any)
	if tk["priority"] != "high" {
		t.Errorf("priority = %v, want high from branch a", tk["priority"])
	}
	labels, _ := tk["labels"].([]any)
	if len(labels) != 1 || labels[0] != "docs" {
		t.Errorf("labels = %v, want the label branch b added", labels)
	}
}
