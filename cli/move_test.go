package cli

import (
	"strings"
	"testing"
)

func TestMoveCommandsGateAndResolveALocalDependent(t *testing.T) {
	dir := newStore(t)
	source := makeTicket(t, dir, "Foreign prerequisite")
	dependent := makeTicket(t, dir, "Local dependent")
	for _, args := range [][]string{
		{"status", source, "done", "--reason", "finished here", "--actor", "human:sothr"},
		{"link", dependent, "--depends-on", source, "--actor", "human:sothr"},
		{"status", dependent, "ready", "--actor", "human:sothr"},
	} {
		got := runCLI(t, dir, nil, args...)
		if got.code != exitOK {
			t.Fatalf("%v: %s", args, got.stderr)
		}
	}
	if got := runCLI(t, dir, nil, "ready"); !strings.Contains(got.stdout, "Local dependent") {
		t.Fatalf("dependent should be ready before move: %s", got.stdout)
	}
	move := runCLI(t, dir, nil, "move", source, "--to-ref", "ledger-ticket:foreign-id", "--reason", "transferred", "--actor", "human:sothr")
	if move.code != exitOK || !strings.Contains(move.stdout, dependent+"  Local dependent") {
		t.Fatalf("move must name the affected dependent and title: %s%s", move.stdout, move.stderr)
	}
	if got := runCLI(t, dir, nil, "ready"); strings.Contains(got.stdout, "Local dependent") {
		t.Fatalf("moved done prerequisite released dependent: %s", got.stdout)
	}
	check := runCLI(t, dir, nil, "--json", "check")
	if !strings.Contains(check.stdout, "dependency_moved") {
		t.Fatalf("check omitted moved edge: %s", check.stdout)
	}
	revision := showTicket(t, dir, source)["revision"].(string)
	resolve := runCLI(t, dir, nil, "resolve-move", dependent, "--from", source,
		"--if-source-revision", revision, "--reason", "foreign completion verified", "--actor", "human:sothr")
	if resolve.code != exitOK {
		t.Fatalf("resolve: %s%s", resolve.stdout, resolve.stderr)
	}
	if got := runCLI(t, dir, nil, "ready"); !strings.Contains(got.stdout, "Local dependent") {
		t.Fatalf("resolved dependent should be ready: %s", got.stdout)
	}
	if got := runCLI(t, dir, nil, "check", "--strict"); got.code != exitOK {
		t.Fatalf("resolved store should pass check: %s%s", got.stdout, got.stderr)
	}
}
