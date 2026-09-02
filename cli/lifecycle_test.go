package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLifecycle is the Phase 2 exit criterion: a scripted end-to-end run from
// create through claim through done through archive.
//
// It reads top to bottom as the sequence a person or an agent actually types,
// against one store in a real repository, and it checks the store at the end.
// Every other test in this package takes one command apart. This one is here to
// catch what only shows up when they are used in order, such as a mutation that
// works alone but leaves the store failing check.
func TestLifecycle(t *testing.T) {
	dir := newGitStore(t)
	const actor = "human:sothr"

	// step runs one command, fails the test with its output if it did not
	// succeed, and returns the parsed envelope.
	step := func(what string, args ...string) map[string]any {
		t.Helper()
		got := runCLI(t, dir, nil, append([]string{"--json", "--actor", actor}, args...)...)
		if got.code != exitOK {
			t.Fatalf("%s: exit %d\nstdout: %s\nstderr: %s", what, got.code, got.stdout, got.stderr)
		}
		return decode(t, got.stdout)
	}

	// Create. A new ticket is a draft, per 6.1.
	created := step("create",
		"create", "--title", "Rotate the signing key without downtime",
		"--type", "task", "--priority", "high", "--label", "auth")
	id := ticketID(t, created)
	if s := showTicket(t, dir, id)["status"]; s != "draft" {
		t.Fatalf("a new ticket is %v, want draft", s)
	}

	// Ready. Now it is work somebody could pick up, and `ready` lists it.
	step("status ready", "status", id, "ready")
	ready := step("list ready", "list", "--status", "ready")
	if n := len(ready["tickets"].([]any)); n != 1 {
		t.Errorf("list --status ready returned %d tickets, want 1", n)
	}

	// Claim. An agent records that it is working this one, and where.
	step("claim", "claim", id)
	claim, ok := showTicket(t, dir, id)["claim"].(map[string]any)
	if !ok {
		t.Fatal("the ticket carries no claim after claim")
	}
	if claim["actor"] != actor {
		t.Errorf("claim actor = %v, want %s", claim["actor"], actor)
	}
	if claim["branch"] != "main" {
		t.Errorf("claim branch = %v, want main", claim["branch"])
	}

	// In progress, then done. A claim is metadata, so the status still has to
	// move on its own, per 6.4.
	step("status in-progress", "status", id, "in-progress")
	step("status done", "status", id, "done")
	if s := showTicket(t, dir, id)["status"]; s != "done" {
		t.Fatalf("status = %v, want done", s)
	}

	// Archive. This is the step that also moves the file, which is why it is
	// its own command and not a status.
	archived := step("archive", "archive", id, "--reason", "shipped in v1.2")
	paths := archived["pathsChanged"].([]any)
	if len(paths) != 2 {
		t.Errorf("pathsChanged = %v, want both ends of the move", paths)
	}
	if _, err := os.Stat(filepath.Join(dir, ".tickets", "archive", id+".md")); err != nil {
		t.Errorf("the ticket is not in archive/: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".tickets", "tickets", id+".md")); !os.IsNotExist(err) {
		t.Error("the ticket is still in tickets/")
	}

	final := showTicket(t, dir, id)
	if final["status"] != "archived" {
		t.Errorf("status = %v, want archived", final["status"])
	}
	// from_status done is what lets a dependent treat this as satisfied.
	if from := final["archive"].(map[string]any)["fromStatus"]; from != "done" {
		t.Errorf("fromStatus = %v, want done", from)
	}

	// An archived ticket is out of the default listing and in the archived one.
	if n := len(step("list", "list")["tickets"].([]any)); n != 0 {
		t.Errorf("the default listing still shows %d tickets", n)
	}
	if n := len(step("list --archived", "list", "--archived")["tickets"].([]any)); n != 1 {
		t.Error("list --archived does not show the archived ticket")
	}

	// The store this run produced is valid. A lifecycle that ends in a store
	// check refuses is the failure this test exists to catch.
	report := step("check", "check", "--strict")
	if report["ok"] != true {
		t.Errorf("check is not green after the full lifecycle:\n%s", report)
	}

	// The reason is in the archive block, which is where 5.1 puts it.
	//
	// It lands in Notes as well, per 6.3, and for the reason 6.3 gives:
	// unarchive deletes the block, so without the note a ticket archived as
	// "shipped in v1.2" and then unarchived would keep nothing saying why it was
	// ever closed out. A status reason lands in two places for the same reason,
	// so there is no asymmetry between them. This asserts the block, and
	// TestArchiveMovesTheFileAndRecordsFromStatus asserts the note.
	if reason := final["archive"].(map[string]any)["reason"]; reason != "shipped in v1.2" {
		t.Errorf("archive reason = %v, want the reason the command was given", reason)
	}
}

// TestLifecycleReopen covers the other way through the table: a done ticket
// that comes back. Reopening needs a reason, per 6.2, because a ticket that
// silently un-finishes is how a status stops meaning anything.
func TestLifecycleReopen(t *testing.T) {
	dir := newStore(t)
	id := readyTicket(t, dir)
	for _, s := range []string{"in-progress", "done"} {
		if got := runCLI(t, dir, nil, "status", id, s, "--actor", "human:sothr"); got.code != exitOK {
			t.Fatalf("status %s: %s", s, got.stderr)
		}
	}

	if got := runCLI(t, dir, nil, "--json", "status", id, "in-progress",
		"--actor", "human:sothr"); got.code != exitError {
		t.Fatal("reopening with no reason should be refused")
	}

	if got := runCLI(t, dir, nil, "status", id, "in-progress",
		"--reason", "the fix regressed on staging", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("reopen: %s", got.stderr)
	}
	tk := showTicket(t, dir, id)
	if tk["status"] != "in-progress" {
		t.Errorf("status = %v, want in-progress", tk["status"])
	}
	if tk["statusReason"] != "the fix regressed on staging" {
		t.Errorf("statusReason = %v", tk["statusReason"])
	}

	// An archived ticket can come back too, and lands in ready.
	if got := runCLI(t, dir, nil, "status", id, "done",
		"--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("status done: %s", got.stderr)
	}
	runCLI(t, dir, nil, "archive", id, "--actor", "human:sothr")
	if got := runCLI(t, dir, nil, "unarchive", id, "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("unarchive: %s", got.stderr)
	}
	if s := showTicket(t, dir, id)["status"]; s != "ready" {
		t.Errorf("status = %v, want ready after unarchive", s)
	}
}
