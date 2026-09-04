package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ticketPath finds the file a ticket occupies, wherever its status put it.
// Asking the store rather than assuming draft/ keeps these tests off the
// status-to-directory mapping, which is section 4's business.
func ticketPath(t *testing.T, dir, id string) string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, ".tickets", "*", id+".md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("want one file for %s, found %v (%v)", id, matches, err)
	}
	return matches[0]
}

// TestRemoveReportsWhatItDeleted covers the human form. A removal names the
// ticket and the path, because after this runs there is nothing left to show.
func TestRemoveReportsWhatItDeleted(t *testing.T) {
	dir := newStore(t)
	id := makeTicket(t, dir, "Filed against the wrong repository")
	path := ticketPath(t, dir, id)

	got := runCLI(t, dir, nil, "remove", id, "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("remove exited %d: %s%s", got.code, got.stdout, got.stderr)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("the file survived: %v", err)
	}
	for _, want := range []string{id, "Filed against the wrong repository", "removed"} {
		if !strings.Contains(got.stdout, want) {
			t.Errorf("stdout does not carry %q:\n%s", want, got.stdout)
		}
	}
	if !strings.Contains(got.stdout, id+".md") {
		t.Errorf("stdout does not name the path it emptied:\n%s", got.stdout)
	}
}

// TestRemoveEmitsAMutationResult holds the envelope to plan section 10, which
// settles remove as a write reporting like any other.
func TestRemoveEmitsAMutationResult(t *testing.T) {
	dir := newStore(t)
	id := makeTicket(t, dir, "Filed by mistake")

	got := runCLI(t, dir, nil, "--json", "remove", id, "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("remove exited %d: %s%s", got.code, got.stdout, got.stderr)
	}
	envelope := decode(t, got.stdout)
	if envelope["kind"] != "mutation-result" {
		t.Errorf("kind = %v, want mutation-result", envelope["kind"])
	}

	// The stub carries the id and the revision the ticket had. 10 keeps it to
	// two keys even here, where show afterwards is ticket_not_found, because
	// adding a key later is additive and removing one is a break.
	tk, ok := envelope["ticket"].(map[string]any)
	if !ok {
		t.Fatalf("no ticket stub in %v", envelope)
	}
	if tk["id"] != id {
		t.Errorf("id = %v, want %s", tk["id"], id)
	}
	if rev, _ := tk["revision"].(string); !strings.HasPrefix(rev, "sha256:") {
		t.Errorf("revision = %v, want the revision it had", tk["revision"])
	}
	paths, _ := envelope["pathsChanged"].([]any)
	if len(paths) != 1 {
		t.Fatalf("pathsChanged = %v, want the one path it no longer occupies", envelope["pathsChanged"])
	}
	if p, _ := paths[0].(string); !strings.HasSuffix(p, id+".md") {
		t.Errorf("pathsChanged = %v, want the removed ticket's file", paths)
	}
}

// TestRemoveRefusalsReportTheirCode covers both rows of the 9.1 table through
// the CLI, in both output forms. A refusal has to exit non-zero and leave the
// file, or a script reads a failure as a success.
func TestRemoveRefusalsReportTheirCode(t *testing.T) {
	cases := []struct {
		name  string
		code  string
		setup func(t *testing.T, dir, id string)
		// detail is the key this refusal puts in the envelope's details, or
		// empty when it carries none.
		detail string
	}{
		{"referenced", "ticket_referenced", func(t *testing.T, dir, id string) {
			other := makeTicket(t, dir, "The ticket waiting on it")
			if got := runCLI(t, dir, nil, "link", other, "--depends-on", id,
				"--actor", "human:sothr"); got.code != exitOK {
				t.Fatalf("link: %s%s", got.stdout, got.stderr)
			}
		}, "referencedBy"},
		{"touched", "ticket_touched", func(t *testing.T, dir, id string) {
			if got := runCLI(t, dir, nil, "note", id, "what I found",
				"--actor", "human:sothr"); got.code != exitOK {
				t.Fatalf("note: %s%s", got.stdout, got.stderr)
			}
		}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := newStore(t)
			id := makeTicket(t, dir, "The ticket somebody will try to remove")
			c.setup(t, dir, id)
			path := ticketPath(t, dir, id)

			got := runCLI(t, dir, nil, "remove", id, "--actor", "human:sothr")
			if got.code != exitError {
				t.Fatalf("remove exited %d, want %d: %s%s", got.code, exitError, got.stdout, got.stderr)
			}
			if _, err := os.Stat(path); err != nil {
				t.Errorf("the refusal deleted the file anyway: %v", err)
			}
			if !strings.Contains(got.stderr, c.code) {
				t.Errorf("stderr does not carry %s:\n%s", c.code, got.stderr)
			}

			// The same refusal has to reach a machine, since the code is the
			// part a caller switches on.
			j := runCLI(t, dir, nil, "--json", "remove", id, "--actor", "human:sothr")
			if j.code != exitError {
				t.Fatalf("--json remove exited %d, want %d", j.code, exitError)
			}
			envelope := decode(t, j.stdout)
			if envelope["kind"] != "error" {
				t.Errorf("kind = %v, want error", envelope["kind"])
			}
			body, ok := envelope["error"].(map[string]any)
			if !ok {
				t.Fatalf("no error body in %v", envelope)
			}
			if body["code"] != c.code {
				t.Errorf("code = %v, want %s", body["code"], c.code)
			}
			if c.detail != "" {
				// The referrers reach a machine through details, because the
				// message naming them is prose and may change.
				details, _ := body["details"].(map[string]any)
				if s, _ := details[c.detail].(string); s == "" {
					t.Errorf("details carries no %s: %v", c.detail, body["details"])
				}
			}
		})
	}
}

// TestRemoveForceWarnsWithoutSpoilingTheEnvelope is the constraint that makes
// the warning safe to emit. A consumer parsing stdout must not have to strip
// prose out of it first, so the dangling references go to stderr.
func TestRemoveForceWarnsWithoutSpoilingTheEnvelope(t *testing.T) {
	dir := newStore(t)
	id := makeTicket(t, dir, "The dependency")
	other := makeTicket(t, dir, "The ticket waiting on it")
	if got := runCLI(t, dir, nil, "link", other, "--depends-on", id,
		"--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("link: %s%s", got.stdout, got.stderr)
	}

	got := runCLI(t, dir, nil, "--json", "remove", id, "--force", "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("--force exited %d: %s%s", got.code, got.stdout, got.stderr)
	}

	// stdout is the envelope and nothing else.
	var envelope map[string]any
	if err := json.Unmarshal([]byte(got.stdout), &envelope); err != nil {
		t.Fatalf("the warning got into stdout: %v\n%s", err, got.stdout)
	}
	if envelope["kind"] != "mutation-result" {
		t.Errorf("kind = %v, want mutation-result", envelope["kind"])
	}

	// The report is what --force buys back for the refusal it overrode, so it
	// has to name the referrer and the repair.
	for _, want := range []string{other, "The ticket waiting on it", "dependencies", "unlink"} {
		if !strings.Contains(got.stderr, want) {
			t.Errorf("the warning does not carry %q:\n%s", want, got.stderr)
		}
	}
}

// TestRemoveForceNamesTheRepairForAParent covers the half unlink cannot fix.
// unlink drops a dependency and does nothing to a parent, so offering it here
// would send a reader to a command that reports success and changes nothing.
func TestRemoveForceNamesTheRepairForAParent(t *testing.T) {
	dir := newStore(t)
	epic := makeTicket(t, dir, "The epic", "--type", "epic")
	child := makeTicket(t, dir, "A child of the epic")
	if got := runCLI(t, dir, nil, "update", child, "--parent", epic,
		"--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("update --parent: %s%s", got.stdout, got.stderr)
	}

	got := runCLI(t, dir, nil, "remove", epic, "--force", "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("--force exited %d: %s%s", got.code, got.stdout, got.stderr)
	}
	if !strings.Contains(got.stderr, "--parent") {
		t.Errorf("the warning does not name the repair for a parent:\n%s", got.stderr)
	}
	if strings.Contains(got.stderr, "--depends-on") {
		t.Errorf("the warning offers unlink for a parent reference:\n%s", got.stderr)
	}
}

// TestRemoveHonoursIfRevision holds the destructive command to the precondition
// every other write gives. This is the one command where losing that race
// cannot be undone from the store itself.
func TestRemoveHonoursIfRevision(t *testing.T) {
	dir := newStore(t)
	id := makeTicket(t, dir, "Changed under the reader")
	show := decode(t, runCLI(t, dir, nil, "--json", "show", id).stdout)
	tk, _ := show["ticket"].(map[string]any)
	stale, _ := tk["revision"].(string)
	if stale == "" {
		t.Fatalf("no revision in %v", show)
	}
	if got := runCLI(t, dir, nil, "update", id, "--description", "somebody else edited this",
		"--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("update: %s%s", got.stdout, got.stderr)
	}
	path := ticketPath(t, dir, id)

	got := runCLI(t, dir, nil, "--if-revision", stale, "remove", id, "--actor", "human:sothr")
	if got.code != exitError {
		t.Fatalf("a stale removal exited %d, want %d", got.code, exitError)
	}
	if !strings.Contains(got.stderr, "stale_revision") {
		t.Errorf("stderr does not carry stale_revision:\n%s", got.stderr)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("the stale removal deleted the file anyway: %v", err)
	}
}

// TestRemoveTakesItsFlagsInEitherOrder covers the parse rule in 12.1. The
// standard library stops at the first non-flag word, so a naive parse never
// sees a flag written after the ID.
func TestRemoveTakesItsFlagsInEitherOrder(t *testing.T) {
	for _, args := range [][]string{
		{"remove", "ID", "--force"},
		{"remove", "--force", "ID"},
	} {
		dir := newStore(t)
		id := makeTicket(t, dir, "Removed whichever way the flag is written")
		call := make([]string, len(args))
		for i, a := range args {
			if a == "ID" {
				a = id
			}
			call[i] = a
		}
		call = append(call, "--actor", "human:sothr")

		got := runCLI(t, dir, nil, call...)
		if got.code != exitOK {
			t.Errorf("%v exited %d: %s%s", call, got.code, got.stdout, got.stderr)
		}
	}
}

// TestRemoveResolvesAPrefix holds remove to 5.5. A destructive command that
// demanded all 26 characters would be the one place a person pastes an ID by
// hand, which is where they get it wrong.
func TestRemoveResolvesAPrefix(t *testing.T) {
	dir := newStore(t)
	id := makeTicket(t, dir, "Removed by prefix")

	got := runCLI(t, dir, nil, "remove", id[:12], "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("a prefix was refused: %s%s", got.stdout, got.stderr)
	}
	if !strings.Contains(got.stdout, id) {
		t.Errorf("stdout does not name the ticket it resolved to:\n%s", got.stdout)
	}
}

// TestRemoveTakesOneID keeps the usage error where the other commands put it.
func TestRemoveTakesOneID(t *testing.T) {
	dir := newStore(t)
	for _, args := range [][]string{{"remove"}, {"remove", "A", "B"}} {
		if got := runCLI(t, dir, nil, args...); got.code == exitOK {
			t.Errorf("%v succeeded, want a usage error", args)
		}
	}
}
