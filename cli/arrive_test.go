package cli

import (
	"os"
	"strings"
	"testing"
)

// TestCreateArrivesDoneFromTheCLI drives plan 6.2.1 end to end: a backported
// ticket files straight into done/, backdated, and the draft gate holds for
// every other status.
func TestCreateArrivesDoneFromTheCLI(t *testing.T) {
	dir := newGitStore(t)

	got := runCLI(t, dir, nil, "--json", "--actor", "human:sothr",
		"create", "--title", "Shipped before the migration",
		"--status", "done", "--created", "2020-03-14")
	if got.code != exitOK {
		t.Fatalf("create --status done: exit %d\nstderr: %s", got.code, got.stderr)
	}
	env := decode(t, got.stdout)
	path := env["pathsChanged"].([]any)[0].(string)
	if !strings.Contains(path, ".tickets/done/") {
		t.Fatalf("path = %q, want it under .tickets/done/", path)
	}
	data, err := os.ReadFile(dir + "/" + path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "2020-03-14") {
		t.Fatalf("the file does not carry the backdated day:\n%s", data)
	}
}

func TestCreateStatusRefusalsFromTheCLI(t *testing.T) {
	dir := newGitStore(t)

	// The draft gate: ready is not a status a create may name.
	got := runCLI(t, dir, nil, "--actor", "human:sothr",
		"create", "--title", "No shortcut", "--status", "ready")
	if got.code == exitOK {
		t.Fatalf("create --status ready succeeded; the draft gate is gone")
	}
	if !strings.Contains(got.stderr, "done and archived") {
		t.Fatalf("refusal does not name the permitted statuses: %s", got.stderr)
	}

	// A malformed --created is settled before the store opens.
	got = runCLI(t, dir, nil, "--actor", "human:sothr",
		"create", "--title", "Bad instant", "--created", "last tuesday")
	if got.code == exitOK {
		t.Fatalf("create --created 'last tuesday' succeeded")
	}
	if !strings.Contains(got.stderr, "RFC 3339") {
		t.Fatalf("refusal does not teach the spelling: %s", got.stderr)
	}
}

// TestDraftClosesDoneWithAReason: the 6.2 addition through the status
// command, both halves of the rule.
func TestDraftClosesDoneWithAReason(t *testing.T) {
	dir := newGitStore(t)
	got := runCLI(t, dir, nil, "--json", "--actor", "human:sothr",
		"create", "--title", "Turned out to exist already")
	if got.code != exitOK {
		t.Fatalf("create: %s", got.stderr)
	}
	id := decode(t, got.stdout)["ticket"].(map[string]any)["id"].(string)

	got = runCLI(t, dir, nil, "--actor", "human:sothr", "status", id, "done")
	if got.code == exitOK {
		t.Fatalf("draft went done without a reason")
	}
	if !strings.Contains(got.stderr, "reason") {
		t.Fatalf("refusal does not ask for the reason: %s", got.stderr)
	}

	got = runCLI(t, dir, nil, "--actor", "human:sothr",
		"status", id, "done", "--reason", "shipped as PR #14 in the old repo")
	if got.code != exitOK {
		t.Fatalf("draft to done with a reason: exit %d\nstderr: %s", got.code, got.stderr)
	}

	show := runCLI(t, dir, nil, "show", id)
	if !strings.Contains(show.stdout, "shipped as PR #14") {
		t.Fatalf("the reason is not readable back:\n%s", show.stdout)
	}
}
