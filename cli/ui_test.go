package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/terva-sh/git-ticket/ticket"
)

// runUIEnv invokes the CLI with a RunUI binding, which runCLI leaves
// nil because only this command has one.
func runUIEnv(t *testing.T, dir string, runUI func(*ticket.Store) error, args ...string) result {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(args, Env{
		Dir:    dir,
		Getenv: func(string) string { return "" },
		Stdout: &stdout,
		Stderr: &stderr,
		Now:    func() time.Time { return referenceInstant },
		RunUI:  runUI,
	})
	return result{stdout: stdout.String(), stderr: stderr.String(), code: code}
}

func TestUIRunsTheWiredRunner(t *testing.T) {
	dir := newStore(t)
	var got *ticket.Store
	calls := 0
	res := runUIEnv(t, dir, func(s *ticket.Store) error {
		got = s
		calls++
		return nil
	}, "ui")
	if res.code != exitOK {
		t.Fatalf("ui failed: %s%s", res.stdout, res.stderr)
	}
	if calls != 1 {
		t.Fatalf("runner called %d times, want 1", calls)
	}
	want, err := filepath.EvalSymlinks(filepath.Join(dir, ticket.StoreDirName))
	if err != nil {
		t.Fatal(err)
	}
	have, err := filepath.EvalSymlinks(got.Path())
	if err != nil {
		t.Fatal(err)
	}
	if have != want {
		t.Fatalf("runner got store %q, want %q", have, want)
	}
}

func TestUIWithoutARunnerSaysSo(t *testing.T) {
	dir := newStore(t)
	res := runUIEnv(t, dir, nil, "ui")
	if res.code == exitOK {
		t.Fatalf("ui with no runner succeeded")
	}
	if !strings.Contains(res.stderr, "no terminal UI wired") {
		t.Fatalf("stderr = %q, want the unwired message", res.stderr)
	}
}

func TestUIRefusesArgumentsAndJSON(t *testing.T) {
	dir := newStore(t)
	noRun := func(*ticket.Store) error {
		t.Fatal("runner must not run on a refused invocation")
		return nil
	}
	if res := runUIEnv(t, dir, noRun, "ui", "extra"); res.code == exitOK {
		t.Fatalf("ui with an argument succeeded")
	}
	if res := runUIEnv(t, dir, noRun, "ui", "--json"); res.code == exitOK {
		t.Fatalf("ui --json succeeded")
	} else if !strings.Contains(res.stderr, "no --json form") {
		t.Fatalf("stderr = %q, want the no-JSON refusal", res.stderr)
	}
}

func TestUIOutsideAStoreFails(t *testing.T) {
	dir := t.TempDir() // no store
	called := false
	res := runUIEnv(t, dir, func(*ticket.Store) error { called = true; return nil }, "ui")
	if res.code == exitOK {
		t.Fatalf("ui outside a store succeeded")
	}
	if called {
		t.Fatalf("runner ran without a store")
	}
}
