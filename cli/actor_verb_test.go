package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// emptyRosterStore is a store as `init` leaves one when nobody named an actor
// and there was no terminal to ask: valid, and refusing any write that names
// nobody. It is the state TKT-01M2NT7QS84SB5P7DR2GXB9DZ0 is about.
func emptyRosterStore(t *testing.T) string {
	t.Helper()
	dir := newStore(t)
	path := filepath.Join(dir, ".tickets", "config.yml")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	out := strings.Replace(string(body),
		"actors:\n  - id: human:sothr\n    name: \"\"\n", "actors: []\n", 1)
	if out == string(body) {
		t.Fatalf("the fixture config did not have the roster this test empties:\n%s", body)
	}
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return dir
}

// TestActorAddFixesAnEmptyRoster is the whole ticket: a store that could only be
// repaired with a text editor can now be repaired with a command.
func TestActorAddFixesAnEmptyRoster(t *testing.T) {
	dir := emptyRosterStore(t)

	// The state before: a write naming nobody is refused.
	if got := runCLI(t, dir, nil, "create", "--title", "refused"); got.code == exitOK {
		t.Fatalf("a write with no actor succeeded against an empty roster: %s", got.stdout)
	}

	// The read works even here, which is the point: this is the command you run
	// when you suspect the roster is the problem.
	got := runCLI(t, dir, nil, "actor")
	if got.code != exitOK {
		t.Fatalf("actor exited %d against an empty roster: %s", got.code, got.stderr)
	}
	if !strings.Contains(got.stdout, "No actors") {
		t.Errorf("the empty roster was not reported as empty:\n%s", got.stdout)
	}

	if got := runCLI(t, dir, nil, "actor", "add", "human:you", "--name", "You", "--default"); got.code != exitOK {
		t.Fatalf("actor add exited %d: %s", got.code, got.stderr)
	}

	// And the state after: the same write now lands, with no warning, because
	// --default made the choice deliberate rather than incidental.
	after := runCLI(t, dir, nil, "create", "--title", "now it works")
	if after.code != exitOK {
		t.Fatalf("a write still fails after adding an actor: %s%s", after.stdout, after.stderr)
	}
	if after.stderr != "" {
		t.Errorf("adding an actor with --default still warns: %q", after.stderr)
	}
}

// TestActorAddWithoutDefaultStillWarns holds the reason --default exists. An
// actor added without it leaves the store resolving to whoever heads the roster,
// and the warning's own advice is to set defaults.actor, which would send the
// caller back to the editor this command replaces.
func TestActorAddWithoutDefaultStillWarns(t *testing.T) {
	dir := emptyRosterStore(t)

	if got := runCLI(t, dir, nil, "actor", "add", "human:you"); got.code != exitOK {
		t.Fatalf("actor add: %s", got.stderr)
	}
	got := runCLI(t, dir, nil, "create", "--title", "unwarned?")
	if got.code != exitOK {
		t.Fatalf("create: %s%s", got.stdout, got.stderr)
	}
	if !strings.Contains(got.stderr, "only the first actor") {
		t.Errorf("an incidental default did not warn: %q", got.stderr)
	}
}

// TestActorAddIsANoOpWhenNothingChanges follows SeriesResult.Changed: running it
// to be sure is not an error, so a caller does not special-case success.
func TestActorAddIsANoOpWhenNothingChanges(t *testing.T) {
	dir := emptyRosterStore(t)
	runCLI(t, dir, nil, "actor", "add", "human:you", "--name", "You", "--default")

	env := decode(t, runCLI(t, dir, nil, "--json", "actor", "add", "human:you", "--name", "You", "--default").stdout)
	if env["kind"] != "actor" {
		t.Fatalf("kind = %v, want actor", env["kind"])
	}
	if env["changed"] != false {
		t.Errorf("a repeat add reported changed = %v", env["changed"])
	}
	paths, _ := env["pathsChanged"].([]any)
	if len(paths) != 0 {
		t.Errorf("a repeat add named changed paths: %v", paths)
	}
}

// TestActorAddKeepsANameWhenOnlySettingTheDefault stops the second run of a
// two-step setup from quietly clearing a display name.
func TestActorAddKeepsANameWhenOnlySettingTheDefault(t *testing.T) {
	dir := emptyRosterStore(t)
	runCLI(t, dir, nil, "actor", "add", "human:you", "--name", "You")

	if got := runCLI(t, dir, nil, "actor", "add", "human:you", "--default"); got.code != exitOK {
		t.Fatalf("actor add --default: %s", got.stderr)
	}
	env := decode(t, runCLI(t, dir, nil, "--json", "actor").stdout)
	actors, _ := env["actors"].([]any)
	if len(actors) != 1 {
		t.Fatalf("actors = %v", actors)
	}
	first, _ := actors[0].(map[string]any)
	if first["name"] != "You" {
		t.Errorf("the display name was lost when only the default was set: %v", first)
	}
	if env["default"] != "human:you" {
		t.Errorf("default = %v, want human:you", env["default"])
	}
}

// TestActorRefusesAMalformedInvocation keeps the usage errors pointing at the
// form that works, since this is a command people meet while already stuck.
func TestActorRefusesAMalformedInvocation(t *testing.T) {
	dir := newStore(t)

	for _, args := range [][]string{
		{"actor", "add"},
		{"actor", "remove", "human:sothr"},
		{"actor", "--name", "You"},
	} {
		got := runCLI(t, dir, nil, args...)
		if got.code == exitOK {
			t.Errorf("%v was accepted: %s", args, got.stdout)
		}
		if !strings.Contains(got.stderr, "actor") {
			t.Errorf("%v: the refusal does not name the command: %q", args, got.stderr)
		}
	}
}
