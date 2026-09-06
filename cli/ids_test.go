package cli

import (
	"strings"
	"testing"
)

// idColumn returns the first whitespace-separated field of each printed row,
// which is where a listing puts the ID.
func idColumn(t *testing.T, out string) []string {
	t.Helper()
	var ids []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		ids = append(ids, fields[0])
	}
	return ids
}

// TestIDsFullPrintsTheWholeID is the mode with nothing to compute. It is worth
// its own test because it is the one a person reaches for when they are about
// to paste an ID somewhere this store cannot be consulted.
func TestIDsFullPrintsTheWholeID(t *testing.T) {
	dir := newStore(t)
	first := makeTicket(t, dir, "The first ticket")
	makeTicket(t, dir, "The second ticket")

	full := runCLI(t, dir, nil, "list", "--ids", "full")
	if full.code != exitOK {
		t.Fatalf("list --ids full exited %d: %s", full.code, full.stderr)
	}
	for _, id := range idColumn(t, full.stdout) {
		if len(id) != len(first) {
			t.Errorf("--ids full printed %q, want the whole %d-character ID", id, len(first))
		}
	}

	// The default is shorter, which is what makes full worth asking for.
	short := runCLI(t, dir, nil, "list")
	for _, id := range idColumn(t, short.stdout) {
		if len(id) >= len(first) {
			t.Errorf("the default printed %q, which is not abbreviated at all", id)
		}
	}
}

// TestIDsRefusesAnUnknownMode. The flag takes a value, so a typo has to fail
// rather than silently selecting the default.
func TestIDsRefusesAnUnknownMode(t *testing.T) {
	dir := newStore(t)
	makeTicket(t, dir, "Something to list")

	got := runCLI(t, dir, nil, "--json", "list", "--ids", "short")
	if got.code != exitError {
		t.Fatalf("an unknown --ids value should be refused, got exit %d", got.code)
	}
	if code := errCode(t, got); code != codeUsage {
		t.Errorf("code = %v, want %s", code, codeUsage)
	}
	// The message names the values that do work, because a reader who typed
	// the wrong one is asking exactly that.
	for _, mode := range idModes {
		if !strings.Contains(got.stderr, mode) {
			t.Errorf("the refusal does not name %q: %s", mode, got.stderr)
		}
	}
}

// TestEveryListingTakesIDs holds the six listing commands to one vocabulary. A
// flag that works on list and not on ready is worse than no flag, because the
// reader learns it and then it fails somewhere they cannot predict.
func TestEveryListingTakesIDs(t *testing.T) {
	dir := newStore(t)
	id := makeTicket(t, dir, "Referenced from everywhere")
	// files and refs answer from what a ticket recorded, so the ticket has to
	// record something before either has a row to print.
	if got := runCLI(t, dir, nil, "link", id,
		"--ref", "proposal:git-ticket", "--path", "src/main.go",
		"--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("link: %s%s", got.stdout, got.stderr)
	}

	for _, c := range []struct {
		name string
		args []string
	}{
		{"list", []string{"list"}},
		{"ready", []string{"ready"}},
		{"search", []string{"search", "Referenced"}},
		{"deps", []string{"deps", id}},
		{"files", []string{"files", "src/main.go"}},
		{"refs", []string{"refs", "proposal:git-ticket"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := runCLI(t, dir, nil, append(c.args, "--ids", "full")...)
			if got.code != exitOK {
				t.Fatalf("%s --ids full exited %d: %s", c.name, got.code, got.stderr)
			}
			// An unknown value has to be refused here too, or the flag is
			// only half wired: registered but never validated.
			bad := runCLI(t, dir, nil, append(c.args, "--ids", "nope")...)
			if bad.code != exitError {
				t.Errorf("%s accepted --ids nope, exit %d", c.name, bad.code)
			}
		})
	}
}

// TestIDsLeavesTheJSONAlone. --ids shapes what a person reads. The envelope is
// parsed rather than read, and an abbreviated ID in a published contract is one
// a later ticket can make ambiguous, so it always carries the whole thing.
func TestIDsLeavesTheJSONAlone(t *testing.T) {
	dir := newStore(t)
	want := makeTicket(t, dir, "The only ticket")

	for _, mode := range idModes {
		got := runCLI(t, dir, nil, "--json", "list", "--ids", mode)
		if got.code != exitOK {
			t.Fatalf("--json list --ids %s exited %d: %s", mode, got.code, got.stderr)
		}
		envelope := decode(t, got.stdout)
		tickets, _ := envelope["tickets"].([]any)
		if len(tickets) != 1 {
			t.Fatalf("--ids %s: %d tickets, want 1", mode, len(tickets))
		}
		row, _ := tickets[0].(map[string]any)
		if row["id"] != want {
			t.Errorf("--ids %s: id = %v, want the whole %s", mode, row["id"], want)
		}
	}
}
